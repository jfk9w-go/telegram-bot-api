package telegram

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type sendTask struct {
	url   string
	body  flu.BodyEncoderTo
	resp  interface{}
	err   error
	retry int
	work  sync.WaitGroup
}

func newSendTask(url string, body flu.BodyEncoderTo, resp interface{}) *sendTask {
	t := &sendTask{url: url, body: body, resp: resp}
	t.work.Add(1)
	return t
}

func (t *sendTask) complete() {
	t.work.Done()
}

func (t *sendTask) wait() {
	t.work.Wait()
}

type sendQueue = chan *sendTask

type sendClient struct {
	*client
	maxRetries int
	gateway    sendQueue
	recipients map[ChatID]sendQueue
	mutex      sync.RWMutex
	workers    sync.WaitGroup
	once       *sync.Once
}

func newSendClient(client *client, maxRetries int) *sendClient {
	return &sendClient{
		client:     client,
		maxRetries: maxRetries,
		once:       new(sync.Once),
	}
}

func (c *sendClient) init() {
	c.gateway = make(sendQueue, 1000)
	c.recipients = make(map[ChatID]sendQueue)
	go c.runGateway()
	log.Printf("Initialized send client")
}

func (c *sendClient) runGateway() {
	c.workers.Add(1)
	defer c.workers.Done()
	for task := range c.gateway {
	taskloop:
		for {
			err := c.send(task.url, task.body, task.resp)
			switch err := err.(type) {
			case nil:
				break taskloop

			case *TooManyMessages:
				log.Printf("Too many messages, sleeping for %s...", err.RetryAfter)
				time.Sleep(err.RetryAfter)
				continue

			default:
				task.retry++
				if task.retry > c.maxRetries {
					task.err = err
					break taskloop
				} else {
					time.Sleep(GatewaySendDelay)
				}
			}
		}

		task.complete()
		time.Sleep(GatewaySendDelay)
	}
}

func (c *sendClient) runWorker(queue sendQueue, delay time.Duration) {
	c.workers.Add(1)
	defer c.workers.Done()
	for t := range queue {
		c.gateway <- t
		time.Sleep(delay)
	}
}

var recipientErr = errors.New("unknown recipient")

func (c *sendClient) submitAndWait(chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	c.once.Do(c.init)
	url := c.method("/send" + strings.Title(item.kind()))
	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}

	exists := false
	task := newSendTask(url, body, resp)

	c.mutex.RLock()
	if queue, ok := c.recipients[chatID]; ok {
		queue <- task
		exists = true
	}

	c.mutex.RUnlock()

	if !exists {
		c.gateway <- task
	}

	task.work.Wait()
	if task.err == nil && !exists {
		return recipientErr
	}

	return task.err
}

func (c *sendClient) newRecipient(chat *Chat) {
	hasUsername := chat.Username != nil
	var queue sendQueue = nil

	c.mutex.Lock()
	if q, ok := c.recipients[chat.ID]; ok {
		queue = q
	} else if hasUsername {
		if q, ok := c.recipients[*chat.Username]; ok {
			queue = q
		}
	}

	ok := false
	if queue == nil {
		queue = make(chan *sendTask, 100)
		ok = true
	}

	c.recipients[chat.ID] = queue
	if hasUsername {
		c.recipients[*chat.Username] = queue
	}

	c.mutex.Unlock()

	if ok {
		go c.runWorker(queue, SendDelays[chat.Type])
	}
}

// Send is an umbrella method for various /send* API calls which return only one Message.
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
//   https://core.telegram.org/bots/api#senddocument
func (c *sendClient) Send(chatID ChatID, item Sendable, options *SendOptions) (*Message, error) {
	m := new(Message)
	err := c.submitAndWait(chatID, item, options, m)
	if err == recipientErr {
		c.newRecipient(&m.Chat)
		err = nil
	}

	return m, err
}

// Use this method to send a group of photos or videos as an album.
// On success, an array of the workers Messages is returned.
// See https://core.telegram.org/bots/api#sendmediagroup
func (c *sendClient) SendMediaGroup(chatID ChatID, media []Media, options *SendOptions) ([]Message, error) {
	ms := make([]Message, 0)
	err := c.submitAndWait(chatID, MediaGroup(media), options, &ms)
	if err == recipientErr {
		c.newRecipient(&ms[0].Chat)
		err = nil
	}

	return ms, err
}
