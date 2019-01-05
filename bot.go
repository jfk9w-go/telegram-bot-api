package telegram

import (
	"log"
	"time"

	"github.com/jfk9w-go/flu"
)

type Bot struct {
	*Client
	*SendQueues
}

func NewBot(http *flu.Client, token string) *Bot {
	if token == "" {
		panic("token must not be empty")
	}

	client := newClient(http, token)
	return &Bot{
		Client:     client,
		SendQueues: newSendQueues(client),
	}
}

func (b *Bot) Listen(listener UpdateListener) {
	listener.SetBot(b)
	updateCh := make(chan Update)
	go b.runUpdatesChan(updateCh, new(UpdatesOpts).
		SetTimeout(time.Minute).
		SetAllowedUpdates(listener.AllowedUpdates()...))
	for update := range updateCh {
		go listener.OnUpdate(update)
	}
}

func (b *Bot) runUpdatesChan(updateCh chan<- Update, opts *UpdatesOpts) {
	for {
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
}
