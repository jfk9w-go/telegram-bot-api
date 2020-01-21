package telegram

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type floodControlAwareClient struct {
	api
	maxRetries int
	gateway    flu.Restraint
	recipients map[ChatID]flu.Restraint
	mutex      sync.RWMutex
}

func newFloodControlAwareClient(api api, maxRetries int) *floodControlAwareClient {
	return &floodControlAwareClient{
		api:        api,
		maxRetries: maxRetries,
		gateway:    flu.NewIntervalRestraint(GatewaySendDelay),
		recipients: make(map[ChatID]flu.Restraint),
	}
}

var unknownRecipientErr = errors.New("unknown recipient")

func (c *floodControlAwareClient) send(chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}
	url := c.method("/send" + strings.Title(item.kind()))
	c.mutex.RLock()
	rec, exists := c.recipients[chatID]
	c.mutex.RUnlock()
	if exists {
		rec.Start()
		defer rec.Complete()
	}
	c.gateway.Start()
	defer c.gateway.Complete()
	for i := 0; i <= c.maxRetries; i++ {
		err = c.api.send(url, body, resp)
		switch err := err.(type) {
		case nil:
			if exists {
				return nil
			} else {
				return unknownRecipientErr
			}
		case TooManyMessages:
			log.Printf("Too many messages, sleeping for %s...", err.RetryAfter)
			time.Sleep(err.RetryAfter)
			continue
		case Error:
			return err
		default:
			time.Sleep(GatewaySendDelay)
		}
	}
	return err
}

func (c *floodControlAwareClient) newRecipient(chat *Chat) {
	c.mutex.Lock()
	if _, ok := c.recipients[chat.ID]; !ok {
		restraint := flu.NewIntervalRestraint(SendDelays[chat.Type])
		c.recipients[chat.ID] = restraint
		if chat.Username != nil {
			c.recipients[*chat.Username] = restraint
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
func (c *floodControlAwareClient) Send(chatID ChatID, item Sendable, options *SendOptions) (*Message, error) {
	m := new(Message)
	err := c.send(chatID, item, options, m)
	if err == unknownRecipientErr {
		c.newRecipient(&m.Chat)
		err = nil
	}
	return m, err
}

// Use this method to send a group of photos or videos as an album.
// On success, an array of the workers Messages is returned.
// See https://core.telegram.org/bots/api#sendmediagroup
func (c *floodControlAwareClient) SendMediaGroup(chatID ChatID, media []Media, options *SendOptions) ([]Message, error) {
	ms := make([]Message, 0)
	err := c.send(chatID, MediaGroup(media), options, &ms)
	if err == unknownRecipientErr {
		c.newRecipient(&ms[0].Chat)
		err = nil
	}
	return ms, err
}
