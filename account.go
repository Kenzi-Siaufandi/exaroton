package exaroton

import (
	"context"
	"fmt"
	"net/http"
)

// Account describes the API account.
type Account struct {
	// Unique account ID.
	ID string `json:"id"`

	// The username of the account.
	Name string `json:"name"`

	// The registered email address.
	Email string `json:"email"`

	// Maximum number of servers this account can own.
	MaxServers int `json:"maxServers"`

	// The verification status of the email address.
	Verified bool `json:"verified"`

	// The current balance. Display only: a binary float cannot represent
	// decimal credits exactly, so format for display and avoid arithmetic.
	Credits float64 `json:"credits"`
}

// GetAccount returns the account tied to the API token.
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	path := "account"

	var resp RawResponse[Account]

	err := c.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetServers lists servers visible to the token.
func (c *Client) GetServers(ctx context.Context) ([]Server, error) {
	path := "servers"

	var resp RawResponse[[]Server]

	err := c.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// CreditPool describes a credit pool.
//
// Credit money fields (Credits, OwnCredits) are display only: binary floats
// cannot represent decimal credits exactly, so format for display and avoid
// arithmetic.
type CreditPool struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Credits    float64 `json:"credits"`
	Servers    int     `json:"servers"`
	Owner      string  `json:"owner"`
	IsOwner    bool    `json:"isOwner"`
	Members    int     `json:"members"`
	OwnShare   float64 `json:"ownShare"`
	OwnCredits float64 `json:"ownCredits"`
}

// ListCreditPools lists credit pools visible to the token.
func (c *Client) ListCreditPools(ctx context.Context) ([]CreditPool, error) {
	path := "billing/pools"

	var resp RawResponse[[]CreditPool]

	err := c.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// CreditRef is a bound reference to one credit pool. Obtain via Client.CreditPool.
type CreditRef struct {
	client *Client
	poolID string
}

// CreditPool returns a reference to the credit pool with the given ID.
func (c *Client) CreditPool(id string) *CreditRef {
	return &CreditRef{
		client: c,
		poolID: id,
	}
}

// GetPool fetches the credit pool details.
func (c *CreditRef) GetPool(ctx context.Context) (*CreditPool, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("billing/pools/%s", c.poolID)

	var resp RawResponse[CreditPool]

	err := c.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// Member describes a credit pool member.
//
// Credits is display only: a binary float cannot represent decimal credits
// exactly, so format for display and avoid arithmetic.
type Member struct {
	Account string  `json:"account"`
	Name    string  `json:"name"`
	Share   float64 `json:"share"`
	Credits float64 `json:"credits"`
	IsOwner bool    `json:"isOwner"`
}

// GetPoolMembers lists credit pool members.
func (c *CreditRef) GetPoolMembers(ctx context.Context) ([]Member, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("billing/pools/%s/members", c.poolID)

	var resp RawResponse[[]Member]

	err := c.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// GetPoolServers lists servers attached to the credit pool.
func (c *CreditRef) GetPoolServers(ctx context.Context) ([]Server, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("billing/pools/%s/servers", c.poolID)

	var resp RawResponse[[]Server]

	err := c.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}
