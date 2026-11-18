package exaroton

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// BackupProvider is a cloud storage provider used for backups.
type BackupProvider string

const (
	BackupProviderGoogleDrive BackupProvider = "google-drive"
	BackupProviderDropbox     BackupProvider = "dropbox"
	BackupProviderOneDrive    BackupProvider = "onedrive"
)

// BackupType describes whether a backup was created automatically or manually.
type BackupType string

const (
	BackupTypeAutomatic BackupType = "automatic"
	BackupTypeManual    BackupType = "manual"
)

// BackupStatus is the creation status of a backup.
type BackupStatus string

const (
	BackupStatusCreating BackupStatus = "creating"
	BackupStatusCreated  BackupStatus = "created"
)

// BackupInterval is an automatic backup interval in seconds.
type BackupInterval int

const (
	BackupInterval6Hours  BackupInterval = 21600
	BackupInterval12Hours BackupInterval = 43200
	BackupInterval24Hours BackupInterval = 86400
	BackupInterval48Hours BackupInterval = 172800
)

// BackupStorage describes the storage usage of a backup account.
type BackupStorage struct {
	// Used storage in bytes, nil if unknown.
	Used *int64 `json:"used"`

	// Storage limit in bytes, nil if unknown.
	Limit *int64 `json:"limit"`
}

// BackupAccount is a cloud storage account linked to a server. Accounts can
// only be linked in the exaroton panel.
type BackupAccount struct {
	// Unique backup account ID.
	ID string `json:"id"`

	// Display name of the backup account.
	Name string `json:"name"`

	// Cloud storage provider of the backup account.
	Provider BackupProvider `json:"provider"`

	// Storage usage of the backup account.
	Storage BackupStorage `json:"storage"`
}

// Backup describes a server backup.
type Backup struct {
	// Unique backup ID.
	ID string `json:"id"`

	// Name of the backup, nil if it has the default name.
	Name *string `json:"name"`

	// Whether the backup was created automatically or manually.
	Type *BackupType `json:"type"`

	// Creating while the backup is being created, created once it is done.
	// A backup that failed to be created is deleted.
	Status BackupStatus `json:"status"`

	// Unique ID of the server the backup was created from.
	Server string `json:"server"`

	// Unique ID of the software the server had when the backup was created.
	Software *string `json:"software"`

	// Size of the backup in bytes.
	Size *int64 `json:"size"`

	// Unix timestamp of when the backup was created.
	Created *int64 `json:"created"`

	// Unique ID of the backup account the backup is stored in.
	BackupAccount *string `json:"backupAccount"`
}

// BackupPage is one page of backups plus the server's current backup state.
type BackupPage struct {
	PageInfo
	Items []Backup `json:"items"`

	// Whether a backup is currently being created or restored on this server.
	Running bool `json:"running"`
}

// BackupSettings describes the automatic backup settings of a server.
type BackupSettings struct {
	// Whether automatic backups are enabled.
	Enabled bool `json:"enabled"`

	// How many automatic backups are kept; older ones are deleted.
	Limit int `json:"limit"`

	// Interval between automatic backups.
	Interval BackupInterval `json:"interval"`

	// Number of automatic backups that failed in a row since the last
	// successful one.
	Failed int `json:"failed"`
}

// CreateBackupOptions configures CreateBackup. A nil value uses the defaults:
// the first backup account linked to the server and the default name.
type CreateBackupOptions struct {
	// ID of the backup account to store the backup in. Defaults to the first
	// backup account linked to the server.
	BackupAccount string `json:"backupAccount,omitempty"`

	// Name of the backup, at most 512 bytes. If empty, the default name is
	// used.
	Name string `json:"name,omitempty"`
}

// UpdateBackupSettingsOptions configures UpdateBackupSettings. Nil fields are
// left unchanged.
type UpdateBackupSettingsOptions struct {
	// Whether automatic backups are enabled.
	Enabled *bool `json:"enabled,omitempty"`

	// How many automatic backups are kept (2-1000).
	Limit *int `json:"limit,omitempty"`

	// Interval between automatic backups.
	Interval *BackupInterval `json:"interval,omitempty"`
}

// GetBackups returns a page of the server's backups across all linked backup
// accounts (20 per page), newest first. page starts at 1; values <= 0 request
// the first page.
func (s *ServerRef) GetBackups(ctx context.Context, page int) (*BackupPage, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backups", s.serverID)

	var resp RawResponse[BackupPage]

	err := s.client.sendRequestWithQuery(ctx, http.MethodGet, path, pageQuery(page), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// CreateBackup starts creating a backup of the server. The backup is created
// asynchronously with the status creating; poll GetBackups (or Running) to see
// when it is done. A 409 response (another backup operation is running)
// surfaces as *ApiError, as does a 413 response (not enough backup space).
func (s *ServerRef) CreateBackup(ctx context.Context, opts *CreateBackupOptions) (*Backup, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if opts != nil {
		if err := validateBackupName(opts.Name); err != nil {
			return nil, err
		}
	}

	path := fmt.Sprintf("servers/%s/backups", s.serverID)

	var body any
	if opts != nil {
		body = opts
	}

	var resp RawResponse[Backup]

	err := s.client.sendRequest(ctx, http.MethodPost, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetBackup returns a single backup of the server.
func (s *ServerRef) GetBackup(ctx context.Context, backupID string) (*Backup, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateBackupID(backupID); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backups/%s", s.serverID, url.PathEscape(backupID))

	var resp RawResponse[Backup]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// RenameBackup renames a backup. An empty name resets it to the default name.
// Renaming an automatic backup turns it into a manual backup.
func (s *ServerRef) RenameBackup(ctx context.Context, backupID string, name string) (*Backup, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateBackupID(backupID); err != nil {
		return nil, err
	}
	if err := validateBackupName(name); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backups/%s", s.serverID, url.PathEscape(backupID))

	body := struct {
		Name string `json:"name"`
	}{Name: name}

	var resp RawResponse[Backup]

	err := s.client.sendRequest(ctx, http.MethodPut, path, body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// DeleteBackup deletes a backup from its backup account.
func (s *ServerRef) DeleteBackup(ctx context.Context, backupID string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := validateBackupID(backupID); err != nil {
		return err
	}
	path := fmt.Sprintf("servers/%s/backups/%s", s.serverID, url.PathEscape(backupID))

	return s.client.sendEmptyRequest(ctx, http.MethodDelete, path, nil, "application/json")
}

// RestoreBackup starts restoring a backup to the server, replacing its current
// files. The server has to be offline and the backup has to be created. The
// restore runs asynchronously; the returned backup should be polled via
// GetBackups (or Running) before starting the server. A 409 response (another
// backup operation is running) surfaces as *ApiError.
func (s *ServerRef) RestoreBackup(ctx context.Context, backupID string) (*Backup, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateBackupID(backupID); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backups/%s/restore", s.serverID, url.PathEscape(backupID))

	var resp RawResponse[Backup]

	err := s.client.sendRequest(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetBackupAccounts lists the backup accounts linked to the server in the
// server's order.
func (s *ServerRef) GetBackupAccounts(ctx context.Context) ([]BackupAccount, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backup-accounts", s.serverID)

	var resp RawResponse[[]BackupAccount]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}

	return *resp.Data, nil
}

// GetBackupSettings returns the automatic backup settings of the server.
func (s *ServerRef) GetBackupSettings(ctx context.Context) (*BackupSettings, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backup-settings", s.serverID)

	var resp RawResponse[BackupSettings]

	err := s.client.sendRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// UpdateBackupSettings changes the included automatic backup settings and
// returns all settings after the change. Nil fields are left unchanged; at
// least one field must be set.
func (s *ServerRef) UpdateBackupSettings(ctx context.Context, opts UpdateBackupSettingsOptions) (*BackupSettings, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := validateUpdateBackupSettings(opts); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("servers/%s/backup-settings", s.serverID)

	var resp RawResponse[BackupSettings]

	err := s.client.sendRequest(ctx, http.MethodPost, path, opts, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
