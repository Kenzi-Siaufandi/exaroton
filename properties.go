package exaroton

import (
	"context"
	"fmt"
	"net/http"
)

// AutoStartType selects which players can start a server automatically.
type AutoStartType string

const (
	AutoStartTypeAny         AutoStartType = "any"
	AutoStartTypeWhitelisted AutoStartType = "whitelisted"
	AutoStartTypeOps         AutoStartType = "ops"
)

// AutoUpdateMode controls automatic software updates.
type AutoUpdateMode string

const (
	AutoUpdateNever  AutoUpdateMode = "never"
	AutoUpdateAsk    AutoUpdateMode = "ask"
	AutoUpdateAlways AutoUpdateMode = "always"
)

// ServerProperties is the set of server properties returned by the API.
//
// Pointer fields distinguish "not available or not changed" (nil) from an
// explicit false or zero value. Only properties supported by the server (and
// permitted for the account) are included, so any field may be nil.
type ServerProperties struct {
	// Whether the server stops automatically when no players are online.
	AutoStop *bool `json:"autostop,omitempty"`

	// Minutes without players after which the server stops (3-1440).
	AutoStopTime *int `json:"autostop-time,omitempty"`

	// Whether the server starts automatically when a player joins.
	AutoStart *bool `json:"autostart,omitempty"`

	// Which players can start the server automatically.
	AutoStartType *AutoStartType `json:"autostart-type,omitempty"`

	// Whether the server starts automatically when a player joins via the
	// launcher.
	LauncherAutoStart *bool `json:"launcher-autostart,omitempty"`

	// Which players can start the server automatically via the launcher.
	LauncherAutoStartType *AutoStartType `json:"launcher-autostart-type,omitempty"`

	// Whether players joining the offline server see a lobby.
	EnableLobby *bool `json:"enable-lobby,omitempty"`

	// Whether the server software is updated automatically.
	AutoUpdate *AutoUpdateMode `json:"auto-update,omitempty"`

	// Whether the world is upgraded to the current version when the server
	// starts.
	ForceUpgrade *bool `json:"force-upgrade,omitempty"`
}

// GetProperties returns the server properties available to the account.
func (s *ServerRef) GetProperties(ctx context.Context) (*ServerProperties, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/options/properties", s.serverID)

	var resp RawResponse[ServerProperties]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// UpdateProperties changes the given server properties and returns all
// properties after the change. Nil fields are not changed; either all given
// properties are applied or none. At least one field must be set.
func (s *ServerRef) UpdateProperties(ctx context.Context, props ServerProperties) (*ServerProperties, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateServerProperties(props); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("servers/%s/options/properties", s.serverID)

	var resp RawResponse[ServerProperties]

	err := s.client.sendRequest(ctx, http.MethodPost, path, props, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
