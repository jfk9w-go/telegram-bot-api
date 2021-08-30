package telegram

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// GatewaySendDelay is a delay between two consecutive /send* API calls per bot token.
var GatewaySendDelay = 35 * time.Millisecond

// SendDelays are delays between two consecutive /send* API calls per chat with a given type.
var SendDelays = map[ChatType]time.Duration{
	PrivateChat: 35 * time.Millisecond,
	GroupChat:   3 * time.Second,
	Supergroup:  time.Second,
	Channel:     3 * time.Second,
}

var MaxSendRetries = 3

type Executor interface {
	Execute(ctx context.Context, method string, body flu.EncoderTo, resp interface{}) error
}

type FloodControlAware struct {
	executor    Executor
	rateLimiter flu.RateLimiter
	recipients  map[ChatID]flu.RateLimiter
	mu          flu.RWMutex
}

func FloodControl(executor Executor) *FloodControlAware {
	return &FloodControlAware{
		executor:    executor,
		rateLimiter: flu.IntervalRateLimiter(GatewaySendDelay),
		recipients:  make(map[ChatID]flu.RateLimiter),
	}
}

var errUnknownRecipient = errors.New("unknown recipient")

func (c *FloodControlAware) send(ctx context.Context, chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}

	method := "send" + strings.Title(item.kind())
	limiter, ok := c.getRecipient(chatID)
	if ok {
		if err := limiter.Start(ctx); err != nil {
			return err
		}
		defer limiter.Complete()
	}

	if err := c.rateLimiter.Start(ctx); err != nil {
		return err
	}
	defer c.rateLimiter.Complete()
	for i := 0; i <= MaxSendRetries; i++ {
		err = c.executor.Execute(ctx, method, body, resp)
		var timeout time.Duration
		switch err := err.(type) {
		case nil:
			if ok {
				return nil
			} else {
				return errUnknownRecipient
			}
		case TooManyMessages:
			logrus.Warnf("too many messages, sleeping for %s...", err.RetryAfter)
			timeout = err.RetryAfter
		case Error:
			return err
		default:
			timeout = GatewaySendDelay
		}

		if err := flu.Sleep(ctx, timeout); err != nil {
			return err
		}
	}

	return err
}

func (c *FloodControlAware) getRecipient(chatID ChatID) (flu.RateLimiter, bool) {
	defer c.mu.RLock().Unlock()
	limiter, ok := c.recipients[chatID]
	return limiter, ok
}

func (c *FloodControlAware) newRecipient(chat *Chat) {
	defer c.mu.Lock().Unlock()
	if _, ok := c.recipients[chat.ID]; ok {
		return
	}

	limiter := flu.IntervalRateLimiter(chat.Type.SendDelay())
	c.recipients[chat.ID] = limiter
	if chat.Username != nil {
		c.recipients[*chat.Username] = limiter
	}
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
func (c *FloodControlAware) Send(ctx context.Context, chatID ChatID, item Sendable, options *SendOptions) (*Message, error) {
	m := new(Message)
	err := c.send(ctx, chatID, item, options, m)
	if err == errUnknownRecipient {
		c.newRecipient(&m.Chat)
		err = nil
	}
	return m, err
}

// SendMediaGroup is used to send a group of photos or videos as an album.
// On success, an array of Message's is returned.
// See https://core.telegram.org/bots/api#sendmediagroup
func (c *FloodControlAware) SendMediaGroup(ctx context.Context, chatID ChatID, media []Media, options *SendOptions) ([]Message, error) {
	ms := make([]Message, 0)
	err := c.send(ctx, chatID, MediaGroup(media), options, &ms)
	if err == errUnknownRecipient {
		c.newRecipient(&ms[0].Chat)
		err = nil
	}
	return ms, err
}
