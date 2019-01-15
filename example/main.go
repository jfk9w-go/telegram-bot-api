package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

// This is an example bot which has three commands:
//   /greet - reply with "Hello, %username%"
//   /count n - count from 1 till n
//   /tick - tick
//   /secret text s - send a text and erase the message in s seconds
//
// You can launch this example by simply doing:
//   cd example/ && go run main.go <token>
// where <token> is your Telegram Bot API token.
func main() {
	// First read the token from command line arguments.
	token := os.Args[1]
	// Create a bot instance.
	bot := telegram.NewBot(nil, token)
	// Listen to the commands. Blocks until Bot.Close() is called.
	// Can be launched in a separate goroutine.
	bot.Listen(telegram.NewCommandUpdateListener(bot).
		AddFunc("/greet", func(c *telegram.Command) {
			c.TextReply("Hello, " + c.User.FirstName)
		}).
		AddFunc("/tick", func(c *telegram.Command) {
			_, err := bot.Send(c.Chat.ID, flu.NewFileSystemResource("tick.png"), telegram.NewSendOpts().
				DisableNotification(true).
				ParseMode(telegram.HTML).
				Media().
				Caption("Here's a <b>tick</b> for ya.").
				Photo())
			if err != nil {
				log.Printf("Failed to send tick.png to %d: %s", c.Chat.ID, err)
			}
		}).
		AddFunc("/count", func(c *telegram.Command) {
			limit, err := strconv.Atoi(c.Payload)
			if err != nil {
				c.ErrorReply(err)
				return
			}

			if limit <= 0 {
				c.TextReply("limit must be positive")
				return
			}

			for i := 1; i <= limit; i++ {
				c.TextReply(strconv.Itoa(i))
			}
		}).
		AddFunc("/secret", func(c *telegram.Command) {
			fields := strings.Fields(c.Payload)
			if len(fields) != 2 {
				c.TextReply("usage: /secret Hi 5")
				return
			}

			timeoutSecs, err := strconv.Atoi(fields[1])
			if err != nil {
				c.ErrorReply(err)
				return
			}

			timeout := time.Duration(timeoutSecs) * time.Second
			m, err := bot.Send(c.Chat.ID, fields[0], telegram.NewSendOpts().Message())
			if err != nil {
				c.ErrorReply(err)
				return
			}

			time.AfterFunc(timeout, func() {
				ok, err := bot.DeleteMessage(m.Chat.ID, m.ID)
				if err != nil {
					c.ErrorReply(err)
					return
				}

				log.Println("Message deleted:", ok)
			})
		}).
		AddFunc("/say", func(c *telegram.Command) {
			if c.Payload == "" {
				c.TextReply("please specify a message")
				return
			}

			_, err := bot.Send(c.Chat.ID, "Here you go.", telegram.NewSendOpts().
				Message().
				ReplyMarkup(telegram.CommandButton("Say "+c.Payload, "say", c.Payload)))
			if err != nil {
				log.Println("Failed to send button", err)
			}
		}).
		AddFunc("say", func(c *telegram.Command) {
			c.TextReply(c.Payload)
		}))
}
