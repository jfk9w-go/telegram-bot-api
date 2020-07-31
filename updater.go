package telegram

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

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
}

func NewBot(client *fluhttp.Client, token string, sendRetries int) *Bot {
	base := NewBaseClient(client, token)
	floodControl := FloodControl(base, sendRetries)
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

func (bot *Bot) Listen(options *GetUpdatesOptions) <-chan Update {
	me, err := bot.GetMe(bot.ctx)
	if err != nil {
		panic(errors.Wrap(err, "getMe"))
	}

	bot.me = me
	channel := make(chan Update)
	bot.work.Add(1)
	go func(bot *Bot) {
		defer func() {
			close(channel)
			bot.work.Done()
		}()

		for {
			updates, err := bot.GetUpdates(bot.ctx, options)
			if bot.ctx.Err() != nil {
				return
			} else if err != nil {
				log.Printf("%s poll error: %s", bot.Username(), err)
				select {
				case <-bot.ctx.Done():
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
					case <-bot.ctx.Done():
						return
					case channel <- update:
					}
				}

				options.Offset = update.ID.Increment()
			}
		}
	}(bot)

	return channel
}

func (bot *Bot) Username() string {
	return bot.me.Username.String()
}

func (bot *Bot) Commands(options *GetUpdatesOptions) <-chan Command {
	options.AllowedUpdates = []string{"message", "edited_message", "callback_query"}
	channel := make(chan Command)
	bot.work.Add(1)
	go func(bot *Bot, updates <-chan Update) {
		defer func() {
			close(channel)
			bot.work.Done()
		}()
		for update := range updates {
			if cmd, ok := bot.extractCommand(update); ok {
				channel <- cmd
			}
		}
	}(bot, bot.Listen(options))
	return channel
}

type CommandListener interface {
	OnCommand(ctx context.Context, client Client, cmd Command) error
}

func (bot *Bot) CommandListener(options *GetUpdatesOptions, rateLimiter flu.RateLimiter, listener CommandListener) *Bot {
	if rateLimiter == nil {
		rateLimiter = flu.RateUnlimiter
	}

	commands := bot.Commands(options)
	log.Printf("%s is running", bot.Username())
	bot.work.Add(1)
	go func(bot *Bot) {
		defer bot.work.Done()
		for cmd := range commands {
			if err := rateLimiter.Start(bot.ctx); err != nil {
				return
			}

			ctx, cancel := context.WithCancel(bot.ctx)
			bot.work.Add(1)
			go func(ctx context.Context, cancel func(), bot *Bot, cmd Command) {
				defer func() {
					cancel()
					rateLimiter.Complete()
					bot.work.Done()
				}()

				err := listener.OnCommand(ctx, bot, cmd)
				if ctx.Err() != nil {
					return
				}

				if err != nil {
					log.Printf("%s processed %s from %d with error %s", bot.Username(), cmd.Key, cmd.User.ID, err)
					sendErr := cmd.Reply(ctx, bot, err.Error())
					if sendErr != nil {
						log.Printf(`%s unable to send error reply "%s" to %s: %s`,
							bot.Username(), err.Error(), cmd.Chat.ID, sendErr.Error())
					}
				} else {
					log.Printf("%s processed %s from %d ok", bot.Username(), cmd.Key, cmd.User.ID)
				}
			}(ctx, cancel, bot, cmd)
		}
	}(bot)
	return bot
}

type CommandListenerFunc func(context.Context, Client, Command) error

func (fun CommandListenerFunc) OnCommand(ctx context.Context, client Client, cmd Command) error {
	return fun(ctx, client, cmd)
}

func (bot *Bot) CommandListenerFunc(options *GetUpdatesOptions, rateLimiter flu.RateLimiter, fun CommandListenerFunc) *Bot {
	return bot.CommandListener(options, rateLimiter, fun)
}

func (bot *Bot) extractCommand(update Update) (Command, bool) {
	switch {
	case update.Message != nil:
		return bot.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return bot.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return bot.extractCommandCallbackQuery(update.CallbackQuery)
	}
	return Command{}, false
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
			cmd.Payload = strings.Trim(message.Text[entity.Offset+entity.Length:], " ")
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
		if c == ':' && len(*query.Data) > i+1 {
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
	CallbackQueryID string
}

func (cmd Command) Reply(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID != "" {
		_, err := client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &AnswerCallbackQueryOptions{Text: text})
		return err
	} else {
		_, err := client.Send(ctx, cmd.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: cmd.Message.ID})
		return err
	}
}
