package exaroton

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Permission is a Share Access permission.
//
// Some permissions require others: stop requires server; reinstall and
// domains require options; access requires all other permissions. The API
// enforces these dependencies.
type Permission string

const (
	PermissionServer       Permission = "server"
	PermissionStop         Permission = "stop"
	PermissionConsole      Permission = "console"
	PermissionLog          Permission = "log"
	PermissionOptions      Permission = "options"
	PermissionReinstall    Permission = "reinstall"
	PermissionDomains      Permission = "domains"
	PermissionSoftware     Permission = "software"
	PermissionPlayers      Permission = "players"
	PermissionFiles        Permission = "files"
	PermissionAddons       Permission = "addons"
	PermissionWorlds       Permission = "worlds"
	PermissionBackups      Permission = "backups"
	PermissionTransactions Permission = "transactions"
	PermissionDuplicate    Permission = "duplicate"
	PermissionAccess       Permission = "access"
)

// AccessAccount identifies the account a server is shared with.
type AccessAccount struct {
	// Unique account ID.
	ID string `json:"id"`

	// Account name.
	Name string `json:"name"`
}

// Access describes an account's access to a server via Share Access.
type Access struct {
	// Unique ID of the account's access to the server.
	ID string `json:"id"`

	// The account the server is shared with.
	Account AccessAccount `json:"account"`

	// Permissions of the account on the server.
	Permissions []Permission `json:"permissions"`
}

// GetAccessList lists the accounts the server is shared with via Share
// Access, without the owner.
func (s *ServerRef) GetAccessList(ctx context.Context) ([]Access, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/access", s.serverID)

	var resp RawResponse[[]Access]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// ShareServer shares the server with another exaroton account. A nil or empty
// permissions list uses the API default of ["server"]. A 409 response (the
// server is already shared with this account) and a 413 response (too many
// servers shared with this account) surface as *ApiError.
func (s *ServerRef) ShareServer(ctx context.Context, username string, permissions []Permission) (*Access, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateAccessUsername(username); err != nil {
		return nil, err
	}
	if err := validatePermissions(permissions); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/access", s.serverID)

	body := struct {
		Username    string       `json:"username"`
		Permissions []Permission `json:"permissions,omitempty"`
	}{
		Username:    username,
		Permissions: permissions,
	}

	var resp RawResponse[Access]

	err := s.client.sendRequest(ctx, http.MethodPost, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetAccess returns an account's access to the server.
func (s *ServerRef) GetAccess(ctx context.Context, accessID string) (*Access, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateAccessID(accessID); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/access/%s", s.serverID, url.PathEscape(accessID))

	var resp RawResponse[Access]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// UpdateAccess replaces the permissions of an account on the server. A nil
// permissions list clears all permissions.
func (s *ServerRef) UpdateAccess(ctx context.Context, accessID string, permissions []Permission) (*Access, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateAccessID(accessID); err != nil {
		return nil, err
	}
	if err := validatePermissions(permissions); err != nil {
		return nil, err
	}
	if permissions == nil {
		permissions = []Permission{}
	}
	path := fmt.Sprintf("servers/%s/access/%s", s.serverID, url.PathEscape(accessID))

	body := struct {
		Permissions []Permission `json:"permissions"`
	}{Permissions: permissions}

	var resp RawResponse[Access]

	err := s.client.sendRequest(ctx, http.MethodPut, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// RemoveAccess removes an account's access to the server.
func (s *ServerRef) RemoveAccess(ctx context.Context, accessID string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := validateAccessID(accessID); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/access/%s", s.serverID, url.PathEscape(accessID))

	return s.client.sendEmptyRequest(ctx, http.MethodDelete, path, nil, "application/json")
}
