package main

import (
	"context"
	_ "embed"
	"fmt"
	_ "net/http/pprof"
	"os"
	"strconv"
	"syscall"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
)

var (
	//go:embed lorem_ipsum.txt
	LoremIpsum string
	Checkmark  = flu.URL("https://thumbs.dreamstime.com/z/black-check-mark-icon-tick-symbol-tick-icon-vector-illustration-flat-ok-sticker-icon-isolated-white-accept-black-check-mark-137505360.jpg")
)

type CommandListener struct {
	syncf.Locker
}

func (l CommandListener) Greet(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	_, err := client.Send(ctx, cmd.Chat.ID,
		telegram.Text{
			ParseMode: telegram.HTML,
			Text:      fmt.Sprintf(`Hello, <i><pre><b><a href="%s"><i>Google</i></a></b></pre></i>`, "https://www.google.com")},
		&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID})
	return err
}

func (l CommandListener) Tick(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	html := (&html.Writer{
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    client,
				ID:        cmd.Chat.ID,
				ParseMode: telegram.HTML,
			},
		},
	}).WithContext(ctx)

	media := syncf.Value[*receiver.Media]{
		Val: &receiver.Media{
			MIMEType: "image/jpeg",
			Input:    Checkmark,
		},
	}

	return html.
		Text("Here's a ").
		Bold("tick").
		Italic(" for ya!").
		Media(Checkmark.String(), media, true, true).
		Flush()
}

func (l CommandListener) Lorem(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	html := (&html.Writer{
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    client,
				ID:        cmd.Chat.ID,
				ParseMode: telegram.HTML,
			},
		},
	}).WithContext(output.With(ctx, telegram.MaxMessageSize*9/10, 0))

	media := syncf.Value[*receiver.Media]{
		Val: &receiver.Media{
			MIMEType: "image/jpeg",
			Input:    flu.File("tick.jpg"),
		},
	}

	return html.
		Text(LoremIpsum).
		Media(Checkmark.String(), media, false, true).
		Media(Checkmark.String(), media, false, true).
		Media(Checkmark.String(), media, false, true).
		Flush()
}

func (l CommandListener) Gif(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	_, err := client.Send(ctx, cmd.Chat.ID,
		telegram.Media{
			Type:    telegram.MediaTypeByMIMEType("image/gif"),
			Input:   flu.File("gif.gif"),
			Caption: "GIF"},
		&telegram.SendOptions{DisableNotification: true})
	return err
}

func (l CommandListener) Webp(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	_, err := client.Send(ctx, cmd.Chat.ID,
		telegram.Media{
			Type:  telegram.MediaTypeByMIMEType("image/webp"),
			Input: flu.File("webp.webp")},
		&telegram.SendOptions{DisableNotification: true})
	return err
}

func (l CommandListener) Count(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if limit, err := strconv.Atoi(cmd.Payload); err != nil || limit <= 0 {
		return errors.New("limit must be a positive integer")
	} else {
		for i := 1; i <= limit; i++ {
			_, err = client.Send(ctx, cmd.Chat.ID, telegram.Text{Text: fmt.Sprintf("%d", i)}, nil)
			if err != nil {
				return errors.Wrapf(err, "send %d", i)
			}
		}
	}

	return nil
}

func (l CommandListener) Say(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if cmd.Payload == "" {
		return errors.New("specify a phrase")
	} else if _, err := client.Send(ctx, cmd.Chat.ID,
		telegram.Text{Text: "Here you go."},
		&telegram.SendOptions{
			ReplyMarkup: telegram.InlineKeyboard([]telegram.Button{
				{"Say " + cmd.Payload, "say", cmd.Payload},
				{"Another button", "", ""}})}); err != nil {
		return errors.Wrap(err, "send")
	}

	return nil
}

func (l CommandListener) SayCallback(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	return cmd.Reply(ctx, client, cmd.Payload)
}

func (l CommandListener) Question(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if reply, err := client.Ask(ctx, cmd.Chat.ID,
		telegram.Text{Text: "Your question is, " + cmd.Payload},
		&telegram.SendOptions{ReplyToMessageID: cmd.Message.ID}); err != nil {
		return errors.Wrap(err, "ask")
	} else if _, err := client.Send(ctx, reply.Chat.ID,
		telegram.Text{Text: "Your answer is, " + reply.Text},
		&telegram.SendOptions{ReplyToMessageID: reply.ID}); err != nil {
		return errors.Wrap(err, "answer")
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
// where <token> is your Telegram bot API token.
func main() {
	logf.ResetLevel(logf.Trace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clock := syncf.DefaultClock
	bot := telegram.NewBot(clock, nil, os.Args[1])

	defer flu.CloseQuietly(
		bot.CommandListener(CommandListener{
			Locker: syncf.Semaphore(clock, 2, 0),
		}),
	)

	syncf.AwaitSignal(ctx, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
}
