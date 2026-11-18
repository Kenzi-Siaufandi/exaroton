package exaroton

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DomainStatus is the linking status of a custom domain.
type DomainStatus string

const (
	// DomainStatusPending means the domain has not been checked successfully yet.
	DomainStatusPending DomainStatus = "pending"

	// DomainStatusLinked means the domain's NS records pointed to the listed
	// nameservers on the last check.
	DomainStatusLinked DomainStatus = "linked"

	// DomainStatusUnlinked means the domain's NS records did not point to the
	// listed nameservers on the last check.
	DomainStatusUnlinked DomainStatus = "unlinked"
)

// DomainProtocol is the protocol of a custom domain.
type DomainProtocol string

const (
	DomainProtocolHTTPS DomainProtocol = "https"
)

// Domain describes a custom domain of a server.
type Domain struct {
	// Unique custom domain ID.
	ID string `json:"id"`

	// The domain (lowercase).
	Domain string `json:"domain"`

	// The linking status of the domain.
	Status DomainStatus `json:"status"`

	// Protocol of the domain.
	Protocol DomainProtocol `json:"protocol"`

	// Port of the domain.
	Port int `json:"port"`

	// Unix timestamp of the last check, nil if it has not been checked.
	CheckedTime *int64 `json:"checkedTime"`

	// Unix timestamp of when the domain was added.
	CreatedTime *int64 `json:"createdTime"`

	// The nameservers the NS records of the domain have to point to.
	Nameservers []string `json:"nameservers"`
}

// UpdateDomainOptions configures UpdateDomain. At least one field must be set.
type UpdateDomainOptions struct {
	// Protocol of the domain.
	Protocol DomainProtocol `json:"protocol,omitempty"`

	// Port of the domain (1-65535).
	Port *int `json:"port,omitempty"`
}

// GetDomains lists the custom domains of the server.
func (s *ServerRef) GetDomains(ctx context.Context) ([]Domain, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/domains", s.serverID)

	var resp RawResponse[[]Domain]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// AddDomain adds a custom domain to the server (up to 10 per server). Point
// the domain's NS records at the returned nameservers; if they already do it
// is linked immediately, otherwise it stays pending until CheckDomain
// succeeds.
func (s *ServerRef) AddDomain(ctx context.Context, domain string) (*Domain, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateDomainName(domain); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/domains", s.serverID)

	body := struct {
		Domain string `json:"domain"`
	}{Domain: domain}

	var resp RawResponse[Domain]

	err := s.client.sendRequest(ctx, http.MethodPost, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetDomain returns a single custom domain of the server.
func (s *ServerRef) GetDomain(ctx context.Context, domainID string) (*Domain, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateDomainID(domainID); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/domains/%s", s.serverID, url.PathEscape(domainID))

	var resp RawResponse[Domain]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// UpdateDomain changes the protocol and/or port of a custom domain. Fields
// left zero are kept; at least one must be set.
func (s *ServerRef) UpdateDomain(ctx context.Context, domainID string, opts UpdateDomainOptions) (*Domain, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateDomainID(domainID); err != nil {
		return nil, err
	}
	if err := validateDomainUpdate(opts); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/domains/%s", s.serverID, url.PathEscape(domainID))

	var resp RawResponse[Domain]

	err := s.client.sendRequest(ctx, http.MethodPut, path, opts, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// DeleteDomain removes a custom domain from the server.
func (s *ServerRef) DeleteDomain(ctx context.Context, domainID string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := validateDomainID(domainID); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/domains/%s", s.serverID, url.PathEscape(domainID))

	return s.client.sendEmptyRequest(ctx, http.MethodDelete, path, nil, "application/json")
}

// CheckDomain checks again whether the NS records of a domain that is not
// linked yet point to the listed nameservers and returns the updated domain.
// A domain is checked at most every 10 seconds; linked domains are not checked
// again.
func (s *ServerRef) CheckDomain(ctx context.Context, domainID string) (*Domain, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateDomainID(domainID); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/domains/%s/check", s.serverID, url.PathEscape(domainID))

	var resp RawResponse[Domain]

	err := s.client.sendRequest(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
