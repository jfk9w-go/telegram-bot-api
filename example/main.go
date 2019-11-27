package main

import (
	"fmt"
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
	go func() { log.Println(http.ListenAndServe("localhost:6060", nil)) }()

	// First read the token and proxy from command line arguments.
	token := os.Args[1]
	proxy := ""
	if len(os.Args) > 2 {
		proxy = os.Args[2]
	}

	// Create a bot instance.
	bot := telegram.NewBot(flu.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		ProxyURL(proxy).
		NewClient(), token)

	// Listen to the commands. Blocks until Bot.Close() is called.
	// Can be launched in a separate goroutine.
	bot.Listen(telegram.NewCommandListener().
		HandleFunc("/greet", func(tg telegram.Client, c *telegram.Command) error {
			_, err := tg.Send(c.Chat.ID,
				&telegram.Text{Text: "Hello, " + c.User.FirstName},
				&telegram.SendOptions{ReplyToMessageID: c.MessageID})

			return err
		}).
		HandleFunc("/tick", func(tg telegram.Client, c *telegram.Command) error {
			_, err := tg.Send(c.Chat.ID,
				&telegram.Media{
					Type:      telegram.Photo,
					Resource:  flu.File("tick.png"),
					Caption:   "Here's a <b>tick</b> for ya.",
					ParseMode: telegram.HTML},
				&telegram.SendOptions{DisableNotification: true})

			return err
		}).
		HandleFunc("/ticks", func(tg telegram.Client, c *telegram.Command) error {
			media := make([]telegram.Media, 4)
			for i := range media {
				media[i] = telegram.Media{
					Type:     telegram.Photo,
					Resource: flu.File("tick.png"),
					Caption:  "Image " + strconv.Itoa(i)}
			}

			_, err := tg.SendMediaGroup(c.Chat.ID, media, &telegram.SendOptions{DisableNotification: true})
			return err
		}).
		HandleFunc("/count", func(tg telegram.Client, c *telegram.Command) error {
			limit, err := strconv.Atoi(c.Payload)
			if err == nil {
				if limit <= 0 {
					err = errors.New("limit must be positive")
				}
			}

			if err != nil {
				_, err = tg.Send(c.Chat.ID,
					&telegram.Text{Text: err.Error()},
					&telegram.SendOptions{DisableNotification: true})

				return err
			}

			for i := 1; i <= limit; i++ {
				_, err := tg.Send(c.Chat.ID, &telegram.Text{Text: fmt.Sprintf("%d", i)}, nil)
				if err != nil {
					return err
				}
			}

			return nil
		}).
		HandleFunc("/secret", func(tg telegram.Client, c *telegram.Command) error {
			var err error
			fields := strings.Fields(c.Payload)
			if len(fields) != 2 {
				err = errors.New("usage: /secret Hi 5")
			}

			var timeout time.Duration
			if err == nil {
				var secs int
				secs, err = strconv.Atoi(fields[1])
				if err == nil {
					if secs <= 0 {
						err = errors.New("secs must be positive")
					} else {
						timeout = time.Duration(secs) * time.Second
					}
				}
			}

			if err != nil {
				_, err := tg.Send(c.Chat.ID,
					&telegram.Text{Text: err.Error()},
					&telegram.SendOptions{ReplyToMessageID: c.MessageID})

				return err
			}

			m, err := tg.Send(c.Chat.ID, &telegram.Text{Text: fields[0]}, nil)
			if err != nil {
				return err
			}

			// executing in a separate goroutine anyway
			// we don't expect that much of a load
			time.Sleep(timeout)

			ok, err := tg.DeleteMessage(m.Chat.ID, m.ID)
			log.Printf("Message deleted: %v", ok)
			return err
		}).
		HandleFunc("/say", func(tg telegram.Client, c *telegram.Command) error {
			if c.Payload == "" {
				_, err := tg.Send(c.Chat.ID,
					&telegram.Text{Text: "Please specify a message"},
					&telegram.SendOptions{ReplyToMessageID: c.MessageID})

				return err
			}

			_, err := tg.Send(c.Chat.ID,
				&telegram.Text{Text: "Here you go."},
				&telegram.SendOptions{ReplyMarkup: telegram.CommandButton("Say "+c.Payload, "say", c.Payload)})

			return err
		}).
		HandleFunc("say", func(tg telegram.Client, c *telegram.Command) error {
			if c.CallbackQueryID == "" {
				return errors.New("callback query ID is nil")
			}

			_, err := tg.AnswerCallbackQuery(c.CallbackQueryID,
				&telegram.AnswerCallbackQueryOptions{Text: c.Payload})

			return err
		}))
}
