package exaroton

import (
	"context"
	"fmt"
	"net/http"
)

// RegionInfo describes a region servers can be located in.
type RegionInfo struct {
	// Unique region ID, e.g. RegionEU ("eu").
	ID Region `json:"id"`

	// Display name of the region, e.g. "Europe".
	Name string `json:"name"`
}

// ListRegions lists the regions servers can be located in. The Region
// constants (RegionEU, RegionNA) cover the regions known to this client; use
// the returned IDs for newly added regions when creating or transferring a
// server.
func (c *Client) ListRegions(ctx context.Context) ([]RegionInfo, error) {
	path := "regions"

	var resp RawResponse[[]RegionInfo]

	err := c.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// TransferServer transfers an offline (or crashed) server to another region.
// The transfer runs asynchronously: the returned server has the status
// TRANSFERRING until the copied files are in place. Poll GetServer (or watch
// the status stream) to see when it is done. A 503 response (transfer
// temporarily unavailable) surfaces as *ApiError.
func (s *ServerRef) TransferServer(ctx context.Context, region Region) (*Server, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateRegion(region); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("servers/%s/transfer", s.serverID)
	body := struct {
		Region Region `json:"region"`
	}{Region: region}

	var resp RawResponse[Server]

	err := s.client.sendRequest(ctx, http.MethodPost, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
