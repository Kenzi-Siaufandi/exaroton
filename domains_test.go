package exaroton

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func domainPayload() map[string]any {
	return map[string]any{
		"id": "Pq4rSt7uVw0xYz3a", "domain": "play.example.com",
		"status": "linked", "protocol": "https", "port": 80,
		"checkedTime": 1763251200, "createdTime": 1763251100,
		"nameservers": []string{"galileo.exaroton.com", "columbus.exaroton.com"},
	}
}

func TestGetDomains(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, []any{domainPayload()})
	})

	domains, err := c.Server(testServerID).GetDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/domains/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(domains) != 1 {
		t.Fatalf("domains = %+v", domains)
	}
	domain := domains[0]
	if domain.ID != "Pq4rSt7uVw0xYz3a" || domain.Status != DomainStatusLinked {
		t.Fatalf("unexpected domain: %+v", domain)
	}
	if domain.Protocol != DomainProtocolHTTPS || domain.Port != 80 {
		t.Fatalf("unexpected protocol/port: %+v", domain)
	}
	if domain.CheckedTime == nil || *domain.CheckedTime != 1763251200 {
		t.Fatalf("checkedTime = %v", domain.CheckedTime)
	}
	if len(domain.Nameservers) != 2 || domain.Nameservers[0] != "galileo.exaroton.com" {
		t.Fatalf("nameservers = %v", domain.Nameservers)
	}
}

func TestAddDomain(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": "Pq4rSt7uVw0xYz3a", "domain": "play.example.com",
			"status": "pending", "protocol": "https", "port": 80,
			"checkedTime": nil, "createdTime": 1763251200,
			"nameservers": []string{"galileo.exaroton.com", "columbus.exaroton.com"},
		})
	})

	domain, err := c.Server(testServerID).AddDomain(context.Background(), "play.example.com")
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/domains/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["domain"] != "play.example.com" {
		t.Fatalf("body = %v", gotBody)
	}
	if domain.Status != DomainStatusPending || domain.CheckedTime != nil {
		t.Fatalf("unexpected domain: %+v", domain)
	}
}

func TestGetAndCheckDomain(t *testing.T) {
	calls := 0
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodGet {
			if r.URL.Path != "/servers/"+testServerID+"/domains/Pq4rSt7uVw0xYz3a/" {
				t.Errorf("GET path = %q", r.URL.Path)
			}
		} else if r.URL.Path != "/servers/"+testServerID+"/domains/Pq4rSt7uVw0xYz3a/check/" {
			t.Errorf("POST path = %q", r.URL.Path)
		}
		encodeEnvelope(t, w, true, nil, domainPayload())
	})

	ctx := context.Background()
	server := c.Server(testServerID)

	if _, err := server.GetDomain(ctx, "Pq4rSt7uVw0xYz3a"); err != nil {
		t.Fatal(err)
	}
	checked, err := server.CheckDomain(ctx, "Pq4rSt7uVw0xYz3a")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if checked.Status != DomainStatusLinked {
		t.Fatalf("unexpected domain: %+v", checked)
	}
}

func TestUpdateDomain(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, domainPayload())
	})

	port := 8080
	domain, err := c.Server(testServerID).UpdateDomain(context.Background(), "Pq4rSt7uVw0xYz3a",
		UpdateDomainOptions{Protocol: DomainProtocolHTTPS, Port: &port})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/domains/Pq4rSt7uVw0xYz3a/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["protocol"] != "https" || gotBody["port"] != float64(8080) {
		t.Fatalf("body = %v", gotBody)
	}
	if domain.ID != "Pq4rSt7uVw0xYz3a" {
		t.Fatalf("unexpected domain: %+v", domain)
	}
}

func TestDeleteDomain(t *testing.T) {
	var gotMethod, gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{})
	})

	if err := c.Server(testServerID).DeleteDomain(context.Background(), "Pq4rSt7uVw0xYz3a"); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/domains/Pq4rSt7uVw0xYz3a/" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestDomainValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	server := c.Server(testServerID)
	ctx := context.Background()

	if _, err := server.AddDomain(ctx, ""); err == nil {
		t.Fatal("expected empty domain error")
	}
	if _, err := server.AddDomain(ctx, "https://play.example.com"); err == nil {
		t.Fatal("expected scheme error")
	}
	if _, err := server.AddDomain(ctx, "play.example.com/path"); err == nil {
		t.Fatal("expected path error")
	}
	if _, err := server.GetDomain(ctx, ""); err == nil {
		t.Fatal("expected empty domain ID error")
	}
	if _, err := server.UpdateDomain(ctx, "", UpdateDomainOptions{Protocol: DomainProtocolHTTPS}); err == nil {
		t.Fatal("expected empty domain ID error")
	}
	if err := server.DeleteDomain(ctx, ""); err == nil {
		t.Fatal("expected empty domain ID error")
	}
	if _, err := server.CheckDomain(ctx, ""); err == nil {
		t.Fatal("expected empty domain ID error")
	}

	if _, err := server.UpdateDomain(ctx, "Pq4rSt7uVw0xYz3a", UpdateDomainOptions{}); err == nil {
		t.Fatal("expected empty update error")
	}
	badProtocol := DomainProtocol("http")
	if _, err := server.UpdateDomain(ctx, "Pq4rSt7uVw0xYz3a",
		UpdateDomainOptions{Protocol: badProtocol}); err == nil {
		t.Fatal("expected protocol error")
	}
	tooLow := minDomainPort - 1
	tooHigh := maxDomainPort + 1
	for _, port := range []int{tooLow, tooHigh} {
		if _, err := server.UpdateDomain(ctx, "Pq4rSt7uVw0xYz3a",
			UpdateDomainOptions{Port: &port}); err == nil {
			t.Fatalf("expected port error for %d", port)
		}
	}

	if _, err := c.Server("short").GetDomains(ctx); err == nil {
		t.Fatal("expected server ID validation error")
	}
}
