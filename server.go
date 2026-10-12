package exaroton

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ServerRef is a bound reference to one server. Obtain via Client.Server.
type ServerRef struct {
	client   *Client
	serverID string
}

// Server returns a reference to the server with the given ID.
func (c *Client) Server(id string) *ServerRef {
	return &ServerRef{
		client:   c,
		serverID: id,
	}
}

// Server describes an exaroton server.
type Server struct {
	// The server unique ID
	ID string `json:"id"`

	// The server name
	Name string `json:"name"`

	// The server domain
	Address string `json:"address"`

	// The server MOTD (Message of the Day)
	Motd string `json:"motd"`

	// The server status (enum of type ServerStatus)
	Status Status `json:"status"`

	// The server host address, only available if the server is online
	Host *string `json:"host"`

	// The server port, only available if the server is online
	Port *int `json:"port"`

	// Information about players
	Players Players `json:"players"`

	// Information about the server software
	Software Software `json:"software"`

	// Whether the server is shared
	Shared bool `json:"shared"`
}

// Players summarizes online player state.
type Players struct {
	// Maximum number of players allowed
	Max int `json:"max"`

	// Current number of players connected
	Count int `json:"count"`

	// List of player names currently online
	List []string `json:"list"`
}

// Software describes the server software.
type Software struct {
	// Unique software ID
	ID string `json:"id"`

	// Software name (e.g., Vanilla, Spigot)
	Name string `json:"name"`

	// Software version (e.g., 1.16.5)
	Version string `json:"version"`
}

// Status is the numeric server status returned by the API.
type Status int

// StatusUnknown marks a status value the client does not recognize. The API
// only sends 0-10; String returns "UNKNOWN" for anything else, including
// future values. Compare against this constant, not the string.
const StatusUnknown Status = -1

const (
	StatusOffline      Status = iota // 0
	StatusOnline                     // 1
	StatusStarting                   // 2
	StatusStopping                   // 3
	StatusRestarting                 // 4
	StatusSaving                     // 5
	StatusLoading                    // 6
	StatusCrashed                    // 7
	StatusPending                    // 8
	StatusTransferring               // 9
	StatusPreparing                  // 10
)

// String returns the API status name (OFFLINE, ONLINE, ...) or UNKNOWN.
func (s Status) String() string {
	switch s {
	case StatusOffline:
		return "OFFLINE"
	case StatusOnline:
		return "ONLINE"
	case StatusStarting:
		return "STARTING"
	case StatusStopping:
		return "STOPPING"
	case StatusRestarting:
		return "RESTARTING"
	case StatusSaving:
		return "SAVING"
	case StatusLoading:
		return "LOADING"
	case StatusCrashed:
		return "CRASHED"
	case StatusPending:
		return "PENDING"
	case StatusTransferring:
		return "TRANSFERRING"
	case StatusPreparing:
		return "PREPARING"
	default:
		return "UNKNOWN"
	}
}

// GetServer fetches the server details.
func (s *ServerRef) GetServer(ctx context.Context) (*Server, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s", s.serverID)
	var resp RawResponse[Server]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// Logs holds raw server log content.
type Logs struct {
	// The log content
	Content string `json:"content"`
}

// GetLogs fetches the server logs.
func (s *ServerRef) GetLogs(ctx context.Context) (*Logs, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/logs", s.serverID)
	var resp RawResponse[Logs]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// ShareLogs holds an mclo.gs share result.
type ShareLogs struct {
	// The log id
	ID string `json:"id"`

	// The mclo.gs url
	URL string `json:"url"`

	// The raw api.mclo.gs url
	Raw string `json:"raw"`
}

// ShareLogs uploads the server logs to mclo.gs and returns the share URLs.
func (s *ServerRef) ShareLogs(ctx context.Context) (*ShareLogs, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/logs/share", s.serverID)
	var resp RawResponse[ShareLogs]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// maxLogLineSize caps a single log line for the line-based helpers. Console
// lines can exceed bufio.Scanner's 64KB default, so both Lines and
// PrintWithLineNumbers raise the limit to the same value.
const maxLogLineSize = 10 * 1024 * 1024

// Lines splits log content into lines.
func (l *Logs) Lines() []string {
	if l.Content == "" {
		return []string{}
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(l.Content))
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogLineSize)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// Tail returns the last n log lines.
func (l *Logs) Tail(n int) []string {
	if n <= 0 {
		return []string{}
	}
	lines := l.Lines()
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// Ram holds the server RAM allocation in GB.
type Ram struct {
	// The amount of RAM in GB
	Ram int `json:"ram"`
}

// GetRam returns the allocated RAM in GB.
func (s *ServerRef) GetRam(ctx context.Context) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	path := fmt.Sprintf("servers/%s/options/ram", s.serverID)
	var resp RawResponse[Ram]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return 0, err
	}

	return resp.Data.Ram, nil
}

// SetRam sets the allocated RAM in GB (2-16). It fails fast on out-of-range values.
func (s *ServerRef) SetRam(ctx context.Context, gb int) (int, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	if err := validateRam(gb); err != nil {
		return 0, err
	}
	path := fmt.Sprintf("servers/%s/options/ram", s.serverID)
	var resp RawResponse[Ram]

	err := s.client.sendRequest(ctx, http.MethodPost, path, &Ram{Ram: gb}, &resp)
	if err != nil {
		return 0, err
	}

	return resp.Data.Ram, nil
}

// MOTD holds the server message of the day.
type MOTD struct {
	Motd string `json:"motd"`
}

// GetMOTD returns the server MOTD.
func (s *ServerRef) GetMOTD(ctx context.Context) (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	path := fmt.Sprintf("servers/%s/options/motd", s.serverID)
	var resp RawResponse[MOTD]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return "", err
	}

	return resp.Data.Motd, nil
}

// SetMOTD sets the server MOTD. Ampersands are converted to section signs.
func (s *ServerRef) SetMOTD(ctx context.Context, motd string) (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	path := fmt.Sprintf("servers/%s/options/motd", s.serverID)
	var resp RawResponse[MOTD]

	motd = strings.ReplaceAll(motd, "&", "§")

	err := s.client.sendRequest(ctx, http.MethodPost, path, &MOTD{Motd: motd}, &resp)
	if err != nil {
		return "", err
	}

	return resp.Data.Motd, nil
}

// StartConfig holds options for Start.
type StartConfig struct {
	UseOwnCredits bool
}

// StartOption configures Start.
type StartOption func(*StartConfig)

// UseOwnCredits makes Start use the account's own credits.
func UseOwnCredits() StartOption {
	return func(c *StartConfig) {
		c.UseOwnCredits = true
	}
}

// Start starts the server. A 208 AlreadyStarting response surfaces as *ApiError.
func (s *ServerRef) Start(ctx context.Context, opts ...StartOption) error {
	if err := s.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/start", s.serverID)

	config := &StartConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.UseOwnCredits {
		body := struct {
			UseOwnCredits bool `json:"useOwnCredits"`
		}{
			UseOwnCredits: true,
		}

		return s.client.sendActionRequest(ctx, http.MethodPost, path, body)
	}
	return s.client.sendActionRequest(ctx, http.MethodGet, path, nil)
}

// Stop stops the server. A 208 AlreadyStopping response surfaces as *ApiError.
func (s *ServerRef) Stop(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/stop", s.serverID)
	return s.client.sendActionRequest(ctx, http.MethodGet, path, nil)
}

// Restart restarts the server.
func (s *ServerRef) Restart(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/restart", s.serverID)
	return s.client.sendActionRequest(ctx, http.MethodGet, path, nil)
}

// Command is a console command payload.
type Command struct {
	Command string `json:"command"`
}

// ExecuteCommand runs a console command via REST.
func (s *ServerRef) ExecuteCommand(ctx context.Context, command string) error {
	if err := s.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/command", s.serverID)
	return s.client.sendActionRequest(ctx, http.MethodPost, path, &Command{Command: command})
}

// ExtendTime holds extra startup wait time in seconds.
type ExtendTime struct {
	// In seconds
	Time int `json:"time"`
}

// ExtendWait extends the startup wait time.
func (s *ServerRef) ExtendWait(ctx context.Context, seconds int) error {
	if err := s.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/extend-time", s.serverID)
	var resp RawResponse[ExtendTime]

	err := s.client.sendRequest(ctx, http.MethodPost, path, &ExtendTime{Time: seconds}, &resp)
	if err != nil {
		return err
	}

	return nil
}

// PlayerListRef is a bound reference to one server player list.
type PlayerListRef struct {
	client   *Client
	serverID string
	listName string
}

func (s *ServerRef) list(name string) *PlayerListRef {
	return &PlayerListRef{
		client:   s.client,
		serverID: s.serverID,
		listName: name,
	}
}

// PlayerList returns a reference to a player list by name (e.g. "whitelist").
// Use GetAvailablePlayerLists to discover valid names.
func (s *ServerRef) PlayerList(name string) *PlayerListRef {
	return s.list(name)
}

// GetAvailablePlayerLists lists all player list names available for the server.
func (s *ServerRef) GetAvailablePlayerLists(ctx context.Context) ([]string, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/playerlists", s.serverID)
	var resp RawResponse[[]string]

	if err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return *resp.Data, nil
}

// Whitelist returns a reference to the whitelist player list.
func (s *ServerRef) Whitelist() *PlayerListRef {
	return s.list("whitelist")
}

// Ops returns a reference to the ops player list.
func (s *ServerRef) Ops() *PlayerListRef {
	return s.list("ops")
}

// BannedPlayers returns a reference to the banned-players player list.
func (s *ServerRef) BannedPlayers() *PlayerListRef {
	return s.list("banned-players")
}

// BannedIPs returns a reference to the banned-ips player list.
func (s *ServerRef) BannedIPs() *PlayerListRef {
	return s.list("banned-ips")
}

// GetEntries lists the entries in the player list.
func (l *PlayerListRef) GetEntries(ctx context.Context) ([]string, error) {
	if err := l.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/playerlists/%s/", l.serverID, url.PathEscape(l.listName))

	var resp RawResponse[[]string]

	err := l.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return *resp.Data, nil
}

// Add appends entries to the player list.
func (l *PlayerListRef) Add(ctx context.Context, entries ...string) error {
	if len(entries) == 0 {
		return errors.New("no entries provided")
	}

	path := fmt.Sprintf("servers/%s/playerlists/%s/", l.serverID, url.PathEscape(l.listName))

	body := map[string][]string{
		"entries": entries,
	}

	return l.client.sendActionRequest(ctx, http.MethodPut, path, body)
}

// Remove deletes entries from the player list.
func (l *PlayerListRef) Remove(ctx context.Context, entries ...string) error {
	if len(entries) == 0 {
		return errors.New("no entries provided")
	}

	path := fmt.Sprintf("servers/%s/playerlists/%s/", l.serverID, url.PathEscape(l.listName))

	body := map[string][]string{
		"entries": entries,
	}

	return l.client.sendActionRequest(ctx, http.MethodDelete, path, body)
}

// FileRef is a bound reference to one file path on a server.
type FileRef struct {
	client   *Client
	serverID string
	path     string
}

// File describes a file or directory.
type File struct {
	Path         string  `json:"path"`
	Name         string  `json:"name"`
	IsTextFile   bool    `json:"isTextFile"`
	IsConfigFile bool    `json:"isConfigFile"`
	IsDirectory  bool    `json:"isDirectory"`
	IsLog        bool    `json:"isLog"`
	IsReadable   bool    `json:"isReadable"`
	IsWritable   bool    `json:"isWritable"`
	Size         int64   `json:"size"`
	Children     []*File `json:"children,omitempty"`
}

// File returns a reference to a path on the server. A leading slash is trimmed.
func (s *ServerRef) File(path string) *FileRef {
	path = strings.TrimPrefix(path, "/")

	return &FileRef{
		client:   s.client,
		serverID: s.serverID,
		path:     path,
	}
}

// escapeFilePath escapes each slash-separated segment so file paths cannot
// break out of the files/ URL namespace.
func escapeFilePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// Info fetches file metadata.
func (f *FileRef) Info(ctx context.Context) (*File, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/files/info/%s", f.serverID, escapeFilePath(f.path))

	var resp RawResponse[File]

	err := f.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// FileResponse holds raw file bytes.
type FileResponse struct {
	Raw []byte
}

// Read downloads the raw file content.
func (f *FileRef) Read(ctx context.Context) (*FileResponse, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/files/data/%s", f.serverID, escapeFilePath(f.path))

	data, err := f.client.sendByteRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	return &FileResponse{Raw: data}, nil
}

// Write uploads content to the file path. Directories are rejected.
func (f *FileRef) Write(ctx context.Context, content []byte) error {
	if err := f.validate(); err != nil {
		return err
	}

	if strings.HasSuffix(f.path, "/") {
		return errors.New("operation can't be performed on a directory")
	}

	path := fmt.Sprintf("servers/%s/files/data/%s", f.serverID, escapeFilePath(f.path))

	return f.client.sendEmptyRequest(ctx, http.MethodPut, path, content, "application/octet-stream")
}

// CreateDirectory creates the referenced directory.
func (f *FileRef) CreateDirectory(ctx context.Context) error {
	if err := f.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/files/data/%s", f.serverID, escapeFilePath(f.path))

	return f.client.sendEmptyRequest(ctx, http.MethodPut, path, nil, "inode/directory")
}

// Delete deletes the file or directory at the referenced path.
func (f *FileRef) Delete(ctx context.Context) error {
	if err := f.validate(); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/files/data/%s", f.serverID, escapeFilePath(f.path))

	return f.client.sendEmptyRequest(ctx, http.MethodDelete, path, nil, "application/json")
}

// ReplaceLineAt returns a copy with the 1-based line replaced, preserving indentation.
func (r *FileResponse) ReplaceLineAt(lineNum int, newText string) ([]byte, error) {
	if r == nil {
		return nil, errors.New("nil file response")
	}
	if lineNum < 1 {
		return nil, fmt.Errorf("invalid line number %d: must be >= 1", lineNum)
	}

	// Count lines to detect out-of-range requests instead of silently returning a copy.
	total := 1
	for _, b := range r.Raw {
		if b == '\n' {
			total++
		}
	}
	// Trailing newline does not create an extra addressable line.
	if len(r.Raw) > 0 && r.Raw[len(r.Raw)-1] == '\n' {
		total--
	}
	if lineNum > total {
		return nil, fmt.Errorf("line number %d out of range (total %d lines)", lineNum, total)
	}

	// Pre-size for the worst case: original content plus replacement text.
	newTextLen := len(newText)
	buffer := bytes.NewBuffer(make([]byte, 0, len(r.Raw)+newTextLen))

	currentLine := 1
	lineStart := 0

	for i := 0; i <= len(r.Raw); i++ {
		// Process line at newline or EOF
		if i == len(r.Raw) || r.Raw[i] == '\n' {
			if currentLine == lineNum {
				// Get indentation and write replacement
				indent := r.getIndentationFromBytes(r.Raw[lineStart:i])
				buffer.Write(indent)
				buffer.WriteString(strings.TrimSpace(newText))
			} else {
				// Write original line
				buffer.Write(r.Raw[lineStart:i])
			}

			if i < len(r.Raw) {
				buffer.WriteByte('\n')
				lineStart = i + 1
			}
			currentLine++
		}
	}

	return buffer.Bytes(), nil
}

func (r *FileResponse) getIndentationFromBytes(line []byte) []byte {
	for i, b := range line {
		if b != ' ' && b != '\t' {
			return line[:i]
		}
	}
	return line
}

// PrintWithLineNumbers writes the file with line numbers for debugging.
func (r *FileResponse) PrintWithLineNumbers(w io.Writer) error {
	scanner := bufio.NewScanner(bytes.NewReader(r.Raw))
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogLineSize)
	lineNum := 1

	for scanner.Scan() {
		_, err := fmt.Fprintf(w, "%4d | %s\n", lineNum, scanner.Text())
		if err != nil {
			return err
		}
		lineNum++
	}

	return scanner.Err()
}

// Config describes one config file option.
type Config struct {
	Key     string   `json:"key"`
	Value   any      `json:"value"`
	Label   string   `json:"label"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"`
}

// Config fetches parsed config options. Only .properties and .dat files are supported.
// TODO: Create detection if they are trying to update a config file with an unsupported extension
func (f *FileRef) Config(ctx context.Context) ([]Config, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/files/config/%s", f.serverID, escapeFilePath(f.path))

	var resp RawResponse[[]Config]

	err := f.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return *resp.Data, nil
}

// UpdateConfig updates config options with string values.
// It is a backward-compatible wrapper around UpdateConfigValues.
func (f *FileRef) UpdateConfig(ctx context.Context, config map[string]string) ([]Config, error) {
	// Backward-compatible wrapper: config values allow any JSON type,
	// so delegate to UpdateConfigValues.
	anyCfg := make(map[string]any, len(config))
	for k, v := range config {
		anyCfg[k] = v
	}
	return f.UpdateConfigValues(ctx, anyCfg)
}

// UpdateConfigValues updates config options with arbitrary JSON values.
func (f *FileRef) UpdateConfigValues(ctx context.Context, config map[string]any) ([]Config, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/files/config/%s", f.serverID, escapeFilePath(f.path))

	var resp RawResponse[[]Config]

	err := f.client.sendRequest(ctx, http.MethodPost, path, config, &resp)
	if err != nil {
		return nil, err
	}

	return *resp.Data, nil
}
