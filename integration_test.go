//go:build integration

package exaroton

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// The live integration tests are excluded from the default build. Run them
// explicitly with the integration tag:
//
//	go test -tags=integration -count=1 -timeout 10m .
//
// They are skipped unless their environment variables are set.
//
// Required:
//   EXAROTON_TOKEN   API token
//   EXAROTON_SERVER  server ID under test
//
// Optional:
//   EXAROTON_CREDIT_POOL       credit pool ID
//   EXAROTON_SHARE_USER        account to share the server with
//   EXAROTON_DOMAIN            domain to add and remove
//   EXAROTON_INSTALL_SOFTWARE  software ID to install
//   EXAROTON_ALLOW_REINSTALL   set to 1 to allow deleting server files
//   EXAROTON_RESTORE_BACKUP    backup ID to restore
//   EXAROTON_TRANSFER_REGION   region to transfer the server to
//   EXAROTON_CREATE_SERVER     set to create and delete a temporary server
//   EXAROTON_CREATE_REGION     optional region for the temporary server

// requireEnv returns an environment variable, skipping when it is not set.
func requireEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("%s not set; live integration test", name)
	}
	return value
}

// liveClient creates a client for live integration tests.
func liveClient(t *testing.T) *Client {
	t.Helper()

	client, err := NewClient(requireEnv(t, "EXAROTON_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// liveServer returns a reference to the configured test server.
func liveServer(t *testing.T) *ServerRef {
	t.Helper()
	return liveClient(t).Server(requireEnv(t, "EXAROTON_SERVER"))
}

// liveName returns a unique name for resources created by live tests.
func liveName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// waitForBackupCreated polls until the backup is no longer being created.
func waitForBackupCreated(ctx context.Context, t *testing.T, server *ServerRef, backupID string) *Backup {
	t.Helper()

	for {
		backup, err := server.GetBackup(ctx, backupID)
		if err != nil {
			t.Fatal(err)
		}
		if backup == nil {
			t.Fatalf("backup %s returned no data", backupID)
		}
		t.Logf("backup %s status: %s", backupID, backup.Status)
		if backup.Status == BackupStatusCreated {
			return backup
		}

		select {
		case <-ctx.Done():
			t.Fatalf("backup %s was not created: %v (last status %s)", backupID, ctx.Err(), backup.Status)
		case <-time.After(5 * time.Second):
		}
	}
}

func TestAccount(t *testing.T) {
	client := liveClient(t)

	account, err := client.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(account)
}

func TestServers(t *testing.T) {
	client := liveClient(t)

	servers, err := client.GetServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(servers)
}

func TestServer(t *testing.T) {
	server := liveServer(t)

	serverInfo, err := server.GetServer(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(serverInfo)
}

func TestServerLogs(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	serverLogs, err := server.GetLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(serverLogs.Lines()[0])

	sharedLogs, err := server.ShareLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(sharedLogs)
}

func TestServerRam(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	ramInfo, err := server.GetRam(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ramInfo)

	updatedRam, err := server.SetRam(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(updatedRam)
}

func TestServerMOTD(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	currentMOTD, err := server.GetMOTD(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(currentMOTD)

	updatedMOTD, err := server.SetMOTD(ctx, "&cNew MOTD")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(updatedMOTD)
}

func TestServerStart(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	err := server.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerStop(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	err := server.Stop(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerRestart(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	err := server.Restart(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerCommand(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	err := server.ExecuteCommand(ctx, "say Hello World")
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerExtendTime(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	err := server.ExtendWait(ctx, 120)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerPlayerList(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	if err := server.Ops().Add(ctx, "Test", "Test1"); err != nil {
		t.Fatal(err)
	}

	t.Log(server.Ops().GetEntries(ctx))

	if err := server.Ops().Remove(ctx, "Test", "Test1"); err != nil {
		t.Fatal(err)
	}

	t.Log(server.Ops().GetEntries(ctx))

}

func TestServerFile(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	configFile := server.File("plugins/bStats/config.yml")
	configInfo, err := configFile.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(configInfo)

	configData, err := configFile.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	patched, err := configData.ReplaceLineAt(6, "enabled: false")
	if err != nil {
		t.Fatal(err)
	}
	if err := configFile.Write(ctx, patched); err != nil {
		t.Fatal(err)
	}

	targetDir := server.File("testing")
	targetFile := server.File("testing/test.yaml")

	err = targetDir.CreateDirectory(ctx)
	if err != nil {
		t.Fatal(err)
	}

	targetFile.Write(ctx, []byte("test"))
	fileContent, err := targetFile.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fileContent.PrintWithLineNumbers(os.Stdout)
}

func TestServerConfig(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	propsFile := server.File("server.properties")
	configOld, err := propsFile.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(configOld)

	configNew, err := propsFile.UpdateConfig(ctx, map[string]string{
		"gamemode": "adventure",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(configNew)
}

func TestCreditPools(t *testing.T) {
	client := liveClient(t)

	ctx := context.Background()
	allPools, err := client.ListCreditPools(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(allPools)

	poolRef := client.CreditPool(requireEnv(t, "EXAROTON_CREDIT_POOL"))
	poolDetail, err := poolRef.GetPool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(poolDetail)

	members, err := poolRef.GetPoolMembers(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(members)

	servers, err := poolRef.GetPoolServers(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(servers)
}

func TestLiveRegions(t *testing.T) {
	client := liveClient(t)

	regions, err := client.ListRegions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(regions) == 0 {
		t.Fatal("no regions returned")
	}

	for _, region := range regions {
		t.Logf("%s: %s", region.ID, region.Name)
	}
}

func TestLiveServerProperties(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	props, err := server.GetProperties(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("properties: %+v", props)
	if props == nil {
		t.Skip("no properties returned")
	}
	if props.AutoStop == nil && props.AutoStopTime == nil && props.AutoStart == nil &&
		props.AutoStartType == nil && props.LauncherAutoStart == nil &&
		props.LauncherAutoStartType == nil && props.EnableLobby == nil &&
		props.AutoUpdate == nil && props.ForceUpgrade == nil {
		t.Skip("no writable properties returned")
	}

	// Send the current values back as a no-op update.
	updated, err := server.UpdateProperties(ctx, *props)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("updated properties: %+v", updated)
}

func TestLiveAccountTransactions(t *testing.T) {
	client := liveClient(t)

	page, err := client.GetAccountTransactions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("no transaction page returned")
	}

	t.Logf("account transactions: page=%d last=%d total=%d", page.Page, page.LastPage, page.Total)
	for _, transaction := range page.Items {
		t.Logf("%s: %s %.2f", transaction.ID, transaction.Type, transaction.Credits)
	}
}

func TestLiveServerTransactions(t *testing.T) {
	server := liveServer(t)

	page, err := server.GetTransactions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("no transaction page returned")
	}

	t.Logf("server transactions: page=%d last=%d total=%d", page.Page, page.LastPage, page.Total)
	for _, transaction := range page.Items {
		t.Logf("%s: %s %.2f", transaction.ID, transaction.Type, transaction.Credits)
	}
}

func TestLiveCreditPoolTransactions(t *testing.T) {
	client := liveClient(t)

	page, err := client.CreditPool(requireEnv(t, "EXAROTON_CREDIT_POOL")).
		GetTransactions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("no transaction page returned")
	}

	t.Logf("pool transactions: page=%d last=%d total=%d", page.Page, page.LastPage, page.Total)
	for _, transaction := range page.Items {
		t.Logf("%s: %s %.2f", transaction.ID, transaction.Type, transaction.Credits)
	}
}

func TestLiveSoftware(t *testing.T) {
	client := liveClient(t)
	server := client.Server(requireEnv(t, "EXAROTON_SERVER"))
	ctx := context.Background()

	list, err := client.ListSoftware(ctx, &SoftwareFilter{Type: SoftwareTypeVanilla})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no vanilla software returned")
	}

	for _, software := range list {
		t.Logf("%s: %s %s", software.ID, software.Name, software.Version)
	}

	details, err := server.GetSoftwareDetails(ctx, list[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("software details: %+v", details)
}

func TestLiveServerBackups(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	page, err := server.GetBackups(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("no backup page returned")
	}

	t.Logf("backups: page=%d last=%d total=%d running=%v",
		page.Page, page.LastPage, page.Total, page.Running)
	for _, backup := range page.Items {
		t.Logf("backup %s: status=%s type=%v name=%v size=%v",
			backup.ID, backup.Status, backup.Type, backup.Name, backup.Size)

		got, err := server.GetBackup(ctx, backup.ID)
		if err != nil {
			t.Errorf("get backup %s: %v", backup.ID, err)
			continue
		}
		if got == nil {
			t.Errorf("backup %s returned no data", backup.ID)
			continue
		}
		t.Logf("backup %s: status=%s created=%v", got.ID, got.Status, got.Created)
	}

	accounts, err := server.GetBackupAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range accounts {
		t.Logf("backup account %s: %s provider=%s used=%v limit=%v",
			account.ID, account.Name, account.Provider, account.Storage.Used, account.Storage.Limit)
	}

	settings, err := server.GetBackupSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("backup settings: %+v", settings)
}

func TestLiveServerBackupLifecycle(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	accounts, err := server.GetBackupAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) == 0 {
		t.Skip("no backup account linked to the server")
	}

	backup, err := server.CreateBackup(ctx, &CreateBackupOptions{
		Name: liveName("go-backup"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if backup == nil || backup.ID == "" {
		t.Fatal("create backup returned no ID")
	}

	backupID := backup.ID
	t.Cleanup(func() {
		if backupID == "" {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := server.DeleteBackup(cleanupCtx, backupID); err != nil {
			t.Logf("cleanup: delete backup %s: %v", backupID, err)
		}
	})

	created := waitForBackupCreated(ctx, t, server, backupID)
	t.Logf("created backup: %+v", created)

	renamedName := liveName("go-backup-renamed")
	renamed, err := server.RenameBackup(ctx, backupID, renamedName)
	if err != nil {
		t.Fatal(err)
	}
	if renamed == nil {
		t.Fatal("rename backup returned no data")
	}
	if renamed.Name == nil || *renamed.Name != renamedName {
		t.Fatalf("renamed backup name = %v, want %q", renamed.Name, renamedName)
	}
	t.Logf("renamed backup: %+v", renamed)

	got, err := server.GetBackup(ctx, backupID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("get backup returned no data")
	}
	if got.ID != backupID {
		t.Fatalf("backup ID = %q, want %q", got.ID, backupID)
	}
	t.Logf("backup details: %+v", got)

	if err := server.DeleteBackup(ctx, backupID); err != nil {
		t.Fatal(err)
	}
	backupID = ""
}

func TestLiveUpdateBackupSettings(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	settings, err := server.GetBackupSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings == nil {
		t.Skip("no backup settings returned")
	}
	t.Logf("backup settings: %+v", settings)

	// Send the current values back as a no-op update.
	updated, err := server.UpdateBackupSettings(ctx, UpdateBackupSettingsOptions{
		Enabled:  &settings.Enabled,
		Limit:    &settings.Limit,
		Interval: &settings.Interval,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("updated backup settings: %+v", updated)
}

func TestLiveRestoreBackup(t *testing.T) {
	backupID := requireEnv(t, "EXAROTON_RESTORE_BACKUP")
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	info, err := server.GetServer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("get server returned no data")
	}
	if info.Status != StatusOffline && info.Status != StatusCrashed {
		t.Skipf("server status is %s; restore requires OFFLINE or CRASHED", info.Status)
	}

	backup, err := server.RestoreBackup(ctx, backupID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("restore started: %+v", backup)

	for {
		page, err := server.GetBackups(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !page.Running {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("restore did not finish: %v", ctx.Err())
		case <-time.After(5 * time.Second):
		}
	}
	t.Log("restore finished")
}

func TestLiveServerDomains(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	domains, err := server.GetDomains(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("domains: %d", len(domains))

	for _, domain := range domains {
		got, err := server.GetDomain(ctx, domain.ID)
		if err != nil {
			t.Errorf("get domain %s: %v", domain.ID, err)
			continue
		}
		if got == nil {
			t.Errorf("domain %s returned no data", domain.ID)
			continue
		}
		t.Logf("domain %s: status=%s protocol=%s port=%d nameservers=%v",
			got.Domain, got.Status, got.Protocol, got.Port, got.Nameservers)

		if got.Status == DomainStatusPending {
			checked, err := server.CheckDomain(ctx, got.ID)
			if err != nil {
				t.Logf("check domain %s: %v", got.Domain, err)
				continue
			}
			t.Logf("checked domain %s: status=%s", checked.Domain, checked.Status)
		}
	}
}

func TestLiveServerDomainLifecycle(t *testing.T) {
	domainName := requireEnv(t, "EXAROTON_DOMAIN")
	server := liveServer(t)
	ctx := context.Background()

	existing, err := server.GetDomains(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range existing {
		if strings.EqualFold(domain.Domain, domainName) {
			t.Skipf("%s is already added; remove it first", domainName)
		}
	}

	domain, err := server.AddDomain(ctx, domainName)
	if err != nil {
		t.Fatal(err)
	}
	if domain == nil || domain.ID == "" {
		t.Fatal("add domain returned no ID")
	}

	domainID := domain.ID
	t.Cleanup(func() {
		if domainID == "" {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := server.DeleteDomain(cleanupCtx, domainID); err != nil {
			t.Logf("cleanup: delete domain %s: %v", domainID, err)
		}
	})
	t.Logf("added domain: %+v", domain)

	got, err := server.GetDomain(ctx, domainID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("get domain returned no data")
	}
	t.Logf("domain details: %+v", got)

	port := got.Port
	updated, err := server.UpdateDomain(ctx, domainID, UpdateDomainOptions{
		Protocol: got.Protocol,
		Port:     &port,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("updated domain: %+v", updated)

	checked, err := server.CheckDomain(ctx, domainID)
	if err != nil {
		t.Logf("check domain: %v", err)
	} else {
		t.Logf("checked domain: %+v", checked)
	}

	if err := server.DeleteDomain(ctx, domainID); err != nil {
		t.Fatal(err)
	}
	domainID = ""
}

func TestLiveServerAccess(t *testing.T) {
	server := liveServer(t)
	ctx := context.Background()

	list, err := server.GetAccessList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("shared accounts: %d", len(list))

	for _, access := range list {
		t.Logf("access %s: %s %v", access.ID, access.Account.Name, access.Permissions)

		got, err := server.GetAccess(ctx, access.ID)
		if err != nil {
			t.Errorf("get access %s: %v", access.ID, err)
			continue
		}
		if got == nil {
			t.Errorf("access %s returned no data", access.ID)
			continue
		}
		t.Logf("access %s: %s %v", got.ID, got.Account.Name, got.Permissions)
	}
}

func TestLiveServerAccessLifecycle(t *testing.T) {
	username := requireEnv(t, "EXAROTON_SHARE_USER")
	server := liveServer(t)
	ctx := context.Background()

	list, err := server.GetAccessList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, access := range list {
		if strings.EqualFold(access.Account.Name, username) {
			t.Skipf("%s already has access; remove it first", username)
		}
	}

	access, err := server.ShareServer(ctx, username, []Permission{PermissionServer})
	if err != nil {
		t.Fatal(err)
	}
	if access == nil || access.ID == "" {
		t.Fatal("share server returned no access ID")
	}

	accessID := access.ID
	t.Cleanup(func() {
		if accessID == "" {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := server.RemoveAccess(cleanupCtx, accessID); err != nil {
			t.Logf("cleanup: remove access %s: %v", accessID, err)
		}
	})
	t.Logf("shared server: %+v", access)

	got, err := server.GetAccess(ctx, accessID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("get access returned no data")
	}
	t.Logf("access details: %+v", got)

	updated, err := server.UpdateAccess(ctx, accessID, []Permission{
		PermissionServer,
		PermissionConsole,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil {
		t.Fatal("update access returned no data")
	}
	if len(updated.Permissions) != 2 {
		t.Fatalf("permissions = %v, want 2", updated.Permissions)
	}
	t.Logf("updated access: %+v", updated)

	if err := server.RemoveAccess(ctx, accessID); err != nil {
		t.Fatal(err)
	}
	accessID = ""
}

func TestLiveInstallSoftware(t *testing.T) {
	softwareID := requireEnv(t, "EXAROTON_INSTALL_SOFTWARE")
	server := liveServer(t)
	ctx := context.Background()

	details, err := server.GetSoftwareDetails(ctx, softwareID)
	if err != nil {
		t.Fatal(err)
	}

	var opts []InstallSoftwareOption
	if details.Reinstall {
		if os.Getenv("EXAROTON_ALLOW_REINSTALL") != "1" {
			t.Skip("reinstall required; set EXAROTON_ALLOW_REINSTALL=1 to allow deleting server files")
		}
		opts = append(opts, ConfirmReinstall())
	}

	if err := server.InstallSoftware(ctx, softwareID, opts...); err != nil {
		t.Fatal(err)
	}
	t.Logf("installed software %s on %s", softwareID, server.serverID)
}

func TestLiveServerTransfer(t *testing.T) {
	region := Region(requireEnv(t, "EXAROTON_TRANSFER_REGION"))
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	info, err := server.GetServer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("get server returned no data")
	}
	if info.Status != StatusOffline && info.Status != StatusCrashed {
		t.Skipf("server status is %s; transfer requires OFFLINE or CRASHED", info.Status)
	}

	transferred, err := server.TransferServer(ctx, region)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("transfer started: status=%s region=%s", transferred.Status, transferred.Region)
}

func TestLiveCreateDeleteServer(t *testing.T) {
	requireEnv(t, "EXAROTON_CREATE_SERVER")
	client := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	region := RegionEU
	if value := os.Getenv("EXAROTON_CREATE_REGION"); value != "" {
		region = Region(value)
	}

	created, err := client.CreateServer(ctx, CreateServerOptions{
		Name:         liveName("sdk-live"),
		SoftwareType: SoftwareTypeVanilla,
		Region:       region,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID == "" {
		t.Fatal("create server returned no ID")
	}

	serverID := created.ID
	server := client.Server(serverID)
	t.Cleanup(func() {
		if serverID == "" {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := server.DeleteServer(cleanupCtx); err != nil {
			t.Logf("cleanup: delete server %s: %v", serverID, err)
		}
	})
	t.Logf("created server: %+v", created)

	deadline := time.Now().Add(10 * time.Minute)
	for {
		info, err := server.GetServer(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if info == nil {
			t.Fatalf("server %s returned no data", serverID)
		}
		t.Logf("server %s status: %s", serverID, info.Status)
		if info.Status == StatusOffline || info.Status == StatusCrashed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server %s did not become offline; delete it manually", serverID)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("server %s did not become offline: %v", serverID, ctx.Err())
		case <-time.After(5 * time.Second):
		}
	}

	if err := server.DeleteServer(ctx); err != nil {
		t.Fatal(err)
	}
	serverID = ""
}

// waitForWSMessage waits for a value on a websocket session channel while
// surfacing session errors and honoring the context deadline.
func waitForWSMessage[T any](ctx context.Context, t *testing.T, name string, ch <-chan T, errCh <-chan error) T {
	t.Helper()

	for {
		select {
		case value, ok := <-ch:
			if !ok {
				t.Fatalf("%s channel closed before a message", name)
			}
			return value
		case err, ok := <-errCh:
			if ok && err != nil {
				t.Fatalf("%s stream error: %v", name, err)
			}
			// The error channel is closed; do not select it again.
			errCh = nil
		case <-ctx.Done():
			var zero T
			t.Fatalf("timed out waiting for %s: %v", name, ctx.Err())
			return zero
		}
	}
}

func TestLiveWSStatus(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	msg := waitForWSMessage(ctx, t, "status", ws.StatusChan, ws.ErrorChan)
	t.Log(msg)
}

func TestLiveWSConsole(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeConsole(ctx, 3); err != nil {
		t.Fatal(err)
	}

	line := waitForWSMessage(ctx, t, "console", ws.ConsoleChan, ws.ErrorChan)
	t.Log(line)
}

func TestLiveWSTick(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeTick(ctx); err != nil {
		t.Fatal(err)
	}

	tick := waitForWSMessage(ctx, t, "tick", ws.TickChan, ws.ErrorChan)
	t.Log(tick)
}

func TestLiveWSStats(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeStats(ctx); err != nil {
		t.Fatal(err)
	}

	stats := waitForWSMessage(ctx, t, "stats", ws.StatsChan, ws.ErrorChan)
	t.Log(stats)
}

func TestLiveWSHeap(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeHeap(ctx); err != nil {
		t.Fatal(err)
	}

	heap := waitForWSMessage(ctx, t, "heap", ws.HeapChan, ws.ErrorChan)
	t.Log(heap)
}

func TestLiveWSManagement(t *testing.T) {
	server := liveServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ws, err := server.OpenWebSocket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeManagement(ctx); err != nil {
		t.Fatal(err)
	}

	players, err := ws.GetAllowlist(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Old: %v", players)

	newPlayers := []Player{
		{Name: "Jiternos", UUID: "981853f1-a3f4-4f5a-808a-1a0ff47a5110"},
	}
	if newResult, err := ws.AddAllowlist(ctx, newPlayers); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("New: %v", newResult)
	}
}
