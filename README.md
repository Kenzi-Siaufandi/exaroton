# exaroton (Go)

Go client for the [exaroton](https://exaroton.com) API.

## API key

Generate a token at <https://exaroton.com/account/settings/> and pass it to
`NewClient`.

## Install

```sh
go get github.com/Kenzi-Siaufandi/exaroton
```

Requires Go 1.25+.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenzi-Siaufandi/exaroton"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := exaroton.NewClient("YOUR_API_TOKEN")
	if err != nil {
		panic(err)
	}

	server := client.Server("EwYiY9IAMtQBTb6U")
	info, err := server.GetServer(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(info.Name, info.Status)
}
```

## Websocket example

```go
session, err := server.OpenWebSocket(ctx)
if err != nil {
	panic(err)
}
defer session.Close()

if err := session.SubscribeConsole(ctx, 5); err != nil {
	panic(err)
}
for line := range session.ConsoleChan {
	fmt.Println(line)
}
```