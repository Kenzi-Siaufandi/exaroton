package exaroton

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseMinecraftVersion(t *testing.T) {
	cases := []struct {
		in                string
		major, minor, pat int
		ok                bool
	}{
		{"1.16.5", 1, 16, 5, true},
		{"1.21.9", 1, 21, 9, true},
		{"1.16", 1, 16, 0, true},
		{"1.21.9-pre1", 1, 21, 9, true},
		{"", 0, 0, 0, false},
		{"abc", 0, 0, 0, false},
		{"1", 0, 0, 0, false},
	}
	for _, c := range cases {
		major, minor, patch, ok := parseMinecraftVersion(c.in)
		if ok != c.ok || major != c.major || minor != c.minor || patch != c.pat {
			t.Fatalf("parse %q = %d.%d.%d,%v", c.in, major, minor, patch, ok)
		}
	}
}

func versionTestSession(t *testing.T, version string) *Session {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":      testServerID,
				"name":    "example",
				"address": "example.exaroton.me",
				"motd":    "hi",
				"status":  1,
				"players": map[string]any{"max": 20, "count": 0, "list": []string{}},
				"software": map[string]any{
					"id":      "kb4p09ABvLjxzedx",
					"name":    "Vanilla",
					"version": version,
				},
				"shared": false,
			},
		})
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient("tok", WithBaseURL(srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	return &Session{server: c.Server(testServerID)}
}

func TestSubscribeVersionGates(t *testing.T) {
	ctx := context.Background()

	old := versionTestSession(t, "1.15.2")
	if err := old.SubscribeTick(ctx); err == nil || !strings.Contains(err.Error(), "1.16") {
		t.Fatalf("expected 1.16 gate, got %v", err)
	}

	pre := versionTestSession(t, "1.21.8")
	if err := pre.SubscribeManagement(ctx); err == nil || !strings.Contains(err.Error(), "1.21.9") {
		t.Fatalf("expected 1.21.9 gate, got %v", err)
	}

	ok := versionTestSession(t, "1.21.9")
	if err := ok.requireMinVersion(ctx, 1, 21, 9, "management"); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}
	if err := ok.requireMinVersion(ctx, 1, 16, 0, "tick"); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}

	// Fail open when there is no server ref to check against.
	bare := &Session{}
	if err := bare.requireMinVersion(ctx, 1, 21, 9, "management"); err != nil {
		t.Fatalf("expected fail-open, got %v", err)
	}
}

func TestStatusUnknown(t *testing.T) {
	if StatusUnknown != -1 {
		t.Fatalf("StatusUnknown = %d", StatusUnknown)
	}
	if got := Status(99).String(); got != "UNKNOWN" {
		t.Fatalf("Status(99).String() = %q", got)
	}
	if got := StatusOnline.String(); got != "ONLINE" {
		t.Fatalf("StatusOnline.String() = %q", got)
	}
}

func TestSendAppliesDefaultTimeout(t *testing.T) {
	// readyChan never closes; Send(Background()) must fail via the
	// session timeout instead of blocking forever.
	s := &Session{
		done:        make(chan struct{}),
		readyChan:   make(chan struct{}),
		sendTimeout: 50 * time.Millisecond,
	}
	start := time.Now()
	err := s.Send(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "cannot send message") {
		t.Fatalf("expected send timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Send blocked for %v", elapsed)
	}
}

func TestDownloadClientNoTimeout(t *testing.T) {
	c, err := NewClient("tok", WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	if err != nil {
		t.Fatal(err)
	}
	dl := c.downloadClient()
	if dl.Timeout != 0 {
		t.Fatalf("download client Timeout = %v, want 0", dl.Timeout)
	}
	if dl.Transport != c.httpClient.Transport {
		t.Fatal("download client must share the transport")
	}
}

func TestPrintWithLineNumbersLongLine(t *testing.T) {
	// Lines and PrintWithLineNumbers must share the raised line limit.
	long := strings.Repeat("x", 100*1024)
	if got := len((&Logs{Content: long + "\nshort\n"}).Lines()); got != 2 {
		t.Fatalf("Lines() = %d, want 2", got)
	}
	r := &FileResponse{Raw: []byte(long + "\nshort\n")}
	var buf bytes.Buffer
	if err := r.PrintWithLineNumbers(&buf); err != nil {
		t.Fatalf("PrintWithLineNumbers: %v", err)
	}
	if !strings.Contains(buf.String(), "   1 | "+long) {
		t.Fatal("long line missing from output")
	}
}
