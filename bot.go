package telegram

import (
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
)

type Bot struct {
	BotApi
	sendQueues  *sendQueues
	updates     chan Update
	updatesOnce sync.Once
}

func NewBot(client *flu.Client, token string) *Bot {
	var api = NewBotApi(client, token)
	return &Bot{
		BotApi:     api,
		sendQueues: newSendQueues(api),
	}
}

func (bot *Bot) Send(chatId ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	return bot.sendQueues.send(chatId, entity, opts)
}

func (bot *Bot) GetUpdatesChan(opts UpdatesOpts) <-chan Update {
	bot.updatesOnce.Do(func() {
		bot.updates = make(chan Update)
		go bot.runUpdatesChan(opts)
	})

	return bot.updates
}

func (bot *Bot) runUpdatesChan(opts UpdatesOpts) {
	for {
		var batch, err = bot.GetUpdates(opts)
		if err == nil {
			if len(batch) > 0 {
				log.Printf("Received %d updates", len(batch))
			}

			for _, update := range batch {
				bot.updates <- update
				opts = opts.WithOffset(ID(update.ID.Int64Value() + 1))
			}

			continue
		}

		if err != nil {
			log.Printf("An error occured while receiving updates: %v", err)
			time.Sleep(time.Minute)
		}
	}
}
