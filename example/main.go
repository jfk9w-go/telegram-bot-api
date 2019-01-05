package main

import (
	"log"
	"os"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

// This is an example bot which has three commands:
//   /greet <name> - reply with "Hello, <name>"
//   /greet_me - reply with "Hello, <username>"
//   /tick - reply with an image of a tick
//   /quit - causes bot instance to shutdown
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
		AddFunc("/quit", func(c *telegram.Command) {
			c.TextReply("Quit. You will have to relaunch the bot manually.")
			bot.Close()
		}))

	log.Printf("Application exit")
}
