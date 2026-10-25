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
