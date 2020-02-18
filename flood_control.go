package telegram

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type floodControlAwareClient struct {
	api
	maxRetries  int
	rateLimiter flu.RateLimiter
	recipients  map[ChatID]flu.RateLimiter
	mutex       sync.RWMutex
}

func newFloodControlAwareClient(api api, maxRetries int) *floodControlAwareClient {
	return &floodControlAwareClient{
		api:         api,
		maxRetries:  maxRetries,
		rateLimiter: flu.IntervalRateLimiter(GatewaySendDelay),
		recipients:  make(map[ChatID]flu.RateLimiter),
	}
}

var unknownRecipientErr = errors.New("unknown recipient")

func (c *floodControlAwareClient) send(ctx context.Context, chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}
	url := c.method("/send" + strings.Title(item.kind()))
	c.mutex.RLock()
	limiter, exists := c.recipients[chatID]
	c.mutex.RUnlock()
	if exists {
		if err := limiter.Start(ctx); err != nil {
			return err
		}
		defer limiter.Complete()
	}
	if err := c.rateLimiter.Start(ctx); err != nil {
		return err
	}
	defer c.rateLimiter.Complete()
	for i := 0; i <= c.maxRetries; i++ {
		err = c.api.send(ctx, url, body, resp)
		var timeout time.Duration
		switch err := err.(type) {
		case nil:
			if exists {
				return nil
			} else {
				return unknownRecipientErr
			}
		case TooManyMessages:
			log.Printf("Too many messages, sleeping for %s...", err.RetryAfter)
			timeout = err.RetryAfter
		case Error:
			return err
		default:
			timeout = GatewaySendDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeout):
		}
	}
	return err
}

func (c *floodControlAwareClient) newRecipient(chat *Chat) {
	c.mutex.Lock()
	if _, ok := c.recipients[chat.ID]; !ok {
		rateLimiter := flu.IntervalRateLimiter(SendDelays[chat.Type])
		c.recipients[chat.ID] = rateLimiter
		if chat.Username != nil {
			c.recipients[*chat.Username] = rateLimiter
		}
	}
	c.mutex.Unlock()
}

// Send is an umbrella method for various /send* API calls which return only one Message.
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
//   https://core.telegram.org/bots/api#senddocument
//   https://core.telegram.org/bots/api#sendaudio
//   https://core.telegram.org/bots/api#sendvoice
//   https://core.telegram.org/bots/api#sendsticker
func (c *floodControlAwareClient) Send(ctx context.Context, chatID ChatID, item Sendable, options *SendOptions) (*Message, error) {
	m := new(Message)
	err := c.send(ctx, chatID, item, options, m)
	if err == unknownRecipientErr {
		c.newRecipient(&m.Chat)
		err = nil
	}
	return m, err
}

// Use this method to send a group of photos or videos as an album.
// On success, an array of the workers Messages is returned.
// See https://core.telegram.org/bots/api#sendmediagroup
func (c *floodControlAwareClient) SendMediaGroup(ctx context.Context, chatID ChatID, media []Media, options *SendOptions) ([]Message, error) {
	ms := make([]Message, 0)
	err := c.send(ctx, chatID, MediaGroup(media), options, &ms)
	if err == unknownRecipientErr {
		c.newRecipient(&ms[0].Chat)
		err = nil
	}
	return ms, err
}
