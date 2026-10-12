package exaroton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Websocket keepalive and limits.
const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 30 * time.Second
	wsReadLimit  = 1 << 20 // 1 MiB per message
	// defaultSendTimeout bounds Send when the caller's context has no
	// deadline, so Send(context.Background()) cannot block forever waiting
	// for the ready signal.
	defaultSendTimeout = 30 * time.Second
)

// Session represents a websocket session to the server.
//
// Delivery semantics: all channels are buffered and sends are non-blocking.
// If a consumer falls behind, messages are dropped (drop-on-full) rather than
// blocking the read loop. There is no automatic reconnect: when the connection
// closes, channels are closed and the caller must call OpenWebSocket again.
type Session struct {
	conn *websocket.Conn

	StatusChan             chan *Server
	ConsoleChan            chan string
	TickChan               chan float64
	StatsChan              chan *Stats
	HeapChan               chan int64
	ManagementResponseChan chan *ManagementResponse
	ManagementNotifyChan   chan *ManagementNotification
	ErrorChan              chan error

	done      chan struct{}
	closeOnce sync.Once
	writeMu   sync.Mutex

	readyChan chan struct{}
	readyOnce sync.Once

	// sendTimeout bounds Send when the caller's context carries no deadline.
	sendTimeout time.Duration

	// Do not touch this without locking the mutex
	streamMu      sync.RWMutex
	streamStarted map[string]chan struct{}

	// Do not touch this without locking the mutex
	pendingMu       sync.Mutex
	pendingRequests map[string]chan *ManagementResponse

	// server anchors REST version checks for gated streams (tick 1.16+,
	// management 1.21.9+). version caches Software.Version after the first
	// lookup. Access version fields with versionMu.
	server    *ServerRef
	versionMu sync.Mutex
	version   string
	versionOK bool
}

type wsEnvelope struct {
	Stream string          `json:"stream"`
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type wsError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *wsError) Error() string {
	return fmt.Sprintf("api error: %s", e.Message)
}

type wsTickPayload struct {
	Tick float64 `json:"averageTickTime"`
}

type wsHeapPayload struct {
	Usage int64 `json:"usage"`
}

// OpenWebSocket opens a new websocket session to the server.
func (s *ServerRef) OpenWebSocket(ctx context.Context) (*Session, error) {
	wsURL, err := s.websocketURL()
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+s.client.token)
	headers.Add("User-Agent", s.client.userAgentOrDefault())

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return nil, err
	}

	conn.SetReadLimit(wsReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	session := &Session{
		conn:                   conn,
		StatusChan:             make(chan *Server, 10),
		ConsoleChan:            make(chan string, 150),
		TickChan:               make(chan float64, 10),
		StatsChan:              make(chan *Stats, 10),
		HeapChan:               make(chan int64, 10),
		ManagementResponseChan: make(chan *ManagementResponse, 10),
		ManagementNotifyChan:   make(chan *ManagementNotification, 10),
		ErrorChan:              make(chan error, 10),
		done:                   make(chan struct{}),
		readyChan:              make(chan struct{}),
		sendTimeout:            defaultSendTimeout,
		streamStarted:          make(map[string]chan struct{}),
		pendingRequests:        make(map[string]chan *ManagementResponse),
		server:                 s,
	}

	go session.listen()
	go session.pingLoop()

	return session, nil
}

// websocketURL derives the websocket endpoint from the client's baseURL,
// so custom baseURLs (tests, proxies) work. https->wss, http->ws.
func (s *ServerRef) websocketURL() (string, error) {
	base := s.client.baseURL
	if base == "" {
		base = "https://api.exaroton.com/v1/"
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid baseURL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", fmt.Errorf("unsupported baseURL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/servers/" + url.PathEscape(s.serverID) + "/websocket"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// serverSoftwareVersion returns the cached server software version
// (e.g. "1.21.9"), fetching it via REST once. The second return value is
// false when the version is unknown (no server ref, fetch error, or empty
// version); callers fail open in that case.
func (s *Session) serverSoftwareVersion(ctx context.Context) (string, bool) {
	s.versionMu.Lock()
	if s.versionOK {
		v := s.version
		s.versionMu.Unlock()
		return v, v != ""
	}
	s.versionMu.Unlock()

	if s.server == nil {
		return "", false
	}
	info, err := s.server.GetServer(ctx)
	if err != nil || info == nil {
		return "", false
	}
	v := info.Software.Version

	s.versionMu.Lock()
	s.version = v
	s.versionOK = true
	s.versionMu.Unlock()
	return v, v != ""
}

// requireMinVersion ensures the server software is at least major.minor.patch
// (e.g. tick needs 1.16.0, management needs 1.21.9). It fails open when the
// version is unknown so REST outages or unparseable versions never block a
// stream the server might support.
func (s *Session) requireMinVersion(ctx context.Context, major, minor, patch int, stream string) error {
	v, ok := s.serverSoftwareVersion(ctx)
	if !ok {
		return nil
	}
	gotMajor, gotMinor, gotPatch, ok := parseMinecraftVersion(v)
	if !ok {
		return nil
	}
	if gotMajor != major {
		if gotMajor < major {
			return fmt.Errorf("%s stream requires Minecraft %d.%d.%d+, server is %s", stream, major, minor, patch, v)
		}
		return nil
	}
	if gotMinor != minor {
		if gotMinor < minor {
			return fmt.Errorf("%s stream requires Minecraft %d.%d.%d+, server is %s", stream, major, minor, patch, v)
		}
		return nil
	}
	if gotPatch < patch {
		return fmt.Errorf("%s stream requires Minecraft %d.%d.%d+, server is %s", stream, major, minor, patch, v)
	}
	return nil
}

// parseMinecraftVersion parses leading "major.minor[.patch]" numbers from a
// software version string, tolerating suffixes like "-pre1" or "+build".
func parseMinecraftVersion(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, 0, 0, false
	}
	if i := strings.IndexAny(v, "-+_ "); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return 0, 0, 0, false
	}
	var nums []int
	for _, p := range parts {
		if p == "" {
			return 0, 0, 0, false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, 0, 0, false
		}
		nums = append(nums, n)
		if len(nums) == 3 {
			break
		}
	}
	for len(nums) < 3 {
		nums = append(nums, 0)
	}
	return nums[0], nums[1], nums[2], true
}

func (s *Session) listen() {
	defer func() {
		s.closeOnce.Do(func() {
			close(s.done)
		})
		s.conn.Close()
		s.closeChannels()
	}()

	for {
		select {
		case <-s.done:
			return
		default:
		}

		var msg wsEnvelope
		err := s.conn.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				select {
				case <-s.done:
				default:
					s.sendError(err)
				}
			}
			return
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(wsPongWait))

		s.routeMessage(msg)
	}
}

func (s *Session) pingLoop() {
	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.writeMu.Lock()
			_ = s.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			err := s.conn.WriteMessage(websocket.PingMessage, nil)
			s.writeMu.Unlock()
			if err != nil {
				s.sendError(err)
				return
			}
		}
	}
}

func (s *Session) closeChannels() {
	// Called exactly once from listen() exit; routeMessage (same goroutine)
	// sends no more after this. Each close is guarded so one double-close
	// cannot skip the rest.
	safeClose := func(closeFn func()) {
		defer func() { _ = recover() }()
		closeFn()
	}
	safeClose(func() { close(s.StatusChan) })
	safeClose(func() { close(s.ConsoleChan) })
	safeClose(func() { close(s.TickChan) })
	safeClose(func() { close(s.StatsChan) })
	safeClose(func() { close(s.HeapChan) })
	safeClose(func() { close(s.ManagementResponseChan) })
	safeClose(func() { close(s.ManagementNotifyChan) })
	safeClose(func() { close(s.ErrorChan) })
}

func (s *Session) routeMessage(msg wsEnvelope) {
	switch msg.Type {
	case "ready":
		s.readyOnce.Do(func() {
			close(s.readyChan)
		})
		return
	case "started":
		s.streamMu.Lock()
		if s.streamStarted == nil {
			s.streamStarted = make(map[string]chan struct{})
		}

		if ch, ok := s.streamStarted[msg.Stream]; ok {
			select {
			case <-ch:
			default:
				close(ch)
			}
		} else {
			// We received "started" before anyone called WaitForStream.
			// Create a channel and immediately close it so future waiters don't block.
			ch := make(chan struct{})
			close(ch)
			s.streamStarted[msg.Stream] = ch
		}
		s.streamMu.Unlock()
	case "connected", "keep-alive":
		return
	case "disconnected":
		reason := string(msg.Data)
		s.sendError(fmt.Errorf("connection disconnected: %s", reason))
		s.Close()
		return
	}

	switch msg.Stream {
	case "status":
		if msg.Type == "status" {
			var server Server
			if json.Unmarshal(msg.Data, &server) == nil {
				select {
				case s.StatusChan <- &server:
				default:
				}
			}
		}

	case "console":
		if msg.Type == "line" {
			var line string
			if json.Unmarshal(msg.Data, &line) == nil {

				select {
				case s.ConsoleChan <- line:
				default:
				}
			}
		}

	case "tick":
		if msg.Type == "tick" {
			var tick wsTickPayload
			if json.Unmarshal(msg.Data, &tick) == nil {
				select {
				case s.TickChan <- tick.Tick:
				default:
				}
			}
		}

	case "stats":
		if msg.Type == "stats" {
			var stats Stats
			if json.Unmarshal(msg.Data, &stats) == nil {
				select {
				case s.StatsChan <- &stats:
				default:
				}
			}
		}

	case "heap":
		if msg.Type != "heap" {
			return
		}
		var heap wsHeapPayload
		if json.Unmarshal(msg.Data, &heap) == nil {
			select {
			case s.HeapChan <- heap.Usage:
			default:
			}
		}
	case "management":
		switch msg.Type {
		case "response":
			var response ManagementResponse
			if json.Unmarshal(msg.Data, &response) != nil {
				return
			}

			s.pendingMu.Lock()
			ch, found := s.pendingRequests[response.ID]
			s.pendingMu.Unlock()

			if found {
				select {
				case ch <- &response:
				default:
				}
			} else {
				select {
				case s.ManagementResponseChan <- &response:
				default:
				}
			}
		case "notification":
			var notification ManagementNotification
			if json.Unmarshal(msg.Data, &notification) == nil {
				select {
				case s.ManagementNotifyChan <- &notification:
				default:
				}
			}
		}
	}
}

func (s *Session) markStreamStopped(streamName string) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	if s.streamStarted != nil {
		delete(s.streamStarted, streamName)
	}
}

func (s *Session) waitForStream(ctx context.Context, channel string) error {
	// RLock allows multiple goroutines to check status simultaneously.
	s.streamMu.RLock()
	var ch chan struct{}
	var ok bool
	if s.streamStarted != nil {
		ch, ok = s.streamStarted[channel]
	}
	s.streamMu.RUnlock() // Release read lock immediately

	// If the channel doesn't exist yet, we need to acquire a WRITE lock to create it.
	if !ok {
		s.streamMu.Lock()
		// Re-check inside the lock in case someone else created it while we were waiting for the lock
		if s.streamStarted == nil {
			s.streamStarted = make(map[string]chan struct{})
		}

		if ch, ok = s.streamStarted[channel]; !ok {
			ch = make(chan struct{})
			s.streamStarted[channel] = ch
		}
		s.streamMu.Unlock()
	}

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for stream %s: %w", channel, ctx.Err())
	case <-s.done:
		return fmt.Errorf("session closed while waiting for stream %s (the server might be offline)", channel)
	}
}

// Send writes a message after the ready signal. Management requests wait for their stream; writes have a deadline.
// When ctx carries no deadline, a session-level timeout (30s by default)
// applies so the ready/stream waits cannot block forever.
func (s *Session) Send(ctx context.Context, v any) error {
	if _, ok := ctx.Deadline(); !ok {
		timeout := s.sendTimeout
		if timeout <= 0 {
			timeout = defaultSendTimeout
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case <-s.readyChan:
	case <-s.done:
		return fmt.Errorf("cannot send message: session is closed (the server might be offline)")
	case <-ctx.Done():
		return fmt.Errorf("cannot send message: %w", ctx.Err())
	}

	if env, ok := v.(wsEnvelope); ok && env.Stream != "" {
		if env.Type != "start" && env.Type != "stop" {
			if err := s.waitForStream(ctx, env.Stream); err != nil {
				return err
			}
		}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	_ = s.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return s.conn.WriteJSON(v)
}

func (s *Session) sendError(err error) {
	// Sending on a closed channel panics, even inside select/default, and
	// ErrorChan is closed by closeChannels while pingLoop may still send.
	// Drop when closed; recover as a backstop for the check-then-send race.
	defer func() { _ = recover() }()
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.ErrorChan <- err:
	default:
	}
}

// Close shuts the session down, sends a close frame, and closes all channels. It is idempotent.
func (s *Session) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.done)
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		_ = s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			closeErr = err
		}
		// Unblock ReadJSON so listen() can exit and close channels.
		_ = s.conn.Close()
	})
	return closeErr
}
