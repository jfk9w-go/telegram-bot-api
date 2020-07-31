package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

func CommandListenerFunc(ctx context.Context, bot telegram.Client, cmd telegram.Command) (err error) {
	switch cmd.Key {
	case "/greet":
		_, err = bot.Send(ctx, cmd.Chat.ID,
			telegram.Text{Text: "Hello, " + cmd.User.FirstName},
			&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID})
	case "/tick":
		_, err = bot.Send(ctx, cmd.Chat.ID,
			telegram.Media{
				Type:      telegram.MediaTypeByMIMEType("image/jpeg"),
				Input:     flu.File("tick.png"),
				Caption:   "Here's a <b>tick</b> for ya.",
				ParseMode: telegram.HTML},
			&telegram.SendOptions{DisableNotification: true})
	case "/gif":
		media := telegram.Media{
			Type:    telegram.MediaTypeByMIMEType("image/gif"),
			Input:   flu.File("gif.gif"),
			Caption: "GIF",
		}
		_, err = bot.Send(ctx, cmd.Chat.ID, media, &telegram.SendOptions{DisableNotification: true})
	case "/webp":
		_, err = bot.Send(ctx, cmd.Chat.ID,
			telegram.Media{
				Type:  telegram.MediaTypeByMIMEType("image/webp"),
				Input: flu.File("webp.webp"),
			},
			&telegram.SendOptions{DisableNotification: true})
	case "/count":
		var limit int
		limit, err = strconv.Atoi(cmd.Payload)
		if err != nil || limit <= 0 {
			return cmd.Reply(ctx, bot, "limit must be a positive integer")
		}
		for i := 1; i <= limit; i++ {
			_, err = bot.Send(ctx, cmd.Chat.ID, telegram.Text{Text: fmt.Sprintf("%d", i)}, nil)
			if err != nil {
				return cmd.Reply(ctx, bot, err.Error())
			}
		}
	case "/secret":
		fields := strings.Fields(cmd.Payload)
		if len(fields) != 2 {
			return cmd.Reply(ctx, bot, "usage: /secret Hi 5")
		}
		var secs int
		if secs, err = strconv.Atoi(fields[1]); err != nil || secs <= 0 {
			return cmd.Reply(ctx, bot, "secs must be a positive integer")
		}
		timeout := time.Duration(secs) * time.Second
		var m *telegram.Message
		if m, err = bot.Send(ctx, cmd.Chat.ID, telegram.Text{Text: fields[0]}, nil); err != nil {
			return cmd.Reply(ctx, bot, err.Error())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeout):
		}
		var ok bool
		ok, err = bot.DeleteMessage(ctx, m.Chat.ID, m.ID)
		log.Printf("Message deleted: %v", ok)
	case "/say":
		if cmd.Payload == "" {
			err = cmd.Reply(ctx, bot, "Please specify a message")
			return
		}
		_, err = bot.Send(ctx, cmd.Chat.ID,
			telegram.Text{Text: "Here you go."},
			&telegram.SendOptions{
				ReplyMarkup: telegram.InlineKeyboard(
					[][3]string{
						{"Say " + cmd.Payload, "say", cmd.Payload},
						{"Another button", "", ""}}),
			},
		)
	case "say":
		return cmd.Reply(ctx, bot, cmd.Payload)
	case "/question":
		reply, err := bot.Ask(ctx, cmd.Chat.ID,
			telegram.Text{Text: "Your question is, " + cmd.Payload},
			&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID})
		if err != nil {
			return cmd.Reply(ctx, bot, err.Error())
		}
		_, err = bot.Send(ctx, reply.Chat.ID,
			telegram.Text{Text: "Your answer is, " + reply.Text},
			&telegram.SendOptions{ReplyToMessageID: reply.ID})
		if err != nil {
			return cmd.Reply(ctx, bot, err.Error())
		}
	}
	return
}

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
	defer log.Printf("main exit")
	go func() { log.Println(http.ListenAndServe("localhost:6060", nil)) }()

	// Create a bot instance.
	defer telegram.NewBot(fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), os.Args[1], 3).
		CommandListenerFunc(
			&telegram.GetUpdatesOptions{TimeoutSecs: 60},
			flu.ConcurrencyRateLimiter(1),
			CommandListenerFunc).
		Close()
	flu.AwaitSignal(syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
}
