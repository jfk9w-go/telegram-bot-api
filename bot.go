package telegram

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
)

// GetUpdatesOptions is /getUpdates request options.
// See https://core.telegram.org/bots/api#getupdates
type GetUpdatesOptions struct {
	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers of previously received Updates.
	// By default, Updates starting with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called with an offset
	// higher than its update_id. The negative offset can be specified to retrieve Updates
	// starting from -offset update from the end of the Updates queue.
	// All previous Updates will be forgotten.
	Offset ID `json:"offset,omitempty"`
	// Limits the number of Updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit int `json:"limit,omitempty"`
	// Timeout for long polling.
	TimeoutSecs int `json:"timeout,omitempty"`
	// List the types of Updates you want your bot to receive.
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

func (o GetUpdatesOptions) body() flu.BodyEncoderTo {
	return flu.JSON(o)
}

type ListenOptions struct {
	Updates              GetUpdatesOptions
	Concurrency          int
	ReceiveUpdateTimeout time.Duration
}

var DefaultListenOptions = ListenOptions{
	Updates:              GetUpdatesOptions{TimeoutSecs: 60},
	Concurrency:          5,
	ReceiveUpdateTimeout: 10 * time.Second,
}

type Bot struct {
	Client
	sync.WaitGroup
}

func NewBot(http *flu.Client, token string, sendRetries int) *Bot {
	api := newApi(http, token)
	fca := newFloodControlAwareClient(api, sendRetries)
	c := newConversationAwareClient(fca)
	return &Bot{
		Client: c,
	}
}

func (bot *Bot) Listen(ctx context.Context, options *ListenOptions, listener UpdateListener) {
	if options == nil {
		options = new(ListenOptions)
		*options = DefaultListenOptions
	}

	options.Updates.AllowedUpdates = listener.AllowedUpdates()
	log.Printf("Listening for the following updates: %v", options.Updates.AllowedUpdates)
	channel := make(chan Update)
	if options.Concurrency < 1 {
		options.Concurrency = 1
	}

	bot.Add(options.Concurrency)
	for i := 0; i < options.Concurrency; i++ {
		go func(ctx context.Context) {
			defer bot.Done()
			for update := range channel {
				ctx, cancel := context.WithTimeout(ctx, options.ReceiveUpdateTimeout)
				err := listener.ReceiveUpdate(ctx, bot.Client, update)
				if err != nil {
					log.Printf("Failed to process update %d: %s", update.ID, err)
				}

				cancel()
				if err == context.Canceled {
					break
				}
			}
		}(ctx)
	}

	defer close(channel)
	for {
		updates, err := bot.GetUpdates(ctx, options.Updates)
		if ctx.Err() != nil {
			return
		} else if err != nil {
			log.Printf("Telegram bot poll error: %s", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(options.Updates.TimeoutSecs) * time.Second):
				continue
			}
		}

		for _, update := range updates {
			if update.Message != nil && bot.Answer(update.Message) {
				// already answered
			} else {
				select {
				case <-ctx.Done():
					return
				case channel <- update:
				}
			}

			options.Updates.Offset = update.ID.Increment()
		}
	}
}
