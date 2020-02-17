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

func (o GetUpdatesOptions) body() flu.BodyEncoderTo {
	return flu.JSON(o)
}

var (
	DefaultUpdateOptions = GetUpdatesOptions{TimeoutSecs: 60}
	MaxSendRetries       = 3
)

type Bot struct {
	Client
	options GetUpdatesOptions
	cancel  context.CancelFunc
	work    sync.WaitGroup
}

func NewBot(http *flu.Client, token string) *Bot {
	api := newApi(http, token)
	fca := newFloodControlAwareClient(api, MaxSendRetries)
	c := newConversationAwareClient(fca)
	return &Bot{
		Client:  c,
		options: DefaultUpdateOptions,
	}
}

func (bot *Bot) Listen(ctx context.Context, concurrency int, listener UpdateListener) {
	bot.options.AllowedUpdates = listener.AllowedUpdates()
	log.Printf("Listening for the following updates: %v", bot.options.AllowedUpdates)
	channel := make(chan Update)
	if concurrency < 1 {
		concurrency = 1
	}

	bot.work.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		ctx, _ := context.WithCancel(ctx)
		go func() {
			defer bot.work.Done()
			for update := range channel {
				err := listener.ReceiveUpdate(ctx, bot.Client, update)
				if err != nil {
					log.Printf("Failed to process update %d: %s", update.ID, err)
				}
			}
		}()
	}

	defer close(channel)
	for {
		updates, err := bot.GetUpdates(ctx, bot.options)
		switch {
		case ctx.Err() != nil:
			return
		case err != nil:
			log.Printf("Telegram bot poll error: %s", err)
			time.Sleep(time.Duration(bot.options.TimeoutSecs) * time.Second)
			continue
		}

		for _, update := range updates {
			if update.Message != nil {
				if err := bot.Answer(ctx, update.Message); err != nil {
					log.Printf("Interrupting update listener because of %s", err)
					return
				}
			}

			channel <- update
			bot.options.Offset = update.ID.Increment()
		}
	}
}

func (bot *Bot) Shutdown(cancel context.CancelFunc) {
	cancel()
	bot.work.Wait()
}
