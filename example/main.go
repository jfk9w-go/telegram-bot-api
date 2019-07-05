package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
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
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// First read the token from command line arguments.
	token := os.Args[1]
	// Create a bot instance.
	bot := telegram.NewBot(nil, token)
	// Listen to the commands. Blocks until Bot.Close() is called.
	// Can be launched in a separate goroutine.
	bot.Listen(telegram.NewCommandListener(bot).
		HandleFunc("/greet", func(c *telegram.Command) error {
			c.Reply("Hello, " + c.User.FirstName)
			return nil
		}).
		HandleFunc("/tick", func(c *telegram.Command) error {
			_, err := bot.Send(c.Chat.ID,
				&telegram.Media{
					Type:      telegram.Photo,
					Resource:  flu.NewFileSystemResource("tick.png"),
					Caption:   "Here's a <b>tick</b> for ya.",
					ParseMode: telegram.HTML,
				},
				&telegram.SendOpts{DisableNotification: true})

			return err
		}).
		HandleFunc("/ticks", func(c *telegram.Command) error {
			media := make([]telegram.Media, 4)
			for i := range media {
				media[i] = telegram.Media{
					Type:     telegram.Photo,
					Resource: flu.NewFileSystemResource("tick.png"),
					Caption:  "Image " + strconv.Itoa(i),
				}
			}

			_, err := bot.SendMediaGroup(c.Chat.ID, media, &telegram.SendOpts{DisableNotification: true})
			return err
		}).
		HandleFunc("/count", func(c *telegram.Command) error {
			limit, err := strconv.Atoi(c.Payload)
			if err != nil {
				return err
			}

			if limit <= 0 {
				return errors.New("limit must be positive")
			}

			for i := 1; i <= limit; i++ {
				c.Reply(strconv.Itoa(i))
			}

			return nil
		}).
		HandleFunc("/secret", func(c *telegram.Command) error {
			fields := strings.Fields(c.Payload)
			if len(fields) != 2 {
				return errors.New("usage: /secret Hi 5")
			}

			timeoutSecs, err := strconv.Atoi(fields[1])
			if err != nil {
				return err
			}

			timeout := time.Duration(timeoutSecs) * time.Second
			m, err := bot.Send(c.Chat.ID, &telegram.Text{Text: fields[0]}, nil)
			if err != nil {
				return err
			}

			time.AfterFunc(timeout, func() {
				ok, err := bot.DeleteMessage(m.Chat.ID, m.ID)
				if err != nil {
					c.Reply(err.Error())
					return
				}

				println("Message deleted:", ok)
			})

			return nil
		}).
		HandleFunc("/say", func(c *telegram.Command) error {
			if c.Payload == "" {
				return errors.New("please specify a message")
			}

			_, err := bot.Send(c.Chat.ID,
				&telegram.Text{Text: "Here you go."},
				&telegram.SendOpts{ReplyMarkup: telegram.CommandButton("Say "+c.Payload, "say", c.Payload)})

			if err != nil {
				return err
			}

			return nil
		}).
		HandleFunc("say", func(c *telegram.Command) error {
			c.Reply(c.Payload)
			return nil
		}))
}
