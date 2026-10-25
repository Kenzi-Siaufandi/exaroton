package exaroton

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestCreateServer(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": testServerID, "name": "example",
			"address": "example.exaroton.me", "status": 0,
			"players":  map[string]any{"max": 20, "count": 0, "list": []string{}},
			"software": map[string]any{"id": testSoftwareID, "name": "Vanilla", "version": "1.20.4", "type": "vanilla"},
			"shared":   false,
		})
	})

	created, err := c.CreateServer(context.Background(), CreateServerOptions{
		Name:     "example",
		Software: testSoftwareID,
		MOTD:     "&aHello",
		Timezone: "Asia/Jakarta",
		Region:   RegionEU,
	})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost || gotPath != "/servers/" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if created.ID != testServerID || created.Name != "example" {
		t.Fatalf("unexpected server: %+v", created)
	}
	if gotBody["name"] != "example" || gotBody["software"] != testSoftwareID {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if gotBody["motd"] != "§aHello" {
		t.Fatalf("motd was not normalized: %v", gotBody["motd"])
	}
	if gotBody["timezone"] != "Asia/Jakarta" || gotBody["region"] != "eu" {
		t.Fatalf("unexpected optional fields: %v", gotBody)
	}
}

func TestCreateServerSoftwareTypeOnly(t *testing.T) {
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{"id": testServerID, "name": "example"})
	})

	if _, err := c.CreateServer(context.Background(), CreateServerOptions{
		Name:         "example",
		SoftwareType: SoftwareTypeVanilla,
	}); err != nil {
		t.Fatal(err)
	}

	if gotBody["softwareType"] != "vanilla" {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if _, ok := gotBody["software"]; ok {
		t.Fatalf("software should be omitted: %v", gotBody)
	}
}

func TestCreateServerValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	ctx := context.Background()

	tests := []struct {
		name string
		opts CreateServerOptions
	}{
		{"empty name", CreateServerOptions{SoftwareType: SoftwareTypeVanilla}},
		{"whitespace name", CreateServerOptions{Name: "  ", SoftwareType: SoftwareTypeVanilla}},
		{"missing software", CreateServerOptions{Name: "example"}},
		{"short software ID", CreateServerOptions{Name: "example", Software: "short"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := c.CreateServer(ctx, tt.opts); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestCreateServerLimitReached(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		encodeEnvelope(t, w, false, "Server limit reached", nil)
	})

	_, err := c.CreateServer(context.Background(), CreateServerOptions{
		Name:         "example",
		SoftwareType: SoftwareTypeVanilla,
	})

	var apiErr *ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
}

func TestDeleteServer(t *testing.T) {
	for _, data := range []any{map[string]any{}, nil} {
		var gotMethod, gotPath string
		c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			encodeEnvelope(t, w, true, nil, data)
		})

		if err := c.Server(testServerID).DeleteServer(context.Background()); err != nil {
			t.Fatal(err)
		}
		if gotMethod != http.MethodDelete || gotPath != "/servers/"+testServerID+"/" {
			t.Fatalf("request = %s %s", gotMethod, gotPath)
		}
	}
}

func TestDeleteServerFailure(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, false, "server is online", nil)
	})

	if err := c.Server(testServerID).DeleteServer(context.Background()); err == nil {
		t.Fatal("expected success:false error")
	}
}
