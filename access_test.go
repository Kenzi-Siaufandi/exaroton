package exaroton

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func accessPayload() map[string]any {
	return map[string]any{
		"id": "Hj8kLm2nPq4rSt7u",
		"account": map[string]any{
			"id": "EwYiY9IAMtQBTb6s", "name": "example",
		},
		"permissions": []string{"server", "stop", "console"},
	}
}

func TestGetAccessList(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, []any{accessPayload()})
	})

	access, err := c.Server(testServerID).GetAccessList(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/access/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(access) != 1 {
		t.Fatalf("access = %+v", access)
	}
	if access[0].ID != "Hj8kLm2nPq4rSt7u" || access[0].Account.Name != "example" {
		t.Fatalf("unexpected access: %+v", access[0])
	}
	if len(access[0].Permissions) != 3 || access[0].Permissions[1] != PermissionStop {
		t.Fatalf("permissions = %v", access[0].Permissions)
	}
}

func TestShareServer(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		wantKey     bool
	}{
		{name: "default permissions"},
		{
			name:        "explicit permissions",
			permissions: []Permission{PermissionServer, PermissionStop},
			wantKey:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]any

			c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				encodeEnvelope(t, w, true, nil, accessPayload())
			})

			access, err := c.Server(testServerID).ShareServer(
				context.Background(), "example", tt.permissions,
			)
			if err != nil {
				t.Fatal(err)
			}

			if gotMethod != http.MethodPost {
				t.Fatalf("method = %s", gotMethod)
			}
			if gotPath != "/servers/"+testServerID+"/access/" {
				t.Fatalf("path = %q", gotPath)
			}
			if gotBody["username"] != "example" {
				t.Fatalf("body = %v", gotBody)
			}
			_, hasPermissions := gotBody["permissions"]
			if hasPermissions != tt.wantKey {
				t.Fatalf("permissions present = %v, want %v", hasPermissions, tt.wantKey)
			}
			if access.ID != "Hj8kLm2nPq4rSt7u" {
				t.Fatalf("unexpected access: %+v", access)
			}
		})
	}
}

func TestGetAccess(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, accessPayload())
	})

	access, err := c.Server(testServerID).GetAccess(context.Background(), "Hj8kLm2nPq4rSt7u")
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/access/Hj8kLm2nPq4rSt7u/" {
		t.Fatalf("path = %q", gotPath)
	}
	if access.Account.ID != "EwYiY9IAMtQBTb6s" {
		t.Fatalf("unexpected access: %+v", access)
	}
}

func TestUpdateAccess(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, accessPayload())
	})

	access, err := c.Server(testServerID).UpdateAccess(context.Background(),
		"Hj8kLm2nPq4rSt7u", []Permission{PermissionServer, PermissionConsole})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/access/Hj8kLm2nPq4rSt7u/" {
		t.Fatalf("path = %q", gotPath)
	}
	permissions, ok := gotBody["permissions"].([]any)
	if !ok || len(permissions) != 2 || permissions[0] != "server" || permissions[1] != "console" {
		t.Fatalf("body = %v", gotBody)
	}
	if access.ID != "Hj8kLm2nPq4rSt7u" {
		t.Fatalf("unexpected access: %+v", access)
	}
}

func TestUpdateAccessNilPermissions(t *testing.T) {
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, accessPayload())
	})

	if _, err := c.Server(testServerID).UpdateAccess(
		context.Background(), "Hj8kLm2nPq4rSt7u", nil,
	); err != nil {
		t.Fatal(err)
	}

	permissions, ok := gotBody["permissions"].([]any)
	if !ok || len(permissions) != 0 {
		t.Fatalf("permissions should be an empty array: %v", gotBody)
	}
}

func TestRemoveAccess(t *testing.T) {
	var gotMethod, gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{})
	})

	if err := c.Server(testServerID).RemoveAccess(context.Background(), "Hj8kLm2nPq4rSt7u"); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/access/Hj8kLm2nPq4rSt7u/" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestShareServerConflict(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		encodeEnvelope(t, w, false, "already shared", nil)
	})

	_, err := c.Server(testServerID).ShareServer(context.Background(), "example", nil)
	var apiErr *ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
}

func TestAccessValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	server := c.Server(testServerID)
	ctx := context.Background()

	if _, err := server.ShareServer(ctx, "", nil); err == nil {
		t.Fatal("expected empty username error")
	}
	badPermission := Permission("root")
	if _, err := server.ShareServer(ctx, "example", []Permission{badPermission}); err == nil {
		t.Fatal("expected permission error")
	}
	if _, err := server.UpdateAccess(ctx, "Hj8kLm2nPq4rSt7u", []Permission{badPermission}); err == nil {
		t.Fatal("expected permission error")
	}
	if _, err := server.GetAccess(ctx, ""); err == nil {
		t.Fatal("expected empty access ID error")
	}
	if err := server.RemoveAccess(ctx, ""); err == nil {
		t.Fatal("expected empty access ID error")
	}
	if _, err := c.Server("short").GetAccessList(ctx); err == nil {
		t.Fatal("expected server ID validation error")
	}
}
