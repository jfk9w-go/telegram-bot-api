package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type CommandListener struct {
	flu.RateLimiter
}

func (l CommandListener) OnCommand(ctx context.Context, bot telegram.Client, cmd telegram.Command) (err error) {
	if err := l.Start(ctx); err != nil {
		return err
	}
	defer l.Complete()

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
				Caption:   "Here's a <b>tick</b> for ya!",
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
		if limit, err := strconv.Atoi(cmd.Payload); err != nil || limit <= 0 {
			return errors.New("limit must be a positive integer")
		} else {
			for i := 1; i <= limit; i++ {
				_, err = bot.Send(ctx, cmd.Chat.ID, telegram.Text{Text: fmt.Sprintf("%d", i)}, nil)
				if err != nil {
					return errors.Wrapf(err, "send %d", i)
				}
			}
		}
	case "/secret":
		if len(cmd.Args) != 2 {
			return errors.New("usage: /secret Hi 5")
		} else if secs, err := strconv.Atoi(cmd.Args[1]); err != nil || secs <= 0 {
			return errors.New("secs must be a positive integer")
		} else if m, err := bot.Send(ctx, cmd.Chat.ID, telegram.Text{Text: cmd.Args[0]}, nil); err != nil {
			return errors.Wrap(err, "send")
		} else {
			timer := time.NewTimer(time.Duration(secs) * time.Second)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
				timer.Stop()
				if ok, err := bot.DeleteMessage(ctx, m.Chat.ID, m.ID); err != nil {
					return errors.Wrap(err, "delete")
				} else if ok {
					return nil
				}
			}
		}
	case "/say":
		if cmd.Payload == "" {
			return errors.New("specify a phrase")
		} else if _, err := bot.Send(ctx, cmd.Chat.ID,
			telegram.Text{Text: "Here you go."},
			&telegram.SendOptions{
				ReplyMarkup: telegram.InlineKeyboard([][3]string{
					{"Say " + cmd.Payload, "say", cmd.Payload},
					{"Another button", "", ""}})}); err != nil {
			return errors.Wrap(err, "send")
		}
	case "say":
		if err := cmd.Reply(ctx, bot, cmd.Payload); err != nil {
			return errors.Wrap(err, "on reply")
		}
	case "/question":
		if reply, err := bot.Ask(ctx, cmd.Chat.ID,
			telegram.Text{Text: "Your question is, " + cmd.Payload},
			&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID}); err != nil {
			return errors.Wrap(err, "ask")
		} else if _, err := bot.Send(ctx, reply.Chat.ID,
			telegram.Text{Text: "Your answer is, " + reply.Text},
			&telegram.SendOptions{ReplyToMessageID: reply.ID}); err != nil {
			return errors.Wrap(err, "answer")
		}
	default:
		return errors.New("invalid command")
	}
	return nil
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
		NewClient(), os.Args[1]).
		CommandListener(CommandListener{RateLimiter: flu.ConcurrencyRateLimiter(5)}).
		Close()
	flu.AwaitSignal(syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
}
