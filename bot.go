package telegram

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/jfk9w-go/flu"
)

// Bot is a Telegram Bot instance.
// In addition to Telegram Bot API client functionality
// it provides an interface to listen to incoming updates and
// reacting to them.
type Bot struct {
	*Client
	*SendQueues
	active *int32
}

// NewBot creates a new Bot instance.
// If http is nil, a default flu.Client will be created.
func NewBot(http *flu.Client, token string) *Bot {
	if token == "" {
		panic("token must not be empty")
	}

	client := newClient(http, token)
	return &Bot{
		Client:     client,
		SendQueues: newSendQueues(client),
		active:     new(int32),
	}
}

// Listen subscribes a listener to incoming updates channel.
func (b *Bot) Listen(listener UpdateListener) {
	if !atomic.CompareAndSwapInt32(b.active, 0, 1) {
		panic("update listener already active")
	}

	listener.SetBot(b)
	updateCh := make(chan Update)
	go b.runUpdatesChan(updateCh, new(UpdatesOpts).
		SetTimeout(time.Minute).
		SetAllowedUpdates(listener.AllowedUpdates()...))
	for update := range updateCh {
		go listener.OnUpdate(update)
	}
}

// Close sets the bot status to false, meaning
// no more update requests will be made.
func (b *Bot) Close() {
	atomic.StoreInt32(b.active, 0)
}

func (b *Bot) runUpdatesChan(updateCh chan<- Update, opts *UpdatesOpts) {
	for atomic.LoadInt32(b.active) == 1 {
		batch, err := b.GetUpdates(opts)
		if err == nil {
			if len(batch) > 0 {
				log.Printf("Received %d updates", len(batch))
			}

			for _, update := range batch {
				updateCh <- update
				opts.SetOffset(update.ID.Increment())
			}

			continue
		}

		if err != nil {
			log.Printf("An error occured while polling: %v", err)
			time.Sleep(time.Minute)
		}
	}

	close(updateCh)
}
