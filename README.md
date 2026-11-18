[![Go Reference](https://pkg.go.dev/badge/github.com/Kenzi-Siaufandi/exaroton.svg)](https://pkg.go.dev/github.com/Kenzi-Siaufandi/exaroton)
[![License](https://img.shields.io/github/license/Kenzi-Siaufandi/exaroton)](LICENSE)

# exaroton (Go)

Go client for the [exaroton](https://exaroton.com) API. Requires Go 1.25+.

## Install

```sh
go get github.com/Kenzi-Siaufandi/exaroton
```

## Quick Start

Generate an API token at [exaroton Settings](https://exaroton.com/account/settings/) and export it:

```sh
export EXAROTON_TOKEN="your-token"
```

Initialize the client and context

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

client, err := exaroton.NewClient(os.Getenv("EXAROTON_TOKEN"))
if err != nil {
	log.Fatal(err)
}

server := client.Server("SERVER_ID")
```

---

## Usage

### Create a Server

`Name` is required. Provide either `Software` (specific ID) or `SoftwareType` (latest version); `Software` takes precedence if both are set.

```go
created, err := client.CreateServer(ctx, exaroton.CreateServerOptions{
	Name:         "my-server",                  // required
	SoftwareType: exaroton.SoftwareTypeVanilla, // optional: latest version of a type
	// Software:  "wdZhC3v1udSOxR1d",           // optional: specific software ID
	MOTD:         "Cool server!",               // optional: defaults to generated MOTD
	Timezone:     "Asia/Jakarta",               // optional: defaults to UTC
	Region:       exaroton.RegionEU,            // optional: RegionEU or RegionNA
})
if err != nil {
	log.Fatal(err)
}

// Use the new server for further operations.
createdServer := client.Server(created.ID)
```

### Server Lifecycle

```go
info, err := server.GetServer(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(info.Name, info.Status)

if err := server.Start(ctx); err != nil {
	log.Fatal(err)
}
if err := server.Stop(ctx); err != nil {
	log.Fatal(err)
}
if err := server.Restart(ctx); err != nil {
	log.Fatal(err)
}

// Permanent deletion requires the server to be completely offline or crashed
// Also, can only be performed by the server owner.
if err := server.DeleteServer(ctx); err != nil {
	log.Fatal(err)
}
```

### Server Software

```go
list, err := client.ListSoftware(ctx, &exaroton.SoftwareFilter{
	Type:     exaroton.SoftwareTypeModpacks,
	Launcher: exaroton.LauncherCurseForge, // CurseForge, Modrinth, FTB, ATL, Technic
})
if err != nil {
	log.Fatal(err)
}
if len(list) == 0 {
	log.Fatal("no software found")
}

softwareID := list[0].ID

details, err := server.GetSoftwareDetails(ctx, softwareID)
if err != nil {
	log.Fatal(err)
}

var opts []exaroton.InstallSoftwareOption
if details.Reinstall {
	opts = append(opts, exaroton.ConfirmReinstall()) // required if existing files will be deleted
}

if err := server.InstallSoftware(ctx, softwareID, opts...); err != nil {
	log.Fatal(err)
}
```

### Regions & Transfers

```go
regions, err := client.ListRegions(ctx)
if err != nil {
	log.Fatal(err)
}
for _, region := range regions {
	fmt.Println(region.ID, region.Name)
}

// The server must be offline or crashed. Transfers run asynchronously:
// the returned server is in status TRANSFERRING until the files are moved.
transferred, err := server.TransferServer(ctx, exaroton.RegionNA)
if err != nil {
	log.Fatal(err)
}
fmt.Println(transferred.Status) // 9 (TRANSFERRING)
```

### Server Properties

Properties are partial: fields the server does not support (or your account
cannot change) are `nil`. When updating, `nil` fields are left unchanged and
pointers let you send `false` or `0` explicitly.

```go
props, err := server.GetProperties(ctx)
if err != nil {
	log.Fatal(err)
}
if props.AutoStop != nil && *props.AutoStop && props.AutoStopTime != nil {
	fmt.Println("auto stop after", *props.AutoStopTime, "minutes")
}

autoStop := false
minutes := 15
updated, err := server.UpdateProperties(ctx, exaroton.ServerProperties{
	AutoStop:     &autoStop,
	AutoStopTime: &minutes,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(updated)
```

### Config Files

Config files are files that contain server configuration and are shown as a
form in the exaroton website instead of the plain-text file editor. Use
`File.IsConfigFile` from `Info` to check support before reading or changing
options.

> **Note:** only files that open in the website's config screen are supported.
> proxy configuration files and Hytale configuration files are
> not yet implemented.

For example, `server.properties`:

```go
file := server.File("server.properties")

info, err := file.Info(ctx)
if err != nil {
	log.Fatal(err)
}
if !info.IsConfigFile {
	log.Fatal("server.properties has no config screen")
}

options, err := file.Config(ctx)
if err != nil {
	log.Fatal(err)
}
for _, option := range options {
	fmt.Printf("%s = %v (%s)\n", option.Key, option.Value, option.Type)

	// Min, Max and Step are only set for integer and float options.
	if option.Min != nil && option.Max != nil {
		fmt.Printf("  allowed: %v to %v", *option.Min, *option.Max)
		if option.Step != nil {
			fmt.Printf(", step %v", *option.Step)
		}
		fmt.Println()
	}
}
```

Validate numeric values against the server-provided bounds, then save them.
`Step` is only a suggestion for UI controls and is not enforced by the API.

```go
maxPlayers := 20
for _, option := range options {
	if option.Key != "max-players" {
		continue
	}
	value := float64(maxPlayers)
	if option.Min != nil && value < *option.Min {
		log.Fatalf("max-players must be at least %v", *option.Min)
	}
	if option.Max != nil && value > *option.Max {
		log.Fatalf("max-players must be at most %v", *option.Max)
	}
}

updated, err := file.UpdateConfigValues(ctx, map[string]any{
	"max-players": maxPlayers,
	"pvp":         false,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(updated)
```

`UpdateConfigValues` accepts typed values so integers, booleans and
multiselect arrays keep their JSON types. `UpdateConfig` is a string-only
wrapper for the same endpoint. Files without a config screen must be edited as
raw files with `Read` and `Write`.

### Backups

```go
page, err := server.GetBackups(ctx, 1)
if err != nil {
	log.Fatal(err)
}
for _, backup := range page.Items {
	fmt.Println(backup.ID, backup.Status)
}

created, err := server.CreateBackup(ctx, &exaroton.CreateBackupOptions{
	Name: "Before the update", // optional
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(created.Status) // creating

// Poll GetBackups (or page.Running) before starting the server again.
if _, err := server.RestoreBackup(ctx, created.ID); err != nil {
	log.Fatal(err)
}

accounts, err := server.GetBackupAccounts(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(accounts)

interval := exaroton.BackupInterval24Hours
enabled := true
settings, err := server.UpdateBackupSettings(ctx, exaroton.UpdateBackupSettingsOptions{
	Enabled:  &enabled,
	Interval: &interval,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(settings)
```

### Custom Domains

```go
domain, err := server.AddDomain(ctx, "play.example.com")
if err != nil {
	log.Fatal(err)
}
fmt.Println(domain.Nameservers) // point your NS records here

domain, err = server.CheckDomain(ctx, domain.ID)
if err != nil {
	log.Fatal(err)
}
if domain.Status == exaroton.DomainStatusLinked {
	fmt.Println("domain is linked")
}

port := 8080
domain, err = server.UpdateDomain(ctx, domain.ID, exaroton.UpdateDomainOptions{
	Protocol: exaroton.DomainProtocolHTTPS,
	Port:     &port,
})
if err != nil {
	log.Fatal(err)
}

if err := server.DeleteDomain(ctx, domain.ID); err != nil {
	log.Fatal(err)
}
```

### Share Access

```go
shared, err := server.GetAccessList(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(shared)

access, err := server.ShareServer(ctx, "username", []exaroton.Permission{
	exaroton.PermissionServer,
	exaroton.PermissionStop,
	exaroton.PermissionConsole,
})
if err != nil {
	log.Fatal(err)
}

access, err = server.UpdateAccess(ctx, access.ID, []exaroton.Permission{
	exaroton.PermissionServer,
	exaroton.PermissionStop,
})
if err != nil {
	log.Fatal(err)
}

if err := server.RemoveAccess(ctx, access.ID); err != nil {
	log.Fatal(err)
}
```

### Transactions

Transactions are paginated (20 per page). Page `0` requests the first page.

```go
page, err := client.GetAccountTransactions(ctx, 1)
if err != nil {
	log.Fatal(err)
}
for _, tx := range page.Items {
	fmt.Printf("%s: %.2f credits\n", tx.Type, tx.Credits)
}

page, err = server.GetTransactions(ctx, 1) // requires the transactions permission
if err != nil {
	log.Fatal(err)
}

page, err = client.CreditPool("POOL_ID").GetTransactions(ctx, 1)
if err != nil {
	log.Fatal(err)
}
```

### Live Console (WebSocket)

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

session, err := server.OpenWebSocket(ctx)
if err != nil {
	log.Fatal(err)
}
defer session.Close()

if err := session.SubscribeConsole(ctx, 100); err != nil {
	log.Fatal(err)
}

for {
	select {
	case line, ok := <-session.ConsoleChan:
		if !ok {
			return
		}
		fmt.Println(line)
	case <-ctx.Done():
		return
	}
}
```

## Documentation

Full API reference: [pkg.go.dev/github.com/Kenzi-Siaufandi/exaroton](https://pkg.go.dev/github.com/Kenzi-Siaufandi/exaroton)

## License

MIT © [Kenzi Siaufandi](LICENSE)
