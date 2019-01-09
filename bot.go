package telegram

import (
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/lego/pool"
)

type stream = pool.Pool

// UpstreamSendDelay is a delay between two consecutive /send* API calls per bot instance.
var UpstreamSendDelay = 30 * time.Millisecond

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
	upstream   stream
	workers    map[ChatType]*worker
	downstream map[ChatID]stream
	mu         *sync.RWMutex
}

// NewBot creates a new Bot instance.
// If http is nil, a default flu.Client will be created.
func NewBot(http *flu.Client, token string) *Bot {
	if token == "" {
		panic("token must not be empty")
	}

	client := newClient(http, token)

	upstream := pool.New().SpawnFunc(func(task *pool.Task) {
		ptr := task.Ptr.(*taskPtr)
		resp, err := client.send(ptr.chatID, ptr.entity, ptr.opts)
		if err != nil {
			if err, ok := err.(*TooManyMessages); ok {
				task.Retry()
				time.Sleep(err.RetryAfter)
				return
			}
		}

		ptr.resp = resp
		task.Complete(err)

		time.Sleep(UpstreamSendDelay)
	})

	workers := make(map[ChatType]*worker)
	for chatType, delay := range SendDelays {
		workers[chatType] = &worker{upstream, delay}
	}

	return &Bot{
		Client:     client,
		upstream:   upstream,
		workers:    workers,
		downstream: make(map[ChatID]pool.Pool),
		mu:         new(sync.RWMutex),
	}
}

// Send is an umbrella method for various /send* API calls.
// Generally entity is either string (/sendMessage, media links in /sendPhoto and others)
// or flu.ReadResource (when sending a local file in /sendPhoto and others).
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
func (b *Bot) Send(chatID ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	b.mu.RLock()
	stream, ok := b.downstream[chatID]
	b.mu.RUnlock()

	ptr := &taskPtr{chatID: chatID, entity: entity, opts: opts}
	if ok {
		err := stream.Execute(ptr)
		return ptr.resp, err
	}

	err := b.upstream.Execute(ptr)
	if err != nil {
		return nil, err
	}

	m := ptr.resp

	b.mu.Lock()
	_, ok = b.downstream[chatID]
	if !ok {
		stream := pool.New().Spawn(b.workers[m.Chat.Type])
		b.downstream[m.Chat.ID] = stream
		if m.Chat.Username != nil {
			b.downstream[*m.Chat.Username] = stream
		}
	}

	b.mu.Unlock()
	return m, nil
}

// Listen subscribes a listener to incoming updates channel.
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

type worker struct {
	upstream stream
	delay    time.Duration
}

func (w *worker) Execute(task *pool.Task) {
	time.Sleep(w.delay)
	ptr := task.Ptr.(*taskPtr)
	err := w.upstream.Execute(ptr)
	if err != nil && ptr.retry < 3 {
		ptr.retry += 1
		task.Retry()
	} else {
		task.Complete(err)
	}
}

type taskPtr struct {
	chatID ChatID
	entity interface{}
	opts   SendOpts
	resp   *Message
	retry  int
}
