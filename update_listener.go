package telegram

import (
	"fmt"
	"log"
	"strings"
)

// UpdateListener is a handler for incoming Updates.
type UpdateListener interface {
	// OnUpdate is called on every received Update.
	OnUpdate(Update)
	// AllowedUpdates is the allowed_updates parameter passed
	// in API calls to /getUpdates or /setWebhook.
	AllowedUpdates() []string
	// SetBot is an internal method used for injecting Bot instance.
	SetBot(b *Bot)
}

// CommandUpdateListener is an UpdateListener handling incoming bot commands
// with message and edited_message allowed updates.
type CommandUpdateListener struct {
	b         *Bot
	listeners map[string]CommandListener
}

// NewCommandUpdateListener creates a new instance of CommandUpdateListener.
func NewCommandUpdateListener() *CommandUpdateListener {
	return &CommandUpdateListener{nil, make(map[string]CommandListener)}
}

// Add binds a CommandListener to a command.
// Panics if the binding already exists.
func (cul *CommandUpdateListener) Add(key string, listener CommandListener) *CommandUpdateListener {
	if _, ok := cul.listeners[key]; ok {
		panic("command listener for " + key + " already registered")
	}

	cul.listeners[key] = listener
	return cul
}

// AddFunc is a shortcut for Add(key, CommandListerFunc(func (*Command) {...}))
func (cul *CommandUpdateListener) AddFunc(key string, listener CommandListenerFunc) *CommandUpdateListener {
	return cul.Add(key, listener)
}

func (cul *CommandUpdateListener) OnUpdate(update Update) {
	cmd := extractCommand(update)
	if cmd == nil {
		return
	}

	cmd.b = cul.b
	if listener, ok := cul.listeners[cmd.Key]; ok {
		listener.OnCommand(cmd)
	}
}

func (cul *CommandUpdateListener) AllowedUpdates() []string {
	return []string{"message", "edited_message"}
}

func (cul *CommandUpdateListener) SetBot(b *Bot) {
	cul.b = b
}

func extractCommand(update Update) *Command {
	var message *Message
	switch {
	case update.Message != nil:
		message = update.Message
	case update.EditedMessage != nil:
		message = update.EditedMessage
	default:
		return nil
	}

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

// Command is a text bot command.
type Command struct {
	Chat      *Chat
	User      *User
	MessageID ID
	Key       string
	Payload   string

	b *Bot
}

func (c *Command) reply(text string) {
	_, err := c.b.Send(c.Chat.ID, text, NewSendOpts().
		DisableNotification(true).
		ReplyToMessageID(c.MessageID).
		Message().
		DisableWebPagePreview(true))

	if err != nil {
		log.Printf("Failed to send reply (%s) to chat %v, message %v: %s\n",
			text, c.Chat.ID, c.MessageID, err)
	}
}

// TextReply replies to the message containing the initial command.
func (c *Command) TextReply(text string) {
	c.reply(text)
}

// ErrorReply replies with an error to the message containing the initial command.
func (c *Command) ErrorReply(err error) {
	c.reply(fmt.Sprintf("an error occured: %s", err))
}

// CommandListener describes a bot command handler.
type CommandListener interface {
	OnCommand(*Command)
}

// CommandListenerFunc implements CommandListener interface for lambdas.
type CommandListenerFunc func(*Command)

func (fcl CommandListenerFunc) OnCommand(cmd *Command) {
	fcl(cmd)
}
