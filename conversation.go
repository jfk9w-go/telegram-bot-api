package telegram

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

type Question chan *Message

type Client = *conversationAwareClient

type conversationAwareClient struct {
	*floodControlAwareClient
	questions map[ID]Question
	mutex     sync.RWMutex
}

func newConversationAwareClient(fca *floodControlAwareClient) Client {
	return &conversationAwareClient{
		floodControlAwareClient: fca,
		questions:               make(map[ID]Question),
	}
}

func (c *conversationAwareClient) Ask(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error) {
	if options == nil {
		options = new(SendOptions)
	}

	options.ReplyMarkup = ForceReply{ForceReply: true, Selective: true}
	m, err := c.Send(ctx, chatID, sendable, options)
	if err != nil {
		return nil, errors.Wrap(err, "send question")
	}

	question := c.addQuestion(m.ID)
	defer c.removeQuestion(m.ID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case answer := <-question:
		return answer, nil
	}
}

func (c *conversationAwareClient) Answer(ctx context.Context, message *Message) error {
	if message.ReplyToMessage != nil {
		c.mutex.RLock()
		question, ok := c.questions[message.ReplyToMessage.ID]
		c.mutex.RUnlock()
		if ok {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case question <- message:
				return nil
			}
		}
	}

	return nil
}

func (c *conversationAwareClient) addQuestion(id ID) Question {
	question := make(Question)
	c.mutex.Lock()
	c.questions[id] = question
	c.mutex.Unlock()
	return question
}

func (c *conversationAwareClient) removeQuestion(id ID) {
	c.mutex.Lock()
	delete(c.questions, id)
	c.mutex.Unlock()
}
