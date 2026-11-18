package exaroton

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestListRegions(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, []any{
			map[string]any{"id": "eu", "name": "Europe"},
			map[string]any{"id": "na", "name": "North America"},
		})
	})

	regions, err := c.ListRegions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/regions/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(regions) != 2 || regions[0].ID != "eu" || regions[1].Name != "North America" {
		t.Fatalf("unexpected regions: %+v", regions)
	}
}

func TestTransferServer(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": testServerID, "name": "example", "status": 9, "region": "na",
		})
	})

	server, err := c.Server(testServerID).TransferServer(context.Background(), RegionNA)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/transfer/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["region"] != "na" {
		t.Fatalf("body = %v", gotBody)
	}
	if server.Region != RegionNA || server.Status != StatusTransferring {
		t.Fatalf("unexpected server: %+v", server)
	}
}

func TestTransferServerValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	ctx := context.Background()

	if _, err := c.Server(testServerID).TransferServer(ctx, ""); err == nil {
		t.Fatal("expected empty region error")
	}
	if _, err := c.Server("short").TransferServer(ctx, RegionEU); err == nil {
		t.Fatal("expected server ID validation error")
	}
}

func TestTransferServerUnavailable(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeEnvelope(t, w, false, "transfer unavailable", nil)
	})

	_, err := c.Server(testServerID).TransferServer(context.Background(), RegionEU)
	var apiErr *ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
}
