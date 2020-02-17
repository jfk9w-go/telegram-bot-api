package telegram

import (
	"strings"

	"github.com/pkg/errors"
)

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
	username string
}

// NewCommandListener creates a new instance of CommandListener.
func NewCommandListener(username string) *CommandListener {
	return &CommandListener{make(map[string]CommandHandler), username}
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

func (l *CommandListener) ReceiveUpdate(client Client, update Update) error {
	if update.Message != nil && client.Answer(update.Message) {
		return nil
	}
	cmd := l.extractCommand(update)
	if cmd == nil {
		return nil
	}
	if listener, ok := l.handlers[cmd.Key]; ok {
		err := listener.HandleCommand(client, cmd)
		if err != nil {
			return errors.Wrapf(err, "while handling %v", update)
		}
	}
	return nil
}

func (l *CommandListener) AllowedUpdates() []string {
	return []string{"message", "edited_message", "callback_query"}
}

func (l *CommandListener) extractCommand(update Update) *Command {
	switch {
	case update.Message != nil:
		return l.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return l.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return l.extractCommandCallbackQuery(update.CallbackQuery)
	}
	return nil
}

func (l *CommandListener) extractCommandMessage(message *Message) *Command {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			key := message.Text[entity.Offset : entity.Offset+entity.Length]
			at := strings.Index(key, "@")
			if at > 0 && len(key) > at && l.username == key[at+1:] {
				key = key[:at]
			}
			return &Command{
				User:    &message.From,
				Chat:    &message.Chat,
				Message: message,
				Key:     key,
				Payload: strings.Trim(message.Text[entity.Offset+entity.Length:], " "),
			}
		}
	}
	return nil
}

func (l *CommandListener) extractCommandCallbackQuery(query *CallbackQuery) *Command {
	if query.Data == nil {
		return nil
	}
	for i, c := range *query.Data {
		if c == ':' && len(*query.Data) > i+1 {
			return &Command{
				Chat:            &query.Message.Chat,
				User:            &query.From,
				Message:         query.Message,
				Key:             (*query.Data)[:i],
				Payload:         (*query.Data)[i+1:],
				CallbackQueryID: query.ID,
			}
		}
	}
	return nil
}

func InlineKeyboard(rows ...[][3]string) ReplyMarkup {
	keyboard := make([][]InlineKeyboardButton, len(rows))
	for i, row := range rows {
		keyboard[i] = make([]InlineKeyboardButton, len(row))
		for j, button := range row {
			keyboard[i][j] = InlineKeyboardButton{
				Text:         button[0],
				CallbackData: button[1] + ":" + button[2],
			}
		}
	}
	return &InlineKeyboardMarkup{keyboard}
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

func (c *Command) Reply(client Client, text string) error {
	if c.CallbackQueryID != "" {
		_, err := client.AnswerCallbackQuery(c.CallbackQueryID, &AnswerCallbackQueryOptions{Text: text})
		return err
	} else {
		_, err := client.Send(c.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: c.Message.ID})
		return err
	}
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
