package telegram

import (
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/lego"
)

type sendQueue = lego.Pool

const globalSendQueueDelay = 30 * time.Millisecond

var sendQueueDelays = map[ChatType]time.Duration{
	PrivateChat: 30 * time.Millisecond,
	GroupChat:   time.Second,
	Supergroup:  time.Second,
	Channel:     0,
}

type sendQueues struct {
	api    BotApi
	global sendQueue
	subs   map[ChatID]sendQueue
	mu     sync.RWMutex
}

func newSendQueues(api BotApi) *sendQueues {
	return &sendQueues{
		api:  api,
		subs: make(map[ChatID]lego.Pool),
		global: lego.NewPool().With(func(task lego.Task) {
			var err = task.Run.(func() error)()
			if err != nil {
				if err, ok := err.(TooManyMessages); ok {
					task.Retry()
					time.Sleep(err.RetryAfter)
					return
				}
			}

			task.Complete(err)
			time.Sleep(globalSendQueueDelay)
		}),
	}
}

func (queues *sendQueues) sub(chatId ChatID) (queue sendQueue, err error) {
	var ok bool
	queues.mu.RLock()
	if queue, ok = queues.subs[chatId]; ok {
		queues.mu.RUnlock()
		return
	}

	queues.mu.RUnlock()
	queues.mu.Lock()
	if queue, ok = queues.subs[chatId]; ok {
		queues.mu.Unlock()
		return
	}

	var chat *Chat
	chat, err = queues.api.GetChat(chatId)
	if err != nil {
		return
	}

	var delay = sendQueueDelays[chat.Type]
	queue = lego.NewPool().With(func(task lego.Task) {
		var err = task.Run.(func() error)()
		if err != nil && task.Retries < 3 {
			task.Retry()
		} else {
			task.Complete(err)
		}

		time.Sleep(delay)
	})

	queues.subs[chat.ID] = queue
	if chat.Username != "" {
		queues.subs[chat.Username] = queue
	}

	queues.mu.Unlock()
	log.Printf("Created new send queue for %s", chatId.StringValue())

	return
}

func (queues *sendQueues) send(chatId ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	var queue, err = queues.sub(chatId)
	if err != nil {
		return nil, err
	}

	var r *Message
	return r, queue.Process(func() error {
		return queues.global.Process(func() error {
			var _r, err = queues.api.Send(chatId, entity, opts)
			if err == nil {
				r = _r
			}

			return err
		})
	})
}
