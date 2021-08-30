package telegram

import (
	"context"
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"

	"github.com/sirupsen/logrus"

	fluhttp "github.com/jfk9w-go/flu/http"
)

// GetUpdatesOptions is /getUpdates request options.
// See https://core.telegram.org/bots/api#getupdates
type GetUpdatesOptions struct {
	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers of previously received updates.
	// By default, updates starting with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called with an offset
	// higher than its update_id. The negative offset can be specified to retrieve updates
	// starting from -offset update from the end of the updates queue.
	// All previous updates will be forgotten.
	Offset ID `json:"offset,omitempty"`
	// Limits the number of updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit int `json:"limit,omitempty"`
	// Timeout for long polling.
	TimeoutSecs int `json:"timeout,omitempty"`
	// List the types of updates you want your bot to receive.
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

type Bot struct {
	BaseClient
	*FloodControlAware
	*ConversationAware
	ctx    context.Context
	cancel func()
	me     *User
	work   flu.WaitGroup
	once   sync.Once
}

func NewBot(ctx context.Context, client *fluhttp.Client, token string) *Bot {
	return NewBotWithEndpoint(ctx, client, token, nil)
}

func NewBotWithEndpoint(ctx context.Context, client *fluhttp.Client, token string, endpoint EndpointFunc) *Bot {
	base := NewBaseClientWithEndpoint(client, token, endpoint)
	floodControl := FloodControl(base)
	conversations := Conversations(floodControl)
	ctx, cancel := context.WithCancel(ctx)
	return &Bot{
		BaseClient:        base,
		FloodControlAware: floodControl,
		ConversationAware: conversations,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (bot *Bot) Labels() metrics.Labels {
	return metrics.Labels{}.Add("bot", bot.Username())
}

func (bot *Bot) log() *logrus.Entry {
	return logrus.WithFields(bot.Labels().Map())
}

func (bot *Bot) Listen(options GetUpdatesOptions) <-chan Update {
	log := bot.log()
	channel := make(chan Update)
	bot.work.Go(bot.ctx, func(ctx context.Context) {
		defer close(channel)
		for {
			updates, err := bot.GetUpdates(ctx, options)
			switch {
			case flu.IsContextRelated(err):
				return

			case err != nil:
				log.Warnf("poll error: %s", err)
				if err := flu.Sleep(ctx, time.Duration(options.TimeoutSecs)*time.Second); err != nil {
					return
				}

			default:
				for _, update := range updates {
					if update.Message != nil && bot.Answer(update.Message) {
						continue
					}

					select {
					case <-ctx.Done():
						return
					case channel <- update:
						options.Offset = update.ID.Increment()
					}
				}
			}
		}
	})

	return channel
}

func (bot *Bot) Username() string {
	bot.once.Do(func() {
		ctx, cancel := context.WithTimeout(bot.ctx, time.Minute)
		defer cancel()
		if me, err := bot.GetMe(ctx); err != nil {
			panic(err)
		} else {
			bot.me = me
		}
	})

	return bot.me.Username.String()
}

var DefaultCommandsOptions = &GetUpdatesOptions{
	TimeoutSecs:    60,
	AllowedUpdates: []string{"message", "edited_message", "callback_query"},
}

func (bot *Bot) Commands() <-chan Command {
	channel := make(chan Command)
	bot.work.Add(1)
	go bot.pipeCommands(bot.Listen(*DefaultCommandsOptions), channel)
	return channel
}

func (bot *Bot) pipeCommands(updates <-chan Update, commands chan<- Command) {
	defer func() {
		close(commands)
		bot.work.Done()
	}()

	for update := range updates {
		if cmd, ok := bot.extractCommand(update); ok {
			commands <- cmd
		}
	}
}

type CommandListener interface {
	OnCommand(ctx context.Context, client Client, cmd *Command) error
}

func (bot *Bot) CommandListener(value interface{}) *Bot {
	var listener CommandListener
	switch value := value.(type) {
	case CommandListener:
		listener = value
	default:
		listener = CommandRegistryFrom(value)
	}

	commands := bot.Commands()
	bot.work.Go(bot.ctx, func(ctx context.Context) {
		for cmd := range commands {
			if err := bot.HandleCommand(ctx, listener, &cmd); err != nil {
				if flu.IsContextRelated(err) {
					return
				}

				cmd.Log(bot).Warnf("handle command: %s", err)
			} else {
				cmd.Log(bot).Debugf("handle command: ok")
			}
		}
	})

	return bot
}

func (bot *Bot) HandleCommand(ctx context.Context, listener CommandListener, cmd *Command) (err error) {
	err = listener.OnCommand(ctx, bot, cmd)
	switch {
	case flu.IsContextRelated(err):
		return
	case err != nil:
		if err := cmd.Reply(ctx, bot, err.Error()); err != nil {
			return errors.Wrap(err, "reply")
		}
	}

	return
}

type CommandListenerFunc func(context.Context, Client, *Command) error

func (fun CommandListenerFunc) OnCommand(ctx context.Context, client Client, cmd *Command) error {
	return fun(ctx, client, cmd)
}

func (bot *Bot) CommandListenerFunc(fun CommandListenerFunc) *Bot {
	return bot.CommandListener(fun)
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
	elemType := valueType
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	log := logrus.WithField("service", fmt.Sprintf("%T", value))
	registry := make(CommandRegistry)
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

	return registry
}

func (bot *Bot) extractCommand(update Update) (cmd Command, ok bool) {
	switch {
	case update.Message != nil:
		cmd, ok = bot.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		cmd, ok = bot.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		cmd, ok = bot.extractCommandCallbackQuery(update.CallbackQuery)
	}

	cmd.Payload = strings.Trim(cmd.Payload, " ")
	if ok && cmd.Payload != "" {
		reader := csv.NewReader(strings.NewReader(cmd.Payload))
		reader.Comma = ' '
		reader.TrimLeadingSpace = true
		args, err := reader.Read()
		if err != nil {
			cmd.Log(bot).Debugf("failed to parse args: %s", err)
		} else {
			cmd.Args = args
		}
	}

	return
}

func (bot *Bot) extractCommandMessage(message *Message) (cmd Command, ok bool) {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			key := message.Text[entity.Offset : entity.Offset+entity.Length]
			at := strings.Index(key, "@")
			if at > 0 && len(key) > at && bot.Username() == key[at+1:] {
				key = key[:at]
			}
			cmd.User = &message.From
			cmd.Chat = &message.Chat
			cmd.Message = message
			cmd.Key = key
			cmd.Payload = message.Text[entity.Offset+entity.Length:]
			ok = true
			return
		}
	}
	return
}

func (bot *Bot) extractCommandCallbackQuery(query *CallbackQuery) (cmd Command, ok bool) {
	if query.Data == nil {
		return
	}
	for i, c := range *query.Data {
		if c == ' ' && len(*query.Data) > i+1 {
			cmd.Chat = &query.Message.Chat
			cmd.User = &query.From
			cmd.Message = query.Message
			cmd.Key = (*query.Data)[:i]
			cmd.Payload = (*query.Data)[i+1:]
			cmd.CallbackQueryID = query.ID
			ok = true
			return
		}
	}
	return
}

func (bot *Bot) Close() error {
	bot.cancel()
	bot.work.Wait()
	return nil
}

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

func (cmd *Command) Reply(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID != "" {
		_, err := client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, AnswerCallbackQueryOptions{Text: text})
		return err
	} else {
		_, err := client.Send(ctx, cmd.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: cmd.Message.ID})
		return err
	}
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

type Button [3]string

func (cmd *Command) Button(text string) Button {
	b := new(strings.Builder)
	writer := csv.NewWriter(b)
	writer.Comma = ' '
	if err := writer.Write(cmd.Args); err != nil {
		return Button{}
	}

	writer.Flush()
	return Button{text, cmd.Key, strings.Trim(b.String(), " \n")}
}

func (cmd *Command) String() string {
	str := fmt.Sprintf("[cmd > %s+%s] %s", cmd.User.ID, cmd.Chat.ID, cmd.Key)
	if cmd.Payload != "" {
		str += " " + cmd.Payload
	}
	return str
}
