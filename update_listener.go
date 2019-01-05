package telegram

import (
	"fmt"
	"log"
	"strings"
)

type UpdateListener interface {
	OnUpdate(Update)
	AllowedUpdates() []string
	SetBot(b *Bot)
}

type CommandUpdateListener struct {
	b         *Bot
	listeners map[string]CommandListener
}

func NewCommandUpdateListener() *CommandUpdateListener {
	return &CommandUpdateListener{nil, make(map[string]CommandListener)}
}

func (cul *CommandUpdateListener) Add(key string, listener CommandListener) *CommandUpdateListener {
	if _, ok := cul.listeners[key]; ok {
		panic("command listener for " + key + " already registered")
	}

	cul.listeners[key] = listener
	return cul
}

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

type Command struct {
	Chat      *Chat
	User      *User
	MessageID ID
	Key       string
	Payload   string

	b *Bot
}

func (c *Command) reply(text string) {
	_, err := c.b.send(c.Chat.ID, text, NewSendOpts().
		DisableNotification(true).
		ReplyToMessageId(c.MessageID).
		Message().
		DisableWebPagePreview(true))

	if err != nil {
		log.Printf("Failed to send reply (%s) to chat %v, message %v: %s\n",
			text, c.Chat.ID, c.MessageID, err)
	}
}

func (c *Command) TextReply(text string) {
	c.reply(text)
}

func (c *Command) ErrorReply(err error) {
	c.reply(fmt.Sprintf("an error occured: %s", err))
}

type CommandListener interface {
	OnCommand(*Command)
}

type CommandListenerFunc func(*Command)

func (fcl CommandListenerFunc) OnCommand(cmd *Command) {
	fcl(cmd)
}
