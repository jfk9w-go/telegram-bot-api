package telegram

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/jfk9w-go/flu/metrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Command is a text bot command.
type Command struct {
	Chat            *Chat
	User            *User
	Message         *Message
	Key             string
	Payload         string
	Args            []string
	CallbackQueryID string
}

func (cmd *Command) init(username, value string) {
	cmd.Key = value

	space := strings.Index(value, " ")
	if space > 0 && len(value) > space+1 {
		cmd.Key = value[:space]
		cmd.Payload = trim(value[space+1:])
	}

	at := strings.Index(cmd.Key, "@")
	if at > 0 && len(cmd.Key) > at+1 && username == cmd.Key[at+1:] {
		cmd.Key = cmd.Key[:at]
	}

	cmd.Key = trim(cmd.Key)
	cmd.Payload = trim(cmd.Payload)

	cmd.Args = make([]string, 0)
	if cmd.Payload == "" {
		return
	}

	reader := csv.NewReader(strings.NewReader(cmd.Payload))
	reader.Comma = ' '
	reader.TrimLeadingSpace = true
	args, err := reader.Read()
	if err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("parse args: %s", err)
		return
	}

	cmd.Args = args
}

func (cmd *Command) Arg(i int) string {
	if len(cmd.Args) > i {
		return cmd.Args[i]
	}

	return ""
}

func (cmd *Command) Reply(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID != "" {
		return client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &AnswerOptions{Text: text})
	}

	_, err := client.Send(ctx, cmd.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: cmd.Message.ID})
	return err
}

func (cmd *Command) Labels() metrics.Labels {
	return metrics.Labels{}.
		Add("chat", cmd.Chat.ID).
		Add("user", cmd.User.ID).
		Add("command", cmd.Key).
		Add("payload", cmd.Payload)
}

func (cmd *Command) Log(bot *Bot) *logrus.Entry {
	return logrus.WithFields(bot.Labels().AddAll(cmd.Labels()).Map())
}

func (cmd *Command) collectArgs() string {
	b := new(strings.Builder)
	writer := csv.NewWriter(b)
	writer.Comma = ' '
	if err := writer.Write(cmd.Args); err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("write args: %s", err)
		return ""
	}

	writer.Flush()
	return b.String()
}

func (cmd *Command) Start(ctx context.Context, client Client) error {
	if cmd.CallbackQueryID == "" {
		return errors.New("not a callback query")
	}

	data := []byte(cmd.Key + " " + cmd.collectArgs())
	url := fmt.Sprintf("https://t.me/%s?start=%s", client.Username(), base64.URLEncoding.EncodeToString(data))
	return client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &AnswerOptions{URL: url})
}

type Button [3]string

func (b Button) StartCallbackURL(username string) string {
	return fmt.Sprintf("https://t.me/%s?start=%s", username, base64.URLEncoding.EncodeToString([]byte(b[1]+" "+b[2])))
}

func (cmd *Command) Button(text string) Button {
	return Button{text, cmd.Key, cmd.collectArgs()}
}

func (cmd *Command) String() string {
	str := fmt.Sprintf("[cmd > %s+%s] %s", cmd.User.ID, cmd.Chat.ID, cmd.Key)
	if cmd.Payload != "" {
		str += " " + cmd.Payload
	}
	return str
}

type CommandListener interface {
	OnCommand(ctx context.Context, client Client, cmd *Command) error
}

type CommandListenerFunc func(context.Context, Client, *Command) error

func (fun CommandListenerFunc) OnCommand(ctx context.Context, client Client, cmd *Command) error {
	return fun(ctx, client, cmd)
}

type CommandRegistry map[string]CommandListener

func (r CommandRegistry) Add(key string, listener CommandListener) CommandRegistry {
	if _, ok := r[key]; ok {
		logrus.Fatalf("duplicate command handler: %s", key)
	}

	r[key] = listener
	return r
}

func (r CommandRegistry) AddFunc(key string, listener CommandListenerFunc) CommandRegistry {
	return r.Add(key, listener)
}

func (r CommandRegistry) OnCommand(ctx context.Context, client Client, cmd *Command) error {
	if listener, ok := r[cmd.Key]; ok {
		return listener.OnCommand(ctx, client, cmd)
	}

	return nil
}

func CommandRegistryFrom(value interface{}) CommandRegistry {
	valueType := reflect.TypeOf(value)
	registry := make(CommandRegistry)
	log := logrus.WithField("service", fmt.Sprintf("%T", value))

	elemType := valueType
	for {
		for i := 0; i < elemType.NumMethod(); i++ {
			method := elemType.Method(i)
			methodType := method.Type
			if method.IsExported() && methodType.NumIn() == 4 && methodType.NumOut() == 1 &&
				methodType.In(1).AssignableTo(reflect.TypeOf(new(context.Context)).Elem()) &&
				methodType.In(2).AssignableTo(reflect.TypeOf(new(Client)).Elem()) &&
				methodType.In(3).AssignableTo(reflect.TypeOf(new(Command))) &&
				methodType.Out(0).AssignableTo(reflect.TypeOf(new(error)).Elem()) {

				name := method.Name
				runes := []rune(name)
				runes[0] = unicode.ToLower(runes[0])
				name = string(runes)
				if strings.HasSuffix(name, "Callback") {
					name = name[:len(name)-8]
				} else {
					name = "/" + name
				}

				handle := CommandListenerFunc(func(ctx context.Context, client Client, command *Command) error {
					err := method.Func.Call([]reflect.Value{
						reflect.ValueOf(value),
						reflect.ValueOf(ctx),
						reflect.ValueOf(client),
						reflect.ValueOf(command),
					})[0].Interface()
					if err != nil {
						return err.(error)
					}

					return nil
				})

				registry[name] = handle
				log.WithField("command", name).
					WithField("handler", method.Name).
					Infof("command handler registered")
			}
		}

		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
			continue
		}

		if len(registry) == 0 {
			log.Fatal("no command listeners found")
		}

		return registry
	}
}
