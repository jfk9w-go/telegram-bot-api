package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
)

const LoremIpsum = `
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Cras semper auctor neque vitae tempus quam pellentesque nec nam. Id diam vel quam elementum pulvinar etiam. Pellentesque eu tincidunt tortor aliquam nulla facilisi cras fermentum. Sagittis id consectetur purus ut faucibus pulvinar elementum integer. Rutrum tellus pellentesque eu tincidunt tortor aliquam. Varius morbi enim nunc faucibus a pellentesque. Pellentesque pulvinar pellentesque habitant morbi. In vitae turpis massa sed elementum tempus. Integer eget aliquet nibh praesent tristique. Pharetra sit amet aliquam id diam maecenas ultricies. Bibendum neque egestas congue quisque egestas diam in arcu cursus. Sed augue lacus viverra vitae congue eu consequat ac. Tempor nec feugiat nisl pretium fusce id velit ut. Sit amet massa vitae tortor condimentum. Lorem ipsum dolor sit amet consectetur. Proin libero nunc consequat interdum varius sit. Dui faucibus in ornare quam viverra orci. Non pulvinar neque laoreet suspendisse interdum. Maecenas pharetra convallis posuere morbi leo.

Mollis aliquam ut porttitor leo a diam sollicitudin. Donec massa sapien faucibus et molestie ac. Dolor sed viverra ipsum nunc aliquet bibendum enim facilisis. Ut pharetra sit amet aliquam id. Arcu risus quis varius quam quisque id. Et ultrices neque ornare aenean euismod elementum nisi. Velit dignissim sodales ut eu sem integer vitae justo. Venenatis tellus in metus vulputate eu scelerisque felis imperdiet proin. Scelerisque purus semper eget duis at tellus at urna condimentum. Mauris in aliquam sem fringilla ut morbi tincidunt augue. Sit amet consectetur adipiscing elit duis tristique sollicitudin nibh sit. Et malesuada fames ac turpis egestas. Ac orci phasellus egestas tellus rutrum tellus pellentesque eu tincidunt. Cras semper auctor neque vitae tempus quam pellentesque. Odio aenean sed adipiscing diam donec adipiscing tristique risus nec. Sem nulla pharetra diam sit amet. In fermentum posuere urna nec tincidunt.

Fusce id velit ut tortor pretium viverra suspendisse potenti nullam. Non odio euismod lacinia at quis risus. In eu mi bibendum neque egestas. Congue quisque egestas diam in arcu cursus. Donec pretium vulputate sapien nec sagittis aliquam. A iaculis at erat pellentesque adipiscing commodo elit. Metus aliquam eleifend mi in nulla posuere sollicitudin aliquam. Hendrerit gravida rutrum quisque non tellus. Sit amet justo donec enim. Egestas congue quisque egestas diam in arcu cursus euismod quis. Lectus vestibulum mattis ullamcorper velit sed ullamcorper morbi tincidunt ornare. Turpis in eu mi bibendum neque. Ac orci phasellus egestas tellus rutrum tellus pellentesque eu. Rutrum quisque non tellus orci ac. Sociis natoque penatibus et magnis dis parturient montes. Laoreet sit amet cursus sit. Sit amet aliquam id diam maecenas ultricies mi.

Quis vel eros donec ac odio tempor orci dapibus. Senectus et netus et malesuada fames ac turpis egestas integer. Ultricies integer quis auctor elit. Molestie at elementum eu facilisis sed odio morbi. Viverra adipiscing at in tellus integer feugiat scelerisque varius. Orci sagittis eu volutpat odio facilisis mauris sit amet massa. Mi proin sed libero enim sed faucibus. Sed viverra ipsum nunc aliquet bibendum enim facilisis gravida neque. Turpis tincidunt id aliquet risus feugiat in ante. Nec ultrices dui sapien eget mi proin sed. Ac tincidunt vitae semper quis lectus nulla at. Eget nunc lobortis mattis aliquam. Molestie at elementum eu facilisis sed odio morbi quis. Nibh cras pulvinar mattis nunc sed blandit libero. Habitant morbi tristique senectus et. Leo a diam sollicitudin tempor id eu nisl. Adipiscing commodo elit at imperdiet dui accumsan. Amet nisl suscipit adipiscing bibendum est ultricies integer quis.

Tellus mauris a diam maecenas. Non pulvinar neque laoreet suspendisse interdum consectetur libero id faucibus. In nibh mauris cursus mattis molestie a iaculis. Massa ultricies mi quis hendrerit. Etiam sit amet nisl purus in mollis. Quam pellentesque nec nam aliquam sem et. Interdum velit laoreet id donec ultrices tincidunt. Et malesuada fames ac turpis egestas integer. Viverra vitae congue eu consequat ac. Tincidunt augue interdum velit euismod in pellentesque. Consectetur lorem donec massa sapien faucibus. Sit amet mauris commodo quis imperdiet massa tincidunt nunc pulvinar. Lorem ipsum dolor sit amet consectetur.

Sagittis aliquam malesuada bibendum arcu vitae elementum curabitur. Vitae auctor eu augue ut lectus. Diam volutpat commodo sed egestas egestas fringilla phasellus faucibus scelerisque. Dictum at tempor commodo ullamcorper a lacus vestibulum. Porttitor rhoncus dolor purus non enim. Scelerisque eleifend donec pretium vulputate sapien nec sagittis aliquam malesuada. Quam lacus suspendisse faucibus interdum posuere lorem ipsum dolor sit. Ultrices sagittis orci a scelerisque purus semper eget. Sit amet consectetur adipiscing elit ut aliquam purus sit. Ornare arcu dui vivamus arcu felis bibendum. Mus mauris vitae ultricies leo integer malesuada nunc vel. Enim eu turpis egestas pretium aenean. Est pellentesque elit ullamcorper dignissim. Orci ac auctor augue mauris augue neque gravida in.`

type CommandListener struct {
	flu.RateLimiter
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
	url := "https://thumbs.dreamstime.com/z/black-check-mark-icon-tick-symbol-tick-icon-vector-illustration-flat-ok-sticker-icon-isolated-white-accept-black-check-mark-137505360.jpg"

	mvar := media.NewVar()
	mvar.Set(&media.Value{
		MIMEType: "image/jpeg",
		Input:    flu.URL(url),
	}, nil)

	html := &html.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    client,
				ID:        cmd.Chat.ID,
				ParseMode: telegram.HTML,
			},
		},
	}

	return html.
		Text("Here's a ").
		Bold("tick").
		Italic(" for ya!").
		Media(url, mvar, true, true).
		Flush()
}

func (l CommandListener) Lorem(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	url := "https://thumbs.dreamstime.com/z/black-check-mark-icon-tick-symbol-tick-icon-vector-illustration-flat-ok-sticker-icon-isolated-white-accept-black-check-mark-137505360.jpg"

	mvar := media.NewVar()
	mvar.Set(&media.Value{
		MIMEType: "image/jpeg",
		Input:    flu.File("tick.jpg"),
	}, nil)

	html := &html.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    client,
				ID:        cmd.Chat.ID,
				ParseMode: telegram.HTML,
			},
			PageSize: telegram.MaxMessageSize * 9 / 10,
		},
	}

	return html.
		Text(LoremIpsum).
		Media(url, mvar, false, true).
		Media(url, mvar, false, true).
		Media(url, mvar, false, true).
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

func (l CommandListener) CommandRegistry() map[string]telegram.CommandListener {
	return telegram.CommandRegistryFrom(l)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetLevel(logrus.DebugLevel)

	bot := telegram.NewBot(ctx, fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), os.Args[1])

	defer flu.CloseQuietly(
		bot.CommandListener(CommandListener{
			RateLimiter: flu.ConcurrencyRateLimiter(2),
		}),
	)

	flu.AwaitSignal(syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
}
