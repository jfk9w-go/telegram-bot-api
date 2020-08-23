package telegram

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
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

func (o *GetUpdatesOptions) body() flu.EncoderTo {
	return flu.JSON{o}
}

type Bot struct {
	BaseClient
	*FloodControlAware
	*ConversationAware
	ctx    context.Context
	cancel func()
	me     *User
	work   sync.WaitGroup
	once   sync.Once
}

func NewBot(client *fluhttp.Client, token string) *Bot {
	return NewBotWithEndpoint(client, token, nil)
}

func NewBotWithEndpoint(client *fluhttp.Client, token string, endpoint EndpointFunc) *Bot {
	base := NewBaseClientWithEndpoint(client, token, endpoint)
	floodControl := FloodControl(base)
	conversations := Conversations(floodControl)
	ctx, cancel := context.WithCancel(context.Background())
	return &Bot{
		BaseClient:        base,
		FloodControlAware: floodControl,
		ConversationAware: conversations,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (bot *Bot) Listen(options GetUpdatesOptions) <-chan Update {
	channel := make(chan Update)
	ctx, cancel := context.WithCancel(bot.ctx)
	bot.work.Add(1)
	go bot.runUpdateListener(ctx, cancel, options, channel)
	return channel
}

func (bot *Bot) runUpdateListener(ctx context.Context, cancel func(), options GetUpdatesOptions, channel chan<- Update) {
	defer func() {
		close(channel)
		cancel()
		bot.work.Done()
	}()

	for {
		updates, err := bot.GetUpdates(ctx, options)
		if ctx.Err() != nil {
			return
		} else if err != nil {
			log.Printf("%s poll error: %s", bot.Username(), err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(options.TimeoutSecs) * time.Second):
				continue
			}
		}

		for _, update := range updates {
			if update.Message != nil && bot.Answer(update.Message) {
				// already answered
			} else {
				select {
				case <-ctx.Done():
					return
				case channel <- update:
				}
			}

			options.Offset = update.ID.Increment()
		}
	}
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
	OnCommand(ctx context.Context, client Client, cmd Command) error
}

func (bot *Bot) CommandListener(listener CommandListener) *Bot {
	commands := bot.Commands()
	log.Printf("%s is running", bot.Username())
	bot.work.Add(1)
	go bot.runCommandListener(commands, listener)
	return bot
}

func (bot *Bot) runCommandListener(commands <-chan Command, listener CommandListener) {
	defer bot.work.Done()
	for cmd := range commands {
		ctx, cancel := context.WithCancel(bot.ctx)
		bot.work.Add(1)
		go bot.handleCommand(ctx, cancel, listener, cmd)
	}
}

func (bot *Bot) handleCommand(ctx context.Context, cancel func(), listener CommandListener, cmd Command) {
	defer func() {
		cancel()
		bot.work.Done()
	}()

	err := listener.OnCommand(ctx, bot, cmd)
	if ctx.Err() != nil {
		log.Printf(`%s => %s`, cmd, ctx.Err())
		return
	} else if err != nil {
		log.Printf(`%s => %s`, cmd, err)
		if sendErr := cmd.Reply(ctx, bot, err.Error()); sendErr != nil {
			log.Printf(`%s unable to send error reply "%s" to %s: %s`,
				bot.Username(), err.Error(), cmd.Chat.ID, sendErr.Error())
		}
	} else {
		log.Printf(`%s => ok`, cmd)
	}
}

type CommandListenerFunc func(context.Context, Client, Command) error

func (fun CommandListenerFunc) OnCommand(ctx context.Context, client Client, cmd Command) error {
	return fun(ctx, client, cmd)
}

func (bot *Bot) CommandListenerFunc(fun CommandListenerFunc) *Bot {
	return bot.CommandListener(fun)
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
			log.Printf("%s => failed to parse args: %s", cmd, err)
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

func (bot *Bot) Close() {
	bot.cancel()
	bot.work.Wait()
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

func (cmd Command) Reply(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID != "" {
		_, err := client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, AnswerCallbackQueryOptions{Text: text})
		return err
	} else {
		_, err := client.Send(ctx, cmd.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: cmd.Message.ID})
		return err
	}
}

func (cmd Command) String() string {
	str := fmt.Sprintf("[cmd-%s+%s] %s", cmd.User.ID, cmd.Chat.ID, cmd.Key)
	if cmd.Payload != "" {
		str += " " + cmd.Payload
	}
	return str
}
