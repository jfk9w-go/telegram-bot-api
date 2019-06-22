package telegram

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
)

// GlobalSendDelay is a delay between two consecutive /send* API calls per bot token.
var GlobalSendDelay = 30 * time.Millisecond

// SendDelays are delays between two consecutive /send* API calls per chat with a given type.
var SendDelays = map[ChatType]time.Duration{
	PrivateChat: 30 * time.Millisecond,
	GroupChat:   time.Second,
	Supergroup:  time.Second,
	Channel:     0,
}

// Bot is a Telegram Bot instance.
// It enhances basic Telegram Bot API client with flood control awareness.
// All /send* API calls are executed with certain delays to keep them "under the radar".
// In addition to Telegram Bot API client functionality
// it provides an interface to listen to incoming updates and
// reacting to them.
type Bot struct {
	*Client

	sendQueue   chan *sendRequest
	maxRetries  int
	sendQueueWG *sync.WaitGroup

	queues map[ChatID]chan *sendRequest
	mu     *sync.RWMutex
	wg     *sync.WaitGroup
}

type sendRequest struct {
	url   string
	body  flu.BodyWriter
	resp  interface{}
	err   error
	retry int
	done  chan struct{}
}

// NewBot creates a new Bot instance.
// If httpClient is nil, a default flu.Client will be created.
func NewBot(http *flu.Client, token string) *Bot {
	if token == "" {
		panic("token must not be empty")
	}

	client := newClient(http, token)
	sendQueue := make(chan *sendRequest, 1000)
	bot := &Bot{
		Client:      client,
		sendQueue:   sendQueue,
		maxRetries:  3,
		sendQueueWG: new(sync.WaitGroup),
		queues:      make(map[ChatID]chan *sendRequest),
		mu:          new(sync.RWMutex),
		wg:          new(sync.WaitGroup),
	}

	go bot.runSendWorker()
	return bot
}

func (b *Bot) runSendWorker() {
	b.sendQueueWG.Add(1)
	defer b.sendQueueWG.Done()
	for req := range b.sendQueue {
		err := b.Client.send(req.url, req.body, req.resp)
		if err != nil {
			if floodErr, ok := err.(*TooManyMessages); ok {
				log.Println(strings.Title(err.Error()))
				time.Sleep(floodErr.RetryAfter)
				b.sendQueue <- req
			} else if req.retry < b.maxRetries {
				req.retry++
				b.sendQueue <- req
			} else {
				req.err = err
				req.done <- struct{}{}
			}
		} else {
			req.done <- struct{}{}
		}

		time.Sleep(GlobalSendDelay)
	}
}

func (b *Bot) runWorker(queue chan *sendRequest, delay time.Duration) {
	b.wg.Add(1)
	defer b.wg.Done()
	for req := range queue {
		b.sendQueue <- req
		time.Sleep(delay)
	}
}

var uninitializedQueueErr = errors.New("queue not initialized")

// Send is an umbrella method for various /send* API calls.
// Generally BaseSendable is either string (/sendMessage, media links in /sendPhoto and others)
// or flu.ReadResource (when sending a local file in /sendPhoto and others).
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
func (b *Bot) Send(chatID ChatID, sendable Sendable, opts *SendOpts) (*Message, error) {
	m := new(Message)
	err := b.send(chatID, sendable, opts, m)
	if err == uninitializedQueueErr {
		b.initializeQueue(&m.Chat)
		err = nil
	}

	return m, err
}

func (b *Bot) SendMediaGroup(chatID ChatID, media []Media, opts *SendOpts) ([]Message, error) {
	ms := make([]Message, 0)
	err := b.send(chatID, MediaGroup(media), opts, &ms)
	if err == uninitializedQueueErr {
		b.initializeQueue(&ms[0].Chat)
		err = nil
	}

	return ms, err
}

func (b *Bot) initializeQueue(chat *Chat) {
	hasUsername := chat.Username != nil
	var queue chan *sendRequest = nil

	b.mu.Lock()
	if q, ok := b.queues[chat.ID]; ok {
		queue = q
	} else if hasUsername {
		if q, ok := b.queues[*chat.Username]; ok {
			queue = q
		}
	}

	ok := false
	if queue == nil {
		queue = make(chan *sendRequest, 500)
		ok = true
	}

	b.queues[chat.ID] = queue
	if hasUsername {
		b.queues[*chat.Username] = queue
	}

	b.mu.Unlock()

	if ok {
		log.Println("Created queue for", chat.ID)
		go b.runWorker(queue, SendDelays[chat.Type])
	}
}

var emptySendOpts = &SendOpts{}

func (b *Bot) send(chatID ChatID, sendable BaseSendable, opts *SendOpts, resp interface{}) error {
	if opts == nil {
		opts = emptySendOpts
	}

	var form *flu.FormBody
	if sendable.encode() {
		form = flu.Form(sendable)
	} else {
		form = flu.Form()
	}

	form.Set("chat_id", chatID.queryParam())
	if opts != nil {
		if opts.DisableNotification {
			form.Set("disable_notification", "1")
		}

		if opts.ReplyToMessageID != 0 {
			form.Set("reply_to_message_id", opts.ReplyToMessageID.queryParam())
		}

		if opts.ReplyMarkup != nil {
			bytes, err := json.Marshal(opts.ReplyMarkup)
			if err != nil {
				return errors.Wrap(err, "failed to serialize reply_markup")
			}

			form.Set("reply_markup", string(bytes))
		}
	}

	url := b.method("/send" + strings.Title(sendable.kind()))
	body, err := sendable.finalize(form)
	if err != nil {
		return errors.Wrap(err, "failed to finalize send data")
	}

	queueExists := false
	req := &sendRequest{url, body, resp, nil, 0, make(chan struct{}, 1)}

	b.mu.RLock()
	if queue, ok := b.queues[chatID]; ok {
		queue <- req
		queueExists = true
	}

	b.mu.RUnlock()

	if !queueExists {
		b.sendQueue <- req
	}

	<-req.done
	if req.err == nil && !queueExists {
		return uninitializedQueueErr
	}

	return req.err
}

// Listen subscribes a listener to incoming updates channel.
func (b *Bot) Listen(listener UpdateListener) {
	updateCh := make(chan Update)
	updatesOpts := &UpdatesOpts{TimeoutSecs: 60, AllowedUpdates: listener.AllowedUpdates()}
	go b.runUpdatesChan(updateCh, updatesOpts)
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
				opts.Offset = update.ID.Increment()
			}

			continue
		}

		if err != nil {
			log.Printf("An error occured while polling: %v", err)
			time.Sleep(time.Minute)
		}
	}
}
