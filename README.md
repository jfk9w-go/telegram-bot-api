# telegram-bot-api

[![GoDoc](https://godoc.org/github.com/jfk9w-go/telegram-bot-api?status.svg)](https://godoc.org/github.com/jfk9w-go/telegram-bot-api) [![Build Status](https://travis-ci.com/jfk9w-go/telegram-bot-api.svg?branch=master)](https://travis-ci.com/jfk9w-go/telegram-bot-api) 

**telegram-bot-api** is a simple Telegram Bot API client and bot implementation.

## Disclaimer

Not all API methods, options and types are implemented at the moment.

## Features
* Simple and concise Telegram Bot API client interface.
* Flood control aware build-in API call throttler.
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
	"os"
	
	"github.com/jfk9w-go/telegram-bot-api"
)

func main() {
    // First read the token from command line arguments.
    token := os.Args[1]
    // Create a bot instance.
    bot := telegram.NewBot(nil, token)
    // Run command listeners.
    bot.Listen(telegram.NewCommandUpdateListener().
        AddFunc("/ping", func(c *telegram.Command) {
            c.TextReply("pong")
        }))
}
```

You can find an extended example [here](https://github.com/jfk9w-go/telegram-bot-api/blob/master/example/main.go).
In order to launch simply do:
```bash
cd examples/ && go run main.go <your_bot_api_token>
```