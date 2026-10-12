[![Go Reference](https://pkg.go.dev/badge/github.com/Kenzi-Siaufandi/exaroton.svg)](https://pkg.go.dev/github.com/Kenzi-Siaufandi/exaroton)
[![License](https://img.shields.io/github/license/Kenzi-Siaufandi/exaroton)](LICENSE)

# exaroton (Go)

Go client for the [exaroton](https://exaroton.com) API.

## Install

```sh
go get github.com/Kenzi-Siaufandi/exaroton
```

Requires Go 1.25+.

## API key

Generate a token at <https://exaroton.com/account/settings/> and pass it to
`NewClient`. The examples read it from an environment variable:

```sh
export EXAROTON_TOKEN="your-token"
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Kenzi-Siaufandi/exaroton"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := exaroton.NewClient(os.Getenv("EXAROTON_TOKEN"))
	if err != nil {
		panic(err)
	}

	server := client.Server("EXAMPLE")
	info, err := server.GetServer(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(info.Name, info.Status)
}
```

## Websocket
Using the `server` from the example above:
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

session, err := server.OpenWebSocket(ctx)
if err != nil {
	log.Fatal(err)
}
defer session.Close()

if err := session.SubscribeConsole(ctx, 5); err != nil {
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

Full API reference: https://pkg.go.dev/github.com/Kenzi-Siaufandi/exaroton

## License

MIT, see [LICENSE](LICENSE).
