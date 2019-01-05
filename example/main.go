package main

import (
	"os"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

func main() {
	token := os.Args[1]
	bot := telegram.NewBot(nil, token)
	bot.Listen(telegram.NewCommandUpdateListener().
		AddFunc("/greet", func(c *telegram.Command) {
			if c.Payload == "" {
				c.TextReply("Please enter a name.")
			} else {
				c.TextReply("Hello, " + c.Payload)
			}
		}).
		AddFunc("/greet_me", func(c *telegram.Command) {
			c.TextReply("Hello, " + c.User.FirstName)
		}))
}
