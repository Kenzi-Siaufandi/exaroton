package exaroton

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func accountTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("tok", WithBaseURL(srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func encodeEnvelope(t *testing.T, w http.ResponseWriter, success bool, errMsg any, data any) {
	t.Helper()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": success,
		"error":   errMsg,
		"data":    data,
	})
}

func TestGetAccount(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": "EwYiY9IAMtQBTb6s", "name": "example", "email": "example@exaroton.com",
			"maxServers": 10, "verified": true, "credits": 42.05,
		})
	})

	a, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "EwYiY9IAMtQBTb6s" || a.Name != "example" || !a.Verified || a.Credits != 42.05 {
		t.Fatalf("unexpected account: %+v", a)
	}
	if a.MaxServers != 10 {
		t.Fatalf("MaxServers = %d", a.MaxServers)
	}
}

func TestGetServers(t *testing.T) {
	serverPayload := []any{map[string]any{
		"id": "EwYiY9IAMtQBTb6U", "name": "example",
		"address": "example.exaroton.me", "motd": "hi", "status": 1,
		"players":  map[string]any{"max": 20, "count": 0, "list": []string{}},
		"software": map[string]any{"id": "x", "name": "Vanilla", "version": "1.16.5"},
		"shared":   false,
	}}
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, true, nil, serverPayload)
	})

	servers, err := c.GetServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].ID != testServerID {
		t.Fatalf("unexpected servers: %+v", servers)
	}

	// Null data fails the strict envelope: list endpoints require data.
	cNull := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, true, nil, nil)
	})
	if _, err := cNull.GetServers(context.Background()); err == nil {
		t.Fatal("expected missing-data error for null list")
	}
}

func TestListCreditPools(t *testing.T) {
	poolPayload := []any{map[string]any{
		"id": "X4iJREgqBGFSBtn0", "name": "example", "credits": 100.05,
		"servers": 1, "owner": "abc", "isOwner": true,
		"members": 2, "ownShare": 0.5, "ownCredits": 21.0,
	}}
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/pools/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		encodeEnvelope(t, w, true, nil, poolPayload)
	})

	pools, err := c.ListCreditPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 1 || pools[0].ID != testPoolID || pools[0].Credits != 100.05 {
		t.Fatalf("unexpected pools: %+v", pools)
	}
}

func TestCreditPoolDetail(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/billing/pools/" + testPoolID + "/members/":
			encodeEnvelope(t, w, true, nil, []any{map[string]any{
				"account": "abc", "name": "example",
				"share": 0.5, "credits": 21.0, "isOwner": false,
			}})
		case "/billing/pools/" + testPoolID + "/servers/":
			encodeEnvelope(t, w, true, nil, []any{})
		case "/billing/pools/" + testPoolID + "/":
			encodeEnvelope(t, w, true, nil, map[string]any{
				"id": testPoolID, "name": "example", "credits": 100.05,
				"servers": 1, "owner": "abc", "isOwner": true,
				"members": 2, "ownShare": 0.5, "ownCredits": 21.0,
			})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()

	pool, err := c.CreditPool(testPoolID).GetPool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if pool.ID != testPoolID || pool.Members != 2 {
		t.Fatalf("unexpected pool: %+v", pool)
	}

	members, err := c.CreditPool(testPoolID).GetPoolMembers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Name != "example" {
		t.Fatalf("unexpected members: %+v", members)
	}

	servers, err := c.CreditPool(testPoolID).GetPoolServers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Fatalf("unexpected servers: %+v", servers)
	}
}

func TestCreditPoolValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	ctx := context.Background()
	if _, err := c.CreditPool("short").GetPool(ctx); err == nil {
		t.Fatal("expected pool ID validation error")
	}
	if _, err := c.CreditPool("short").GetPoolMembers(ctx); err == nil {
		t.Fatal("expected pool ID validation error")
	}
	if _, err := c.CreditPool("short").GetPoolServers(ctx); err == nil {
		t.Fatal("expected pool ID validation error")
	}
	if _, err := c.CreditPool("").GetPool(ctx); err == nil {
		t.Fatal("expected pool ID validation error")
	}
}
