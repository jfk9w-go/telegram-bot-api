package telegram

import (
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type floodControl struct {
	event    chan time.Time
	interval time.Duration
}

func newFloodControl(interval time.Duration) floodControl {
	event := make(chan time.Time, 1)
	event <- time.Unix(0, 0)
	return floodControl{event, interval}
}

func (fc floodControl) start() {
	prev := <-fc.event
	time.Sleep(fc.interval - time.Now().Sub(prev))
}

func (fc floodControl) complete() {
	fc.event <- time.Now()
}

type floodControlAwareClient struct {
	*apiClient
	maxRetries int
	gateway    floodControl
	recipients map[ChatID]floodControl
	mutex      sync.RWMutex
}

func newClient(api *apiClient, maxRetries int) Client {
	return &floodControlAwareClient{
		apiClient:  api,
		maxRetries: maxRetries,
		gateway:    newFloodControl(GatewaySendDelay),
		recipients: make(map[ChatID]floodControl),
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
		rec.start()
		rec.complete()
		exists = true
	}
	c.mutex.RUnlock()

	c.gateway.start()
	err = c.apiClient.send(url, body, resp)
	c.gateway.complete()

	if err == nil && !exists {
		return unknownRecipientErr
	} else {
		return err
	}
}

func (c *floodControlAwareClient) newRecipient(chat *Chat) {
	hasUsername := chat.Username != nil
	var control floodControl
	c.mutex.Lock()
	if rec, ok := c.recipients[chat.ID]; ok {
		control = rec
	} else if hasUsername {
		if rec, ok := c.recipients[*chat.Username]; ok {
			control = rec
		}
	}
	if control == (floodControl{}) {
		control = newFloodControl(SendDelays[chat.Type])
	}
	c.recipients[chat.ID] = control
	if hasUsername {
		c.recipients[*chat.Username] = control
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
