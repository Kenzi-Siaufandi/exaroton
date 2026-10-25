package exaroton

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// SoftwareFilter narrows the results of ListSoftware. Empty fields are
// omitted from the request.
type SoftwareFilter struct {
	// Only show software options of this type (e.g. SoftwareTypeVanilla).
	Type SoftwareType
	// Only show modpacks for this launcher (e.g. LauncherCurseForge).
	Launcher SoftwareLauncher
}

// ListSoftware lists the software options available to install. A nil filter
// returns every option.
func (c *Client) ListSoftware(ctx context.Context, filter *SoftwareFilter) ([]Software, error) {
	query := url.Values{}
	if filter != nil {
		if filter.Type != "" {
			query.Set("type", string(filter.Type))
		}
		if filter.Launcher != "" {
			query.Set("launcher", string(filter.Launcher))
		}
	}

	var resp RawResponse[[]Software]

	err := c.sendRequestWithQuery(ctx, http.MethodGet, "software", query, nil, &resp)
	if err != nil {
		return nil, err
	}

	return *resp.Data, nil
}

// SoftwareDetails describes installing a software on a specific server.
type SoftwareDetails struct {
	Software

	// Description of the software, if available.
	Description *string `json:"description"`

	// Link with more information about the software, if available.
	Link *string `json:"link"`

	// Whether installing this software on the server requires a reinstall
	// (deleting all existing server files).
	Reinstall bool `json:"reinstall"`
}

// GetSoftwareDetails returns information about installing softwareID on the
// server, including whether a reinstall is required.
func (s *ServerRef) GetSoftwareDetails(ctx context.Context, softwareID string) (*SoftwareDetails, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateSoftwareID(softwareID); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("servers/%s/software/%s", s.serverID, url.PathEscape(softwareID))

	var resp RawResponse[SoftwareDetails]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

type installSoftwareConfig struct {
	reinstall bool
}

// InstallSoftwareOption configures InstallSoftware.
type InstallSoftwareOption func(*installSoftwareConfig)

// ConfirmReinstall confirms that installing the target software may delete
// all existing server files. It must be passed when GetSoftwareDetails
// reports Reinstall == true.
func ConfirmReinstall() InstallSoftwareOption {
	return func(c *installSoftwareConfig) {
		c.reinstall = true
	}
}

// InstallSoftware installs softwareID on the server. Check
// GetSoftwareDetails first: when a reinstall is required, pass
// ConfirmReinstall() to acknowledge that existing server files will be
// deleted.
func (s *ServerRef) InstallSoftware(ctx context.Context, softwareID string, opts ...InstallSoftwareOption) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := validateSoftwareID(softwareID); err != nil {
		return err
	}

	config := &installSoftwareConfig{}
	for _, opt := range opts {
		opt(config)
	}

	path := fmt.Sprintf("servers/%s/software/%s", s.serverID, url.PathEscape(softwareID))

	var body any
	if config.reinstall {
		body = struct {
			Reinstall bool `json:"reinstall"`
		}{Reinstall: true}
	}

	return s.client.sendActionRequest(ctx, http.MethodPost, path, body)
}
