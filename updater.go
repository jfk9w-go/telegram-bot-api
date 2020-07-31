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

type UpdateAware struct {
	BaseClient
	*FloodControlAware
	*ConversationAware
	ctx    context.Context
	cancel func()
	me     *User
	work   sync.WaitGroup
}

func NewBot(client *fluhttp.Client, token string, sendRetries int) Bot {
	base := NewBaseClient(client, token)
	floodControl := FloodControl(base, sendRetries)
	conversations := Conversations(floodControl)
	ctx, cancel := context.WithCancel(context.Background())
	return &UpdateAware{
		BaseClient:        base,
		FloodControlAware: floodControl,
		ConversationAware: conversations,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (u *UpdateAware) Listen(options *GetUpdatesOptions) <-chan Update {
	me, err := u.GetMe(u.ctx)
	if err != nil {
		panic(errors.Wrap(err, "getMe"))
	}

	u.me = me
	channel := make(chan Update)
	u.work.Add(1)
	go func() {
		defer func() {
			close(channel)
			u.work.Done()
		}()

		for {
			updates, err := u.GetUpdates(u.ctx, options)
			if u.ctx.Err() != nil {
				return
			} else if err != nil {
				log.Printf("%s poll error: %s", u.Username(), err)
				select {
				case <-u.ctx.Done():
					return
				case <-time.After(time.Duration(options.TimeoutSecs) * time.Second):
					continue
				}
			}

			for _, update := range updates {
				if update.Message != nil && u.Answer(update.Message) {
					// already answered
				} else {
					select {
					case <-u.ctx.Done():
						return
					case channel <- update:
					}
				}

				options.Offset = update.ID.Increment()
			}
		}
	}()

	return channel
}

func (u *UpdateAware) Username() string {
	return u.me.Username.String()
}

func (u *UpdateAware) Commands(options *GetUpdatesOptions) <-chan Command {
	options.AllowedUpdates = []string{"message", "edited_message", "callback_query"}
	channel := make(chan Command)
	go func(updates <-chan Update) {
		defer close(channel)
		for update := range updates {
			if cmd, ok := u.extractCommand(update); ok {
				channel <- cmd
			}
		}
	}(u.Listen(options))

	return channel
}

func (u *UpdateAware) extractCommand(update Update) (Command, bool) {
	switch {
	case update.Message != nil:
		return u.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return u.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return u.extractCommandCallbackQuery(update.CallbackQuery)
	}
	return Command{}, false
}

func (u *UpdateAware) extractCommandMessage(message *Message) (cmd Command, ok bool) {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			key := message.Text[entity.Offset : entity.Offset+entity.Length]
			at := strings.Index(key, "@")
			if at > 0 && len(key) > at && u.Username() == key[at+1:] {
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

func (u *UpdateAware) extractCommandCallbackQuery(query *CallbackQuery) (cmd Command, ok bool) {
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

func (u *UpdateAware) Close() {
	u.cancel()
	u.work.Wait()
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
