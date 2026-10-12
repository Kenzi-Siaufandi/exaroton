package exaroton

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testServerID = "EwYiY9IAMtQBTb6U"
	testPoolID   = "X4iJREgqBGFSBtn0"
)

func TestApiErrorTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "Invalid API token", "data": nil})
	}))
	defer srv.Close()

	c, err := NewClient("tok", WithBaseURL(srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetAccount(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if !strings.Contains(err.Error(), "api error (403): Invalid API token") {
		t.Fatalf("message format changed: %q", err.Error())
	}
}

func TestApiErrorUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	}))
	defer srv.Close()

	c, _ := NewClient("tok", WithBaseURL(srv.URL+"/"))
	_, err := c.GetAccount(context.Background())
	var apiErr *ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
		t.Fatalf("expected typed 500, got %v", err)
	}
	if err.Error() != "unexpected status code: 500" {
		t.Fatalf("legacy format broken: %q", err.Error())
	}
}

func TestValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	if _, err := c.Server("short").GetServer(context.Background()); err == nil {
		t.Fatal("expected server ID validation error")
	}
	if _, err := c.CreditPool("short").GetPool(context.Background()); err == nil {
		t.Fatal("expected pool ID validation error")
	}
	s := c.Server(testServerID)
	if _, err := s.SetRam(context.Background(), 1); err == nil {
		t.Fatal("expected ram range error for 1")
	}
	if _, err := s.SetRam(context.Background(), 17); err == nil {
		t.Fatal("expected ram range error for 17")
	}
	if _, err := s.File("").Info(context.Background()); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestUpdateConfigValuesMixedTypes(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []any{
				map[string]any{"key": "max-players", "value": 20, "label": "Max Players", "type": "integer"},
				map[string]any{"key": "pvp", "value": true, "label": "PvP", "type": "boolean"},
			},
		})
	}))
	defer srv.Close()

	c, _ := NewClient("tok", WithBaseURL(srv.URL+"/"))
	f := c.Server(testServerID).File("server.properties")
	got, err := f.UpdateConfigValues(context.Background(), map[string]any{"max-players": 20, "pvp": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if n, ok := gotBody["max-players"].(float64); !ok || n != 20 {
		t.Fatalf("int value not preserved: %v", gotBody)
	}
	if b, ok := gotBody["pvp"].(bool); !ok || !b {
		t.Fatalf("bool value not preserved: %v", gotBody)
	}
	// legacy string wrapper still works
	_, err = f.UpdateConfig(context.Background(), map[string]string{"motd": "hi"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileWriteEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": map[string]any{}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": map[string]any{}})
	}))
	defer srv.Close()

	c, _ := NewClient("tok", WithBaseURL(srv.URL+"/"))
	f := c.Server(testServerID).File("a.txt")
	if err := f.Write(context.Background(), []byte("hi")); err != nil {
		t.Fatal(err)
	}
	if err := f.CreateDirectory(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}

	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "nope", "data": nil})
	}))
	defer srvFail.Close()
	c2, _ := NewClient("tok", WithBaseURL(srvFail.URL+"/"))
	if err := c2.Server(testServerID).File("a.txt").Delete(context.Background()); err == nil {
		t.Fatal("expected success:false error")
	}
}

func TestHandModelsRoundTrip(t *testing.T) {
	// Spec example server must decode into the shipped hand-written types.
	raw := `{"id":"EwYiY9IAMtQBTb6U","name":"example","address":"example.exaroton.me","motd":"hi","status":1,"host":null,"port":null,"players":{"max":20,"count":0,"list":[]},"software":{"id":"kb4p09ABvLjxzedx","name":"Vanilla","version":"1.16.5"},"shared":false}`
	var s Server
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if s.ID != "EwYiY9IAMtQBTb6U" {
		t.Fatalf("Server.ID = %q", s.ID)
	}
	if s.Status != StatusOnline {
		t.Fatalf("Server.Status = %v", s.Status)
	}
	if s.Players.Max != 20 || s.Software.Version != "1.16.5" {
		t.Fatalf("unexpected players/software: %+v", s)
	}
	var fi File
	if err := json.Unmarshal([]byte(`{"path":"world/playerdata","name":"playerdata","isDirectory":true,"size":4096}`), &fi); err != nil {
		t.Fatal(err)
	}
	if !fi.IsDirectory || fi.Size != 4096 {
		t.Fatalf("unexpected file: %+v", fi)
	}
}

func TestServerActionsAllowNullData(t *testing.T) {
	// Server action responses carry nullable data: data:null must succeed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": nil})
	}))
	defer srv.Close()

	c, _ := NewClient("tok", WithBaseURL(srv.URL+"/"))
	s := c.Server(testServerID)
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Start(ctx, UseOwnCredits()); err != nil {
		t.Fatalf("Start with UseOwnCredits: %v", err)
	}
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := s.Restart(ctx); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if err := s.ExecuteCommand(ctx, "say hi"); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if err := s.Whitelist().Add(ctx, "notch"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Whitelist().Remove(ctx, "notch"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "nope", "data": nil})
	}))
	defer srvFail.Close()
	c2, _ := NewClient("tok", WithBaseURL(srvFail.URL+"/"))
	if err := c2.Server(testServerID).Stop(context.Background()); err == nil {
		t.Fatal("expected success:false error for Stop")
	}
}

func TestFilePathValidation(t *testing.T) {
	for _, p := range []string{"", "/abs", "a/../b", "a/./b", "a//b", "..", "."} {
		if err := validateFilePath(p); err == nil {
			t.Fatalf("expected error for path %q", p)
		}
	}
	for _, p := range []string{"a.txt", "dir/a.txt", "dir/", "a b/c+d"} {
		if err := validateFilePath(p); err != nil {
			t.Fatalf("unexpected error for path %q: %v", p, err)
		}
	}
	// Traversal must fail before any network I/O.
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	if _, err := c.Server(testServerID).File("../../etc/passwd").Info(context.Background()); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestFilePathEscaping(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "error": nil,
			"data": map[string]any{"path": "x", "name": "x"},
		})
	}))
	defer srv.Close()

	c, _ := NewClient("tok", WithBaseURL(srv.URL+"/"))
	if _, err := c.Server(testServerID).File("a b/c").Info(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "a%20b/c") {
		t.Fatalf("path not escaped: %q", gotPath)
	}
}
