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
	proxy := ""
	if len(os.Args) > 3 {
		proxy = os.Args[3]
	}
	// Create a bot instance.
	telegram.NewBot(flu.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		ProxyURL(proxy).
		NewClient(), os.Args[1]).
		// Listen to the commands. Blocks until Bot.Close() is called.
		// Can be launched in a separate goroutine.
		Listen(2, telegram.NewCommandListener(os.Args[2]).
			HandleFunc("/greet", func(tg telegram.Client, cmd *telegram.Command) error {
				_, err := tg.Send(cmd.Chat.ID,
					&telegram.Text{Text: "Hello, " + cmd.User.FirstName},
					&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID})
				return err
			}).
			HandleFunc("/tick", func(tg telegram.Client, cmd *telegram.Command) error {
				_, err := tg.Send(cmd.Chat.ID,
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
			HandleFunc("/count", func(tg telegram.Client, cmd *telegram.Command) error {
				limit, err := strconv.Atoi(cmd.Payload)
				if err != nil || limit <= 0 {
					return cmd.Reply(tg, "limit must be a positive integer")
				}
				for i := 1; i <= limit; i++ {
					_, err := tg.Send(cmd.Chat.ID, &telegram.Text{Text: fmt.Sprintf("%d", i)}, nil)
					if err != nil {
						return err
					}
				}
				return nil
			}).
			HandleFunc("/secret", func(tg telegram.Client, cmd *telegram.Command) error {
				fields := strings.Fields(cmd.Payload)
				if len(fields) != 2 {
					return cmd.Reply(tg, "usage: /secret Hi 5")
				}
				secs, err := strconv.Atoi(fields[1])
				if err != nil || secs <= 0 {
					return cmd.Reply(tg, "secs must be a positive integer")
				}
				timeout := time.Duration(secs) * time.Second
				m, err := tg.Send(cmd.Chat.ID, &telegram.Text{Text: fields[0]}, nil)
				if err != nil {
					return err
				}
				time.Sleep(timeout)
				ok, err := tg.DeleteMessage(m.Chat.ID, m.ID)
				log.Printf("Message deleted: %v", ok)
				return err
			}).
			HandleFunc("/say", func(tg telegram.Client, cmd *telegram.Command) error {
				if cmd.Payload == "" {
					return cmd.Reply(tg, "Please specify a message")
				}
				_, err := tg.Send(cmd.Chat.ID,
					&telegram.Text{Text: "Here you go."},
					&telegram.SendOptions{
						ReplyMarkup: telegram.InlineKeyboard(
							[][3]string{
								{"Say " + cmd.Payload, "say", cmd.Payload},
								{"Another button", "", ""}}),
					})
				return err
			}).
			HandleFunc("say", func(tg telegram.Client, cmd *telegram.Command) error {
				return cmd.Reply(tg, cmd.Payload)
			}).
			HandleFunc("/question", func(tg telegram.Client, cmd *telegram.Command) error {
				reply, err := tg.Ask(cmd.Chat.ID,
					&telegram.Text{Text: "Your question is, " + cmd.Payload},
					&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID})
				if err != nil {
					return cmd.Reply(tg, err.Error())
				}
				_, err = tg.Send(reply.Chat.ID,
					&telegram.Text{Text: "Your answer is, " + reply.Text},
					&telegram.SendOptions{ReplyToMessageID: reply.ID})
				if err != nil {
					return cmd.Reply(tg, err.Error())
				}
				return nil
			}))
}
