# telegram-bot-api

[![GoDoc](https://godoc.org/github.com/jfk9w-go/telegram-bot-api?status.svg)](https://godoc.org/github.com/jfk9w-go/telegram-bot-api) [![ci](https://github.com/jfk9w-go/telegram-bot-api/actions/workflows/ci.yml/badge.svg)](https://github.com/jfk9w-go/telegram-bot-api/actions/workflows/ci.yml) 

**telegram-bot-api** is a simple Telegram Bot API client and bot implementation.

## Disclaimer

Not all API methods, options and types are implemented at the moment.

## Features
* Simple and concise Telegram Bot API client interface.
* Flood control aware built-in API call throttler.
* Update and command listener support.

## Installation
Simply install the package via go get:
```bash
go get -u github.com/jfk9w-go/telegram-bot-api
```

## Example

```go
package main

import (
	"context"
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api"
)

func main() {
	// read token from command line arguments
	token := os.Args[1]
	
	// define a listener
	listener := make(telegram.CommandRegistry).
		AddFunc("/ping", func(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
			return cmd.Reply(ctx, client, "pong")
		})
	
	defer telegram.
		// create bot instance
		NewBot(context.Background(), nil, token).
		// start command listener
		CommandListener(listener).
		// defer shutdown
		Close()
	
	// wait for signal
	flu.AwaitSignal()
}
```

You can find an extended example [here](https://github.com/jfk9w-go/telegram-bot-api/blob/master/example/main.go).
In order to launch simply do:
```bash
cd examples/ && go run main.go <your_bot_api_token>
```
