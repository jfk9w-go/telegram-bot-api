package telegram

import (
	"strings"
)

// UpdateListener is a handler for incoming Updates.
type UpdateListener interface {
	// ReceiveUpdate is called on every received Update.
	ReceiveUpdate(Update)
	// AllowedUpdates_ is the allowed_updates parameter passed
	// in API calls to /getUpdates or /setWebhook.
	AllowedUpdates() []string
}

// CommandListener is a UpdateListener handling incoming bot commands
// with message and edited_message allowed updates.
type CommandListener struct {
	bot      *Bot
	handlers map[string]CommandHandler
}

// NewCommandListener creates a new instance of CommandListener.
func NewCommandListener(bot *Bot) *CommandListener {
	return &CommandListener{bot, make(map[string]CommandHandler)}
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

// HandleFunc is a shortcut for Handle(key, CommandListerFunc(func (*Command) {...}))
func (l *CommandListener) HandleFunc(key string, handler CommandHandlerFunc) *CommandListener {
	return l.Handle(key, handler)
}

func (l *CommandListener) ReceiveUpdate(update Update) {
	cmd := extractCommand(update)
	if cmd == nil {
		return
	}

	cmd.bot = l.bot
	if listener, ok := l.handlers[cmd.Key]; ok {
		err := listener.HandleCommand(cmd)
		if err != nil {
			cmd.Reply(err.Error())
		}
	}
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

func extractCommandCallbackQuery(callbackQuery *CallbackQuery) *Command {
	if callbackQuery.Data == nil {
		return nil
	}

	for i, c := range *callbackQuery.Data {
		if c == ':' && len(*callbackQuery.Data) > i+1 {
			return &Command{
				Chat:            &callbackQuery.Message.Chat,
				User:            &callbackQuery.From,
				MessageID:       callbackQuery.Message.ID,
				Key:             (*callbackQuery.Data)[:i],
				Payload:         (*callbackQuery.Data)[i+1:],
				callbackQueryID: &callbackQuery.ID,
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
	Chat      *Chat
	User      *User
	MessageID ID
	Key       string
	Payload   string

	callbackQueryID *string
	bot             *Bot
}

func (c *Command) Reply(text string) {
	var err error
	if c.callbackQueryID != nil {
		_, err = c.bot.AnswerCallbackQuery(*c.callbackQueryID, &AnswerCallbackQueryOpts{Text: text})
	} else if text != "" {
		_, err = c.bot.Send(c.Chat.ID,
			&Text{Text: text, DisableWebPagePreview: true},
			&SendOpts{DisableNotification: true, ReplyToMessageID: c.MessageID})
	}

	if err != nil {
		println("reply to ", c.Chat.ID, " error: ", err)
	}
}

// CommandHandler describes a bot command handler.
type CommandHandler interface {
	HandleCommand(*Command) error
}

// CommandHandlerFunc implements CommandHandler interface for lambdas.
type CommandHandlerFunc func(*Command) error

func (f CommandHandlerFunc) HandleCommand(cmd *Command) error {
	return f(cmd)
}
