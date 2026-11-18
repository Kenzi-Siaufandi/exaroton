package exaroton

import (
	"context"
	"fmt"
	"net/http"
)

// Transaction describes a credit transaction.
//
// Credits and Amount are display only: binary floats cannot represent decimal
// credits exactly, so format for display and avoid arithmetic.
type Transaction struct {
	// Unique transaction ID.
	ID string `json:"id"`

	// Type of the transaction, e.g. "run" (running a server), "retention"
	// (keeping a server's files), "payment", "refund", "pool-deposit",
	// "pool-withdraw", "redeem-code" or a compensation ("comp-*").
	Type string `json:"type"`

	// Credits of the transaction from the perspective of the account, pool
	// or server the list belongs to: positive for income, negative for
	// usage. Grows while the transaction is active.
	Credits float64 `json:"credits"`

	// Rate used to calculate the credits, in credits per Interval.
	Rate *int `json:"rate"`

	// Interval of Rate in seconds.
	Interval *int `json:"interval"`

	// Factor used to calculate the credits, e.g. the GB of RAM.
	Amount *float64 `json:"amount"`

	// Whether the transaction is still running (e.g. the server is online).
	Active bool `json:"active"`

	// Whether the transaction has been paid.
	Paid bool `json:"paid"`

	// Unix timestamp of the start of the transaction.
	StartTime int64 `json:"startTime"`

	// Unix timestamp of the end of the transaction, nil while it is active.
	EndTime *int64 `json:"endTime"`

	// Unique ID of the account of the transaction, if any.
	Account *string `json:"account"`

	// Unique ID of the credit pool of the transaction, if any.
	Pool *string `json:"pool"`

	// Unique ID of the server of the transaction, if any.
	Server *string `json:"server"`
}

// TransactionPage is one page of transactions.
type TransactionPage struct {
	PageInfo
	Items []Transaction `json:"items"`
}

// GetAccountTransactions returns a page of the account's transactions (20 per
// page), newest first. page starts at 1; values <= 0 request the first page.
func (c *Client) GetAccountTransactions(ctx context.Context, page int) (*TransactionPage, error) {
	path := "account/transactions"

	var resp RawResponse[TransactionPage]

	err := c.sendRequestWithQuery(ctx, http.MethodGet, path, pageQuery(page), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetTransactions returns a page of the server's transactions (20 per page),
// newest first. Requires the transactions permission. page starts at 1;
// values <= 0 request the first page.
func (s *ServerRef) GetTransactions(ctx context.Context, page int) (*TransactionPage, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/transactions", s.serverID)

	var resp RawResponse[TransactionPage]

	err := s.client.sendRequestWithQuery(ctx, http.MethodGet, path, pageQuery(page), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetTransactions returns a page of the credit pool's transactions (20 per
// page), newest first. page starts at 1; values <= 0 request the first page.
func (c *CreditRef) GetTransactions(ctx context.Context, page int) (*TransactionPage, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("billing/pools/%s/transactions", c.poolID)

	var resp RawResponse[TransactionPage]

	err := c.client.sendRequestWithQuery(ctx, http.MethodGet, path, pageQuery(page), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
