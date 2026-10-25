package exaroton

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestListSoftware(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		encodeEnvelope(t, w, true, nil, []any{map[string]any{
			"id": testSoftwareID, "name": "Vanilla", "version": "1.20.4",
			"type": "vanilla", "launcher": "CurseForge",
		}})
	})

	list, err := c.ListSoftware(context.Background(), &SoftwareFilter{
		Type:     SoftwareTypeVanilla,
		Launcher: LauncherCurseForge,
	})
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/software/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery.Get("type") != "vanilla" || gotQuery.Get("launcher") != "CurseForge" {
		t.Fatalf("query = %v", gotQuery)
	}
	if len(list) != 1 || list[0].ID != testSoftwareID {
		t.Fatalf("unexpected software: %+v", list)
	}
	if list[0].Launcher == nil || *list[0].Launcher != LauncherCurseForge {
		t.Fatalf("launcher = %v", list[0].Launcher)
	}
}

func TestListSoftwareNoFilter(t *testing.T) {
	var gotRawQuery string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		encodeEnvelope(t, w, true, nil, []any{})
	})

	list, err := c.ListSoftware(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotRawQuery != "" {
		t.Fatalf("unexpected query: %q", gotRawQuery)
	}
	if len(list) != 0 {
		t.Fatalf("unexpected software: %+v", list)
	}
}

func TestListSoftwareCustomType(t *testing.T) {
	var gotType string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("type")
		encodeEnvelope(t, w, true, nil, []any{})
	})

	if _, err := c.ListSoftware(context.Background(), &SoftwareFilter{
		Type: SoftwareType("future-type"),
	}); err != nil {
		t.Fatal(err)
	}
	if gotType != "future-type" {
		t.Fatalf("type = %q", gotType)
	}
}

func TestGetSoftwareDetails(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": testSoftwareID, "name": "Paper", "version": "1.20.4",
			"type": "paper", "launcher": nil,
			"addons":      []string{"plugin"},
			"description": "A high-performance fork.",
			"link":        "https://papermc.io/",
			"reinstall":   true,
		})
	})

	details, err := c.Server(testServerID).GetSoftwareDetails(context.Background(), testSoftwareID)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/software/"+testSoftwareID+"/" {
		t.Fatalf("path = %q", gotPath)
	}
	if details.ID != testSoftwareID || details.Reinstall != true {
		t.Fatalf("unexpected details: %+v", details)
	}
	if details.Description == nil || *details.Description != "A high-performance fork." {
		t.Fatalf("description = %v", details.Description)
	}
	if details.Link == nil || *details.Link != "https://papermc.io/" {
		t.Fatalf("link = %v", details.Link)
	}
	if len(details.Addons) != 1 || details.Addons[0] != "plugin" {
		t.Fatalf("addons = %v", details.Addons)
	}
	if details.Launcher != nil {
		t.Fatalf("launcher = %v", details.Launcher)
	}
}

func TestGetSoftwareDetailsNullFields(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": testSoftwareID, "name": "Vanilla", "version": "1.20.4",
			"type": "vanilla", "description": nil, "link": nil, "reinstall": false,
		})
	})

	details, err := c.Server(testServerID).GetSoftwareDetails(context.Background(), testSoftwareID)
	if err != nil {
		t.Fatal(err)
	}
	if details.Description != nil || details.Link != nil || details.Reinstall {
		t.Fatalf("unexpected details: %+v", details)
	}
}

func TestInstallSoftware(t *testing.T) {
	tests := []struct {
		name          string
		opts          []InstallSoftwareOption
		wantReinstall bool
	}{
		{name: "without reinstall"},
		{name: "with reinstall", opts: []InstallSoftwareOption{ConfirmReinstall()}, wantReinstall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody []byte

			c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotBody, _ = io.ReadAll(r.Body)
				encodeEnvelope(t, w, true, nil, nil)
			})

			if err := c.Server(testServerID).InstallSoftware(
				context.Background(), testSoftwareID, tt.opts...,
			); err != nil {
				t.Fatal(err)
			}

			if gotMethod != http.MethodPost {
				t.Fatalf("method = %s", gotMethod)
			}
			if gotPath != "/servers/"+testServerID+"/software/"+testSoftwareID+"/" {
				t.Fatalf("path = %q", gotPath)
			}

			if tt.wantReinstall {
				var body map[string]any
				if err := json.Unmarshal(gotBody, &body); err != nil {
					t.Fatalf("body %q: %v", gotBody, err)
				}
				if body["reinstall"] != true {
					t.Fatalf("body = %v", body)
				}
			} else if len(strings.TrimSpace(string(gotBody))) != 0 {
				t.Fatalf("unexpected body: %q", gotBody)
			}
		})
	}
}

func TestInstallSoftwareFailure(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, false, "reinstall required", nil)
	})

	if err := c.Server(testServerID).InstallSoftware(context.Background(), testSoftwareID); err == nil {
		t.Fatal("expected success:false error")
	}
}

func TestSoftwareValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	ctx := context.Background()

	if _, err := c.Server(testServerID).GetSoftwareDetails(ctx, "short"); err == nil {
		t.Fatal("expected software ID validation error")
	}
	if err := c.Server(testServerID).InstallSoftware(ctx, "short"); err == nil {
		t.Fatal("expected software ID validation error")
	}
	if _, err := c.Server("short").GetSoftwareDetails(ctx, testSoftwareID); err == nil {
		t.Fatal("expected server ID validation error")
	}
	if err := c.Server("short").InstallSoftware(ctx, testSoftwareID); err == nil {
		t.Fatal("expected server ID validation error")
	}
}
