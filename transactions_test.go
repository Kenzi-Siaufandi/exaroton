package exaroton

import (
	"context"
	"net/http"
	"testing"
)

func transactionPagePayload() map[string]any {
	return map[string]any{
		"page": 2, "lastPage": 3, "total": 21,
		"items": []any{map[string]any{
			"id": "Tr4nS4cT10nId123", "type": "run", "credits": -1.5,
			"rate": 1, "interval": 3600, "amount": 4.0, "active": false,
			"paid": true, "startTime": 1763247600, "endTime": nil,
			"account": "EwYiY9IAMtQBTb6s", "pool": nil, "server": testServerID,
		}},
	}
}

func assertTransactionPage(t *testing.T, page *TransactionPage) {
	t.Helper()
	if page.Page != 2 || page.LastPage != 3 || page.Total != 21 {
		t.Fatalf("pagination = %+v", page.PageInfo)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %+v", page.Items)
	}
	tx := page.Items[0]
	if tx.ID != "Tr4nS4cT10nId123" || tx.Type != "run" || tx.Credits != -1.5 {
		t.Fatalf("unexpected transaction: %+v", tx)
	}
	if tx.Rate == nil || *tx.Rate != 1 || tx.Interval == nil || *tx.Interval != 3600 {
		t.Fatalf("rate/interval = %v/%v", tx.Rate, tx.Interval)
	}
	if tx.Amount == nil || *tx.Amount != 4 {
		t.Fatalf("amount = %v", tx.Amount)
	}
	if tx.Active || !tx.Paid || tx.StartTime != 1763247600 {
		t.Fatalf("unexpected flags/times: %+v", tx)
	}
	if tx.EndTime != nil || tx.Pool != nil {
		t.Fatalf("nullable fields should be nil: %+v", tx)
	}
	if tx.Account == nil || *tx.Account != "EwYiY9IAMtQBTb6s" {
		t.Fatalf("account = %v", tx.Account)
	}
	if tx.Server == nil || *tx.Server != testServerID {
		t.Fatalf("server = %v", tx.Server)
	}
}

func TestGetAccountTransactions(t *testing.T) {
	var gotPath, gotPage string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPage = r.URL.Query().Get("page")
		encodeEnvelope(t, w, true, nil, transactionPagePayload())
	})

	page, err := c.GetAccountTransactions(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/account/transactions/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPage != "2" {
		t.Fatalf("page = %q", gotPage)
	}
	assertTransactionPage(t, page)
}

func TestGetAccountTransactionsDefaultPage(t *testing.T) {
	var gotRawQuery string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		encodeEnvelope(t, w, true, nil, transactionPagePayload())
	})

	if _, err := c.GetAccountTransactions(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if gotRawQuery != "" {
		t.Fatalf("unexpected query: %q", gotRawQuery)
	}
}

func TestGetServerTransactions(t *testing.T) {
	var gotPath, gotPage string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPage = r.URL.Query().Get("page")
		encodeEnvelope(t, w, true, nil, transactionPagePayload())
	})

	page, err := c.Server(testServerID).GetTransactions(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/transactions/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPage != "2" {
		t.Fatalf("page = %q", gotPage)
	}
	assertTransactionPage(t, page)
}

func TestGetPoolTransactions(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, transactionPagePayload())
	})

	page, err := c.CreditPool(testPoolID).GetTransactions(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/billing/pools/"+testPoolID+"/transactions/" {
		t.Fatalf("path = %q", gotPath)
	}
	assertTransactionPage(t, page)
}

func TestTransactionValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	ctx := context.Background()

	if _, err := c.Server("short").GetTransactions(ctx, 1); err == nil {
		t.Fatal("expected server ID validation error")
	}
	if _, err := c.CreditPool("short").GetTransactions(ctx, 1); err == nil {
		t.Fatal("expected pool ID validation error")
	}
}
