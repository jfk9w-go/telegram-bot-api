package telegram

import (
	"github.com/jfk9w-go/flu"
)

// Bot is a Telegram Bot instance.
// It enhances basic Telegram Bot API apiClient with flood control awareness.
// All /send* API calls are executed with certain delays to keep them "under the radar".
// In addition to Telegram Bot API apiClient functionality
// it provides an interface to listen to incoming updates and
// reacting to them.
type Bot = *updater

// NewBot creates a new Bot instance.
// If http is nil, a default flu.apiClient will be created.
func NewBot(http *flu.Client, token string) Bot {
	api := newApiClient(http, token)
	client := newClient(api, 3)
	updater := newUpdater(client, &GetUpdatesOptions{TimeoutSecs: 60})
	return updater
}
