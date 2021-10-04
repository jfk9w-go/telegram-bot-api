package telegram

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"

	"github.com/sirupsen/logrus"

	fluhttp "github.com/jfk9w-go/flu/http"
)

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
					if update.Message != nil && update.Message.ReplyToMessage != nil {
						if err := bot.Answer(ctx, update.Message); err != nil {
							if flu.IsContextRelated(err) {
								return
							}

							bot.log().Warnf("answer %d: %s", update.Message.ID, err)
						}
					} else {
						select {
						case <-ctx.Done():
							return
						case channel <- update:
							break
						}
					}

					options.Offset = update.ID.Increment()
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

func (bot *Bot) Commands() <-chan *Command {
	channel := make(chan *Command)
	bot.work.Add(1)
	go bot.pipeCommands(bot.Listen(*DefaultCommandsOptions), channel)
	return channel
}

func (bot *Bot) pipeCommands(updates <-chan Update, commands chan<- *Command) {
	defer func() {
		close(commands)
		bot.work.Done()
	}()

	for update := range updates {
		if cmd := bot.extractCommand(update); cmd != nil {
			commands <- cmd
		}
	}
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
			if cmd.Key == "/start" && cmd.Payload != "" {
				if data, err := base64.URLEncoding.DecodeString(cmd.Payload); err != nil {
					logrus.WithFields(cmd.Labels().Map()).
						Debugf("parse base64 start data: %s", err)
				} else {
					cmd.init(bot.Username(), string(data))
				}
			}

			if err := bot.HandleCommand(ctx, listener, cmd); err != nil {
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

func (bot *Bot) CommandListenerFunc(fun CommandListenerFunc) *Bot {
	return bot.CommandListener(fun)
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

func (bot *Bot) extractCommand(update Update) *Command {
	switch {
	case update.Message != nil:
		return bot.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return bot.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return bot.extractCommandCallbackQuery(update.CallbackQuery)
	default:
		return nil
	}
}

func (bot *Bot) extractCommandMessage(message *Message) *Command {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			cmd := &Command{
				User:    &message.From,
				Chat:    &message.Chat,
				Message: message,
			}

			cmd.init(bot.Username(), message.Text[entity.Offset:])
			return cmd
		}
	}

	return nil
}

func (bot *Bot) extractCommandCallbackQuery(query *CallbackQuery) *Command {
	if query.Data == nil {
		return nil
	}

	cmd := &Command{
		Chat:            &query.Message.Chat,
		User:            &query.From,
		Message:         query.Message,
		CallbackQueryID: query.ID,
	}

	cmd.init(bot.Username(), *query.Data)
	return cmd
}

func trim(value string) string {
	return strings.Trim(value, " \n\t\v")
}

func (bot *Bot) Close() error {
	bot.cancel()
	bot.work.Wait()
	return nil
}
