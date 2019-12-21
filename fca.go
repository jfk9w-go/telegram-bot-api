package telegram

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Client = *floodControlAwareClient

type floodControlAwareClient struct {
	api
	maxRetries int
	gateway    Restraint
	recipients map[ChatID]Restraint
	mutex      sync.RWMutex
}

func newClient(api api, maxRetries int) Client {
	return &floodControlAwareClient{
		api:        api,
		maxRetries: maxRetries,
		gateway:    NewIntervalRestraint(GatewaySendDelay),
		recipients: make(map[ChatID]Restraint),
	}
}

var unknownRecipientErr = errors.New("unknown recipient")

func (c *floodControlAwareClient) send(chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}
	url := c.method("/send" + strings.Title(item.kind()))
	exists := false
	c.mutex.RLock()
	if rec, ok := c.recipients[chatID]; ok {
		rec.Start()
		rec.Complete()
		exists = true
	}
	c.mutex.RUnlock()
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
		default:
			time.Sleep(GatewaySendDelay)
		}
	}
	return err
}

func (c *floodControlAwareClient) newRecipient(chat *Chat) {
	hasUsername := chat.Username != nil
	var restraint Restraint
	c.mutex.Lock()
	if rec, ok := c.recipients[chat.ID]; ok {
		restraint = rec
	} else if hasUsername {
		if rec, ok := c.recipients[*chat.Username]; ok {
			restraint = rec
		}
	}
	if restraint == nil {
		restraint = NewIntervalRestraint(SendDelays[chat.Type])
	}
	c.recipients[chat.ID] = restraint
	if hasUsername {
		c.recipients[*chat.Username] = restraint
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
