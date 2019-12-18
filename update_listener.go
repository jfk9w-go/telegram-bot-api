package telegram

import (
	"strings"

	"github.com/pkg/errors"
)

type Client = *floodControlAwareClient

// UpdateListener is a handler for incoming Updates.
type UpdateListener interface {
	// ReceiveUpdate is called on every received Update.
	ReceiveUpdate(Client, Update) error
	// AllowedUpdates_ is the allowed_updates parameter passed
	// in API calls to /getUpdates or /setWebhook.
	AllowedUpdates() []string
}

// CommandListener is a UpdateListener handling incoming bot commands
// with message and edited_message allowed updates.
type CommandListener struct {
	handlers map[string]CommandHandler
}

// NewCommandListener creates a new instance of CommandListener.
func NewCommandListener() *CommandListener {
	return &CommandListener{make(map[string]CommandHandler)}
}

// Handle binds a CommandHandler to a command.
// Panics if the binding already exists.
func (l *CommandListener) Handle(key string, handler CommandHandler) *CommandListener {
	if _, ok := l.handlers[key]; ok {
		panic("command handler for " + key + " already registered")
	}
	l.handlers[key] = handler
	return l
}

// HandleFunc is a shortcut for Handle(key, CommandListerFunc(func (*floodControlAwareClient, *Command) {...}))
func (l *CommandListener) HandleFunc(key string, handler CommandHandlerFunc) *CommandListener {
	return l.Handle(key, handler)
}

func (l *CommandListener) ReceiveUpdate(c Client, update Update) error {
	cmd := extractCommand(update)
	if cmd == nil {
		return nil
	}
	if listener, ok := l.handlers[cmd.Key]; ok {
		err := listener.HandleCommand(c, cmd)
		if err != nil {
			return errors.Wrapf(err, "while handling %v", update)
		}
	}
	return nil
}

func (l *CommandListener) AllowedUpdates() []string {
	return []string{"message", "edited_message", "callback_query"}
}

func extractCommand(update Update) *Command {
	switch {
	case update.Message != nil:
		return extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return extractCommandCallbackQuery(update.CallbackQuery)
	}
	return nil
}

func extractCommandMessage(message *Message) *Command {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			return &Command{
				User:      &message.From,
				Chat:      &message.Chat,
				MessageID: message.ID,
				Key:       message.Text[entity.Offset : entity.Offset+entity.Length],
				Payload:   strings.Trim(message.Text[entity.Offset+entity.Length:], " "),
			}
		}
	}
	return nil
}

func extractCommandCallbackQuery(query *CallbackQuery) *Command {
	if query.Data == nil {
		return nil
	}
	for i, c := range *query.Data {
		if c == ':' && len(*query.Data) > i+1 {
			return &Command{
				Chat:            &query.Message.Chat,
				User:            &query.From,
				MessageID:       query.Message.ID,
				Key:             (*query.Data)[:i],
				Payload:         (*query.Data)[i+1:],
				CallbackQueryID: query.ID,
			}
		}
	}
	return nil
}

func CommandButton(text, key, data string) ReplyMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         text,
					CallbackData: key + ":" + data,
				},
			},
		},
	}
}

// Command is a text bot command.
type Command struct {
	Chat            *Chat
	User            *User
	MessageID       ID
	Key             string
	Payload         string
	CallbackQueryID string
}

// CommandHandler describes a bot command handler.
type CommandHandler interface {
	HandleCommand(Client, *Command) error
}

// CommandHandlerFunc implements CommandHandler interface for lambdas.
type CommandHandlerFunc func(tg Client, cmd *Command) error

func (f CommandHandlerFunc) HandleCommand(c Client, cmd *Command) error {
	return f(c, cmd)
}
