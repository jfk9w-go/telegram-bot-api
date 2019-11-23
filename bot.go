package telegram

import (
	"github.com/jfk9w-go/flu"
)

// Bot is a Telegram Bot instance.
// It enhances basic Telegram Bot API client with flood control awareness.
// All /send* API calls are executed with certain delays to keep them "under the radar".
// In addition to Telegram Bot API client functionality
// it provides an interface to listen to incoming updates and
// reacting to them.
type Bot = *updateClient

// NewBot creates a new Bot instance.
// If http is nil, a default flu.client will be created.
func NewBot(http *flu.Client, token string) Bot {
	base := newClient(http, token)
	send := newSendClient(base, 3)
	update := newUpdateClient(send, &GetUpdatesOptions{TimeoutSecs: 60})
	return update
}
