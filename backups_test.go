package exaroton

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetBackups(t *testing.T) {
	var gotPath, gotPage string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPage = r.URL.Query().Get("page")
		encodeEnvelope(t, w, true, nil, map[string]any{
			"page": 2, "lastPage": 3, "total": 21, "running": true,
			"items": []any{map[string]any{
				"id": "Xy7zAb3dEf6hIj9k", "name": nil, "type": nil,
				"status": "creating", "server": testServerID, "software": nil,
				"size": nil, "created": 1763251200, "backupAccount": "Ab3dEf6hIj9kLm2n",
			}},
		})
	})

	page, err := c.Server(testServerID).GetBackups(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/backups/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPage != "2" {
		t.Fatalf("page = %q", gotPage)
	}
	if page.Page != 2 || page.LastPage != 3 || page.Total != 21 || !page.Running {
		t.Fatalf("unexpected page: %+v", page)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %+v", page.Items)
	}
	backup := page.Items[0]
	if backup.ID != "Xy7zAb3dEf6hIj9k" || backup.Status != BackupStatusCreating {
		t.Fatalf("unexpected backup: %+v", backup)
	}
	if backup.Name != nil || backup.Type != nil || backup.Software != nil || backup.Size != nil {
		t.Fatalf("nullable fields should be nil: %+v", backup)
	}
	if backup.Created == nil || *backup.Created != 1763251200 {
		t.Fatalf("created = %v", backup.Created)
	}
	if backup.BackupAccount == nil || *backup.BackupAccount != "Ab3dEf6hIj9kLm2n" {
		t.Fatalf("backupAccount = %v", backup.BackupAccount)
	}
}

func TestCreateBackup(t *testing.T) {
	tests := []struct {
		name     string
		opts     *CreateBackupOptions
		wantBody map[string]any
	}{
		{name: "defaults"},
		{
			name: "with options",
			opts: &CreateBackupOptions{
				BackupAccount: "Ab3dEf6hIj9kLm2n",
				Name:          "Before the update",
			},
			wantBody: map[string]any{
				"backupAccount": "Ab3dEf6hIj9kLm2n",
				"name":          "Before the update",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody []byte

			c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotBody, _ = io.ReadAll(r.Body)
				encodeEnvelope(t, w, true, nil, map[string]any{
					"id": "Xy7zAb3dEf6hIj9k", "status": "creating", "server": testServerID,
				})
			})

			backup, err := c.Server(testServerID).CreateBackup(context.Background(), tt.opts)
			if err != nil {
				t.Fatal(err)
			}

			if gotMethod != http.MethodPost {
				t.Fatalf("method = %s", gotMethod)
			}
			if gotPath != "/servers/"+testServerID+"/backups/" {
				t.Fatalf("path = %q", gotPath)
			}
			if backup.ID != "Xy7zAb3dEf6hIj9k" {
				t.Fatalf("unexpected backup: %+v", backup)
			}

			if tt.wantBody == nil {
				if len(strings.TrimSpace(string(gotBody))) != 0 {
					t.Fatalf("unexpected body: %q", gotBody)
				}
				return
			}
			var body map[string]any
			if err := json.Unmarshal(gotBody, &body); err != nil {
				t.Fatalf("body %q: %v", gotBody, err)
			}
			for key, want := range tt.wantBody {
				if body[key] != want {
					t.Fatalf("%s = %v, want %v", key, body[key], want)
				}
			}
		})
	}
}

func TestGetBackup(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": "Xy7zAb3dEf6hIj9k", "status": "created", "server": testServerID,
		})
	})

	backup, err := c.Server(testServerID).GetBackup(context.Background(), "Xy7zAb3dEf6hIj9k")
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/backups/Xy7zAb3dEf6hIj9k/" {
		t.Fatalf("path = %q", gotPath)
	}
	if backup.Status != BackupStatusCreated {
		t.Fatalf("unexpected backup: %+v", backup)
	}
}

func TestRenameBackup(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": "Xy7zAb3dEf6hIj9k", "name": "Before the update",
			"status": "created", "server": testServerID,
		})
	})

	backup, err := c.Server(testServerID).RenameBackup(
		context.Background(), "Xy7zAb3dEf6hIj9k", "Before the update",
	)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/backups/Xy7zAb3dEf6hIj9k/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["name"] != "Before the update" {
		t.Fatalf("body = %v", gotBody)
	}
	if backup.Name == nil || *backup.Name != "Before the update" {
		t.Fatalf("name = %v", backup.Name)
	}
}

func TestDeleteBackup(t *testing.T) {
	var gotMethod, gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{})
	})

	if err := c.Server(testServerID).DeleteBackup(context.Background(), "Xy7zAb3dEf6hIj9k"); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/backups/Xy7zAb3dEf6hIj9k/" {
		t.Fatalf("path = %q", gotPath)
	}

	cFail := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, false, "nope", nil)
	})
	if err := cFail.Server(testServerID).DeleteBackup(context.Background(), "Xy7zAb3dEf6hIj9k"); err == nil {
		t.Fatal("expected success:false error")
	}
}

func TestRestoreBackup(t *testing.T) {
	var gotMethod, gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{
			"id": "Xy7zAb3dEf6hIj9k", "status": "created", "server": testServerID,
		})
	})

	backup, err := c.Server(testServerID).RestoreBackup(context.Background(), "Xy7zAb3dEf6hIj9k")
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/backups/Xy7zAb3dEf6hIj9k/restore/" {
		t.Fatalf("path = %q", gotPath)
	}
	if backup.ID != "Xy7zAb3dEf6hIj9k" {
		t.Fatalf("unexpected backup: %+v", backup)
	}
}

func TestGetBackupAccounts(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, []any{
			map[string]any{
				"id": "Ab3dEf6hIj9kLm2n", "name": "example@gmail.com",
				"provider": "google-drive",
				"storage":  map[string]any{"used": 1073741824, "limit": 16106127360},
			},
			map[string]any{
				"id": "Ab3dEf6hIj9kLm2o", "name": "example",
				"provider": "dropbox",
				"storage":  map[string]any{"used": nil, "limit": nil},
			},
		})
	})

	accounts, err := c.Server(testServerID).GetBackupAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/backup-accounts/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(accounts) != 2 {
		t.Fatalf("accounts = %+v", accounts)
	}
	if accounts[0].Provider != BackupProviderGoogleDrive ||
		accounts[0].Storage.Used == nil || *accounts[0].Storage.Used != 1073741824 {
		t.Fatalf("unexpected account: %+v", accounts[0])
	}
	if accounts[1].Storage.Used != nil || accounts[1].Storage.Limit != nil {
		t.Fatalf("storage should be nil: %+v", accounts[1].Storage)
	}
}

func TestGetBackupSettings(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{
			"enabled": true, "limit": 5, "interval": 86400, "failed": 0,
		})
	})

	settings, err := c.Server(testServerID).GetBackupSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/backup-settings/" {
		t.Fatalf("path = %q", gotPath)
	}
	if !settings.Enabled || settings.Limit != 5 ||
		settings.Interval != BackupInterval24Hours || settings.Failed != 0 {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func backupIntervalPtr(v BackupInterval) *BackupInterval {
	return &v
}

func TestUpdateBackupSettings(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"enabled": false, "limit": 10, "interval": 43200, "failed": 2,
		})
	})

	settings, err := c.Server(testServerID).UpdateBackupSettings(context.Background(),
		UpdateBackupSettingsOptions{
			Enabled:  boolPtr(false),
			Limit:    intPtr(10),
			Interval: backupIntervalPtr(BackupInterval12Hours),
		})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/backup-settings/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotBody) != 3 {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if v, ok := gotBody["enabled"]; !ok || v != false {
		t.Fatalf("enabled not sent explicitly: %v", gotBody)
	}
	if gotBody["limit"] != float64(10) || gotBody["interval"] != float64(43200) {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if settings.Enabled || settings.Limit != 10 || settings.Interval != BackupInterval12Hours {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestBackupValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	server := c.Server(testServerID)
	ctx := context.Background()

	var emptyBackupID string
	if _, err := server.GetBackup(ctx, emptyBackupID); err == nil {
		t.Fatal("expected empty backup ID error")
	}
	if _, err := server.RenameBackup(ctx, emptyBackupID, "x"); err == nil {
		t.Fatal("expected empty backup ID error")
	}
	if err := server.DeleteBackup(ctx, emptyBackupID); err == nil {
		t.Fatal("expected empty backup ID error")
	}
	if _, err := server.RestoreBackup(ctx, emptyBackupID); err == nil {
		t.Fatal("expected empty backup ID error")
	}

	tooLongName := strings.Repeat("a", maxBackupNameBytes+1)
	if _, err := server.CreateBackup(ctx, &CreateBackupOptions{Name: tooLongName}); err == nil {
		t.Fatal("expected long backup name error")
	}
	if _, err := server.RenameBackup(ctx, "Xy7zAb3dEf6hIj9k", tooLongName); err == nil {
		t.Fatal("expected long backup name error")
	}

	tooFew := minBackupLimit - 1
	tooMany := maxBackupLimit + 1
	badInterval := BackupInterval(3600)
	tests := []struct {
		name string
		opts UpdateBackupSettingsOptions
	}{
		{"empty", UpdateBackupSettingsOptions{}},
		{"limit too small", UpdateBackupSettingsOptions{Limit: &tooFew}},
		{"limit too large", UpdateBackupSettingsOptions{Limit: &tooMany}},
		{"invalid interval", UpdateBackupSettingsOptions{Interval: backupIntervalPtr(badInterval)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := server.UpdateBackupSettings(ctx, tt.opts); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	if _, err := c.Server("short").GetBackups(ctx, 1); err == nil {
		t.Fatal("expected server ID validation error")
	}
}
