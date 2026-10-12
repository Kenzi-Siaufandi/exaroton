package exaroton

import (
	"context"
	"os"
	"testing"
)

var (
	envToken        = os.Getenv("EXAROTON_TOKEN")
	envServerID     = os.Getenv("EXAROTON_SERVER")
	envCreditPoolID = os.Getenv("EXAROTON_CREDIT_POOL")
)

// requireToken skips live integration tests when no API token is configured.
func requireToken(t *testing.T) {
	t.Helper()
	if envToken == "" {
		t.Skip("EXAROTON_TOKEN not set; live integration test")
	}
}

// requireServerID returns the server ID under test, skipping when not configured.
func requireServerID(t *testing.T) string {
	t.Helper()
	requireToken(t)
	if envServerID == "" {
		t.Skip("EXAROTON_SERVER not set; live integration test")
	}
	return envServerID
}

// requireCreditPoolID returns the credit pool ID under test, skipping when not configured.
func requireCreditPoolID(t *testing.T) string {
	t.Helper()
	requireToken(t)
	if envCreditPoolID == "" {
		t.Skip("EXAROTON_CREDIT_POOL not set; live integration test")
	}
	return envCreditPoolID
}

func TestAccount(t *testing.T) {
	requireToken(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	account, err := client.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(account)
}

func TestServers(t *testing.T) {
	requireToken(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	servers, err := client.GetServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(servers)
}

func TestServer(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)

	serverInfo, err := server.GetServer(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Log(serverInfo)
}

func TestServerLogs(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
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
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
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
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
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
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	err = server.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerStop(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	err = server.Stop(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerRestart(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	err = server.Restart(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerCommand(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	err = server.ExecuteCommand(ctx, "say Hello World")
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerExtendTime(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	err = server.ExtendWait(ctx, 120)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerPlayerList(t *testing.T) {
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
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
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
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
	serverID := requireServerID(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	server := client.Server(serverID)
	ctx := context.Background()

	propsFile := server.File("velocity.toml")
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
	requireToken(t)
	client, err := NewClient(envToken)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	allPools, err := client.ListCreditPools(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(allPools)

	poolID := requireCreditPoolID(t)
	poolRef := client.CreditPool(poolID)
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
