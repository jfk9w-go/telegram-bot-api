package telegram

import (
	"log"
	"time"

	"github.com/jfk9w-go/flu"
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

func (o *GetUpdatesOptions) body() flu.BodyEncoderTo {
	return flu.JSON(o)
}

type updater struct {
	Client
	options *GetUpdatesOptions
}

func newUpdater(client Client, options *GetUpdatesOptions) *updater {
	return &updater{
		Client:  client,
		options: options,
	}
}

func (u *updater) Listen(listener UpdateListener) {
	u.options.AllowedUpdates = listener.AllowedUpdates()
	log.Printf("Listening for the following updates: %v", u.options.AllowedUpdates)
	for {
		updates, err := u.GetUpdates(u.options)
		if err == nil {
			for _, update := range updates {
				//noinspection ALL
				go listener.ReceiveUpdate(u.Client, update)
				u.options.Offset = update.ID.Increment()
			}
			continue
		}
		log.Printf("Poll error: %v", err)
		time.Sleep(time.Duration(u.options.TimeoutSecs) * time.Second)
	}
}
