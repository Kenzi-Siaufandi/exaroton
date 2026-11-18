package exaroton

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockWSConn is the server side of a mock websocket connection.
type mockWSConn struct {
	conn *websocket.Conn
	req  *http.Request
}

// mockWSServer serves both the REST server lookup used by version-gated
// streams and the websocket endpoint. It never contacts the real API.
type mockWSServer struct {
	srv   *httptest.Server
	conns chan *mockWSConn
	done  chan struct{}
	once  sync.Once
}

func newMockWSServer(t *testing.T) *mockWSServer {
	t.Helper()

	m := &mockWSServer{
		conns: make(chan *mockWSConn, 1),
		done:  make(chan struct{}),
	}
	upgrader := websocket.Upgrader{}

	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/websocket") {
			writeMockServerInfo(w)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := conn.WriteJSON(map[string]any{"type": "ready"}); err != nil {
			return
		}

		select {
		case m.conns <- &mockWSConn{conn: conn, req: r}:
		case <-m.done:
			return
		}

		<-m.done
	}))

	t.Cleanup(m.close)
	return m
}

func writeMockServerInfo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"error":   nil,
		"data": map[string]any{
			"id":      testServerID,
			"name":    "example",
			"address": "example.exaroton.me",
			"motd":    "hi",
			"status":  1,
			"players": map[string]any{"max": 20, "count": 0, "list": []string{}},
			"software": map[string]any{
				"id":      testSoftwareID,
				"name":    "Vanilla",
				"version": "1.21.9",
			},
		},
	})
}

func (m *mockWSServer) close() {
	m.once.Do(func() {
		close(m.done)
		m.srv.Close()
	})
}

func (m *mockWSServer) client(t *testing.T, opts ...Option) *Client {
	t.Helper()

	options := make([]Option, 0, len(opts)+1)
	options = append(options, WithBaseURL(m.srv.URL+"/"))
	options = append(options, opts...)

	client, err := NewClient("tok", options...)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func (m *mockWSServer) accept(t *testing.T) *mockWSConn {
	t.Helper()

	select {
	case conn := <-m.conns:
		return conn
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func openMockSession(t *testing.T, m *mockWSServer) (*Session, *mockWSConn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	ws, err := m.client(t).Server(testServerID).OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ws.Close() })

	return ws, m.accept(t)
}

func readWSEnvelope(t *testing.T, conn *websocket.Conn) wsEnvelope {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var env wsEnvelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("read websocket envelope: %v", err)
	}
	return env
}

func writeWSEnvelope(t *testing.T, conn *websocket.Conn, value any) {
	t.Helper()

	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := conn.WriteJSON(value); err != nil {
		t.Fatalf("write websocket envelope: %v", err)
	}
}

func wsTestEnvelope(t *testing.T, stream, typ string, data any) wsEnvelope {
	t.Helper()

	env := wsEnvelope{Stream: stream, Type: typ}
	if data == nil {
		return env
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	env.Data = raw
	return env
}

func waitForWSChan[T any](t *testing.T, name string, ch <-chan T) T {
	t.Helper()

	select {
	case value, ok := <-ch:
		if !ok {
			t.Fatalf("%s channel closed before a message", name)
		}
		return value
	case <-time.After(2 * time.Second):
		var zero T
		t.Fatalf("timed out waiting for %s", name)
		return zero
	}
}

func openManagementStream(t *testing.T, ws *Session, peer *mockWSConn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.SubscribeManagement(ctx); err != nil {
		t.Fatal(err)
	}

	start := readWSEnvelope(t, peer.conn)
	if start.Stream != "management" || start.Type != "start" {
		t.Fatalf("management start envelope = %+v", start)
	}
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "management", "started", nil))
}

type mockManagementRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func readManagementRequest(t *testing.T, conn *websocket.Conn) mockManagementRequest {
	t.Helper()

	env := readWSEnvelope(t, conn)
	if env.Stream != "management" || env.Type != "request" {
		t.Fatalf("management request envelope = %+v", env)
	}

	var req mockManagementRequest
	if err := json.Unmarshal(env.Data, &req); err != nil {
		t.Fatal(err)
	}
	if req.ID == "" {
		t.Fatal("management request has no ID")
	}
	return req
}

func writeManagementResponse(t *testing.T, conn *websocket.Conn, id string, data any) {
	t.Helper()

	writeWSEnvelope(t, conn, wsTestEnvelope(t, "management", "response", map[string]any{
		"id":   id,
		"data": data,
	}))
}

func TestWSHandshake(t *testing.T) {
	m := newMockWSServer(t)
	client := m.client(t, WithUserAgent("ws-test-agent"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := client.Server(testServerID).OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	peer := m.accept(t)
	if got, want := peer.req.URL.Path, "/servers/"+testServerID+"/websocket"; got != want {
		t.Fatalf("websocket path = %q, want %q", got, want)
	}
	if got, want := peer.req.Header.Get("Authorization"), "Bearer tok"; got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
	if got, want := peer.req.Header.Get("User-Agent"), "ws-test-agent"; got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
}

func TestWSStatus(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "status", "status", map[string]any{
		"id":      testServerID,
		"name":    "example",
		"address": "example.exaroton.me",
		"status":  1,
	}))

	status := waitForWSChan(t, "status", ws.StatusChan)
	if status.ID != testServerID {
		t.Fatalf("status ID = %q, want %q", status.ID, testServerID)
	}
	if status.Status != StatusOnline {
		t.Fatalf("status = %v, want %v", status.Status, StatusOnline)
	}
}

func TestWSConsole(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.SubscribeConsole(ctx, 3); err != nil {
		t.Fatal(err)
	}

	start := readWSEnvelope(t, peer.conn)
	if start.Stream != "console" || start.Type != "start" {
		t.Fatalf("console start envelope = %+v", start)
	}

	var payload struct {
		Tail int `json:"tail"`
	}
	if err := json.Unmarshal(start.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Tail != 3 {
		t.Fatalf("console tail = %d, want 3", payload.Tail)
	}

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "console", "started", nil))
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "console", "line", "hello"))
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "console", "line", "world"))

	if line := waitForWSChan(t, "console", ws.ConsoleChan); line != "hello" {
		t.Fatalf("console line = %q, want %q", line, "hello")
	}
	if line := waitForWSChan(t, "console", ws.ConsoleChan); line != "world" {
		t.Fatalf("console line = %q, want %q", line, "world")
	}
}

func TestWSTick(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.SubscribeTick(ctx); err != nil {
		t.Fatal(err)
	}

	start := readWSEnvelope(t, peer.conn)
	if start.Stream != "tick" || start.Type != "start" {
		t.Fatalf("tick start envelope = %+v", start)
	}

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "tick", "started", nil))
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "tick", "tick", map[string]any{
		"averageTickTime": 12.5,
	}))

	if tick := waitForWSChan(t, "tick", ws.TickChan); tick != 12.5 {
		t.Fatalf("tick = %v, want 12.5", tick)
	}
}

func TestWSStats(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.SubscribeStats(ctx); err != nil {
		t.Fatal(err)
	}

	start := readWSEnvelope(t, peer.conn)
	if start.Stream != "stats" || start.Type != "start" {
		t.Fatalf("stats start envelope = %+v", start)
	}

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "stats", "started", nil))
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "stats", "stats", map[string]any{
		"memory": map[string]any{"percent": 42.5, "usage": 1024},
		"cpu":    map[string]any{"percent": 12.5, "usage": 0.5, "limit": 4},
	}))

	stats := waitForWSChan(t, "stats", ws.StatsChan)
	if stats.Memory.Percent != 42.5 || stats.Memory.Usage != 1024 {
		t.Fatalf("memory stats = %+v", stats.Memory)
	}
	if stats.CPU.Percent != 12.5 || stats.CPU.Usage != 0.5 || stats.CPU.Limit != 4 {
		t.Fatalf("cpu stats = %+v", stats.CPU)
	}
}

func TestWSHeap(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.SubscribeHeap(ctx); err != nil {
		t.Fatal(err)
	}

	start := readWSEnvelope(t, peer.conn)
	if start.Stream != "heap" || start.Type != "start" {
		t.Fatalf("heap start envelope = %+v", start)
	}

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "heap", "started", nil))
	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "heap", "heap", map[string]any{
		"usage": 4096,
	}))

	if heap := waitForWSChan(t, "heap", ws.HeapChan); heap != 4096 {
		t.Fatalf("heap = %d, want 4096", heap)
	}
}

func TestWSManagementAllowlist(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)
	openManagementStream(t, ws, peer)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type result struct {
		players []Player
		err     error
	}

	getResult := make(chan result, 1)
	go func() {
		players, err := ws.GetAllowlist(ctx)
		getResult <- result{players: players, err: err}
	}()

	getReq := readManagementRequest(t, peer.conn)
	if got, want := getReq.Method, string(MethodAllowlistGet); got != want {
		t.Fatalf("get method = %q, want %q", got, want)
	}
	if got, want := string(getReq.Params), "[]"; got != want {
		t.Fatalf("get params = %s, want %s", got, want)
	}

	existing := []Player{{
		Name: "Jiternos",
		UUID: "981853f1-a3f4-4f5a-808a-1a0ff47a5110",
	}}
	writeManagementResponse(t, peer.conn, getReq.ID, existing)

	got := waitForResult(t, getResult)
	if got.err != nil {
		t.Fatal(got.err)
	}
	if len(got.players) != 1 || got.players[0].Name != existing[0].Name {
		t.Fatalf("allowlist = %+v, want %+v", got.players, existing)
	}

	added := []Player{{
		Name: "Alex",
		UUID: "11111111-2222-3333-4444-555555555555",
	}}
	addResult := make(chan result, 1)
	go func() {
		players, err := ws.AddAllowlist(ctx, added)
		addResult <- result{players: players, err: err}
	}()

	addReq := readManagementRequest(t, peer.conn)
	if got, want := addReq.Method, string(MethodAllowlistAdd); got != want {
		t.Fatalf("add method = %q, want %q", got, want)
	}

	var args []json.RawMessage
	if err := json.Unmarshal(addReq.Params, &args); err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 {
		t.Fatalf("add params have %d args, want 1", len(args))
	}
	var sentPlayers []Player
	if err := json.Unmarshal(args[0], &sentPlayers); err != nil {
		t.Fatal(err)
	}
	if len(sentPlayers) != 1 || sentPlayers[0].Name != added[0].Name {
		t.Fatalf("add params = %+v, want %+v", sentPlayers, added)
	}

	updated := append(append([]Player{}, existing...), added...)
	writeManagementResponse(t, peer.conn, addReq.ID, updated)

	got = waitForResult(t, addResult)
	if got.err != nil {
		t.Fatal(got.err)
	}
	if len(got.players) != 2 {
		t.Fatalf("updated allowlist = %+v, want 2 players", got.players)
	}
}

func waitForResult[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		var zero T
		t.Fatal("timed out waiting for result")
		return zero
	}
}

func TestWSManagementNotification(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)
	openManagementStream(t, ws, peer)

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "management", "notification", map[string]any{
		"name": string(NotificationPlayersJoined),
		"data": Player{
			Name: "Alex",
			UUID: "11111111-2222-3333-4444-555555555555",
		},
	}))

	notification := waitForWSChan(t, "management notification", ws.ManagementNotifyChan)
	if notification.Name != string(NotificationPlayersJoined) {
		t.Fatalf("notification name = %q, want %q", notification.Name, NotificationPlayersJoined)
	}

	var player Player
	if err := json.Unmarshal(notification.Data, &player); err != nil {
		t.Fatal(err)
	}
	if player.Name != "Alex" {
		t.Fatalf("notification player = %+v", player)
	}
}

func TestWSManagementError(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)
	openManagementStream(t, ws, peer)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		_, err := ws.GetAllowlist(ctx)
		result <- err
	}()

	req := readManagementRequest(t, peer.conn)
	writeManagementResponse(t, peer.conn, req.ID, map[string]any{
		"error": map[string]any{
			"code":    42,
			"message": "denied",
		},
	})

	err := waitForResult(t, result)
	if err == nil {
		t.Fatal("expected management error")
	}

	var wsErr *wsError
	if !errors.As(err, &wsErr) {
		t.Fatalf("expected *wsError, got %T: %v", err, err)
	}
	if wsErr.Code != 42 || wsErr.Message != "denied" {
		t.Fatalf("management error = %+v", wsErr)
	}
}

func TestWSDisconnect(t *testing.T) {
	m := newMockWSServer(t)
	ws, peer := openMockSession(t, m)

	writeWSEnvelope(t, peer.conn, wsTestEnvelope(t, "", "disconnected", "server stopped"))

	select {
	case err, ok := <-ws.ErrorChan:
		if !ok || err == nil {
			t.Fatal("expected disconnect error")
		}
		if !strings.Contains(err.Error(), "disconnected") {
			t.Fatalf("disconnect error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for disconnect error")
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ws.StatusChan:
			if ok {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := ws.Send(ctx, wsEnvelope{}); err == nil {
				t.Fatal("expected Send to fail on a closed session")
			}
			return
		case <-deadline:
			t.Fatal("session channels were not closed after disconnect")
		}
	}
}

func TestWSSendWaitsForStream(t *testing.T) {
	m := newMockWSServer(t)
	ws, _ := openMockSession(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := ws.AddAllowlist(ctx, []Player{{Name: "Alex"}})
	if err == nil {
		t.Fatal("expected stream timeout")
	}
	if !strings.Contains(err.Error(), "waiting for stream management") {
		t.Fatalf("error = %v, want a management stream timeout", err)
	}
}

func TestWebSocketURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{
			name:    "https",
			baseURL: "https://api.exaroton.com/v1/",
			want:    "wss://api.exaroton.com/v1/servers/" + testServerID + "/websocket",
		},
		{
			name:    "http",
			baseURL: "http://127.0.0.1:8080/",
			want:    "ws://127.0.0.1:8080/servers/" + testServerID + "/websocket",
		},
		{
			name:    "custom path",
			baseURL: "http://127.0.0.1:8080/api/v1/",
			want:    "ws://127.0.0.1:8080/api/v1/servers/" + testServerID + "/websocket",
		},
		{
			name:    "unsupported scheme",
			baseURL: "ftp://example.com/",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient("tok", WithBaseURL(tc.baseURL))
			if err != nil {
				t.Fatal(err)
			}

			got, err := client.Server(testServerID).websocketURL()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("websocket URL = %q, want %q", got, tc.want)
			}
		})
	}
}
