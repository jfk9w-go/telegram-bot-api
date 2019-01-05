package telegram

import (
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/lego/pool"
)

type sendQueue = pool.Pool

const globalSendQueueDelay = 30 * time.Millisecond

var sendQueueDelays = map[ChatType]time.Duration{
	PrivateChat: 30 * time.Millisecond,
	GroupChat:   time.Second,
	Supergroup:  time.Second,
	Channel:     0,
}

type SendQueues struct {
	client *Client
	global sendQueue
	subs   map[ChatID]sendQueue
	mu     *sync.RWMutex
}

func newSendQueues(client *Client) *SendQueues {
	return &SendQueues{
		client: client,
		subs:   make(map[ChatID]pool.Pool),
		global: pool.New().SpawnFunc(func(task *pool.Task) {
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

			time.Sleep(globalSendQueueDelay)
		}),
		mu: new(sync.RWMutex),
	}
}

func (queues *SendQueues) sub(chatID ChatID) (sendQueue, error) {
	queues.mu.RLock()
	if queue, ok := queues.subs[chatID]; ok {
		queues.mu.RUnlock()
		return queue, nil
	}

	queues.mu.RUnlock()
	queues.mu.Lock()
	if queue, ok := queues.subs[chatID]; ok {
		queues.mu.Unlock()
		return queue, nil
	}

	chat, err := queues.client.GetChat(chatID)
	if err != nil {
		return nil, err
	}

	queue := pool.New().Spawn(&sub{queues.global, sendQueueDelays[chat.Type]})
	queues.subs[chat.ID] = queue
	if chat.Username != "" {
		queues.subs[chat.Username] = queue
	}

	queues.mu.Unlock()
	log.Printf("Created new send queue for %s", chatID.StringValue())

	return queue, nil
}

func (queues *SendQueues) Send(chatID ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	queue, err := queues.sub(chatID)
	if err != nil {
		return nil, err
	}

	ptr := &taskPtr{chatID: chatID, entity: entity, opts: opts}
	err = queue.Execute(ptr)
	return ptr.resp, err
}

type sub struct {
	global sendQueue
	delay  time.Duration
}

func (s *sub) Execute(task *pool.Task) {
	ptr := task.Ptr.(*taskPtr)
	err := s.global.Execute(ptr)
	if err != nil && ptr.retry < 3 {
		ptr.retry += 1
		task.Retry()
	} else {
		task.Complete(err)
	}

	time.Sleep(s.delay)
}

type taskPtr struct {
	chatID ChatID
	entity interface{}
	opts   SendOpts
	resp   *Message
	retry  int
}
