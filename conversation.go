package telegram

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

type Question chan *Message

type Sender interface {
	Send(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error)
}

type ConversationAware struct {
	sender    Sender
	questions map[ID]Question
	mu        sync.RWMutex
}

func Conversations(sender Sender) *ConversationAware {
	return &ConversationAware{
		sender:    sender,
		questions: make(map[ID]Question),
	}
}

func (c *ConversationAware) Ask(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error) {
	if options == nil {
		options = new(SendOptions)
	}

	options.ReplyMarkup = ForceReply{ForceReply: true, Selective: true}
	m, err := c.sender.Send(ctx, chatID, sendable, options)
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

func (c *ConversationAware) Answer(message *Message) bool {
	if message.ReplyToMessage != nil {
		c.mu.RLock()
		question, ok := c.questions[message.ReplyToMessage.ID]
		c.mu.RUnlock()
		if ok {
			question <- message
			return true
		}
	}

	return false
}

func (c *ConversationAware) addQuestion(id ID) Question {
	question := make(Question)
	c.mu.Lock()
	c.questions[id] = question
	c.mu.Unlock()
	return question
}

func (c *ConversationAware) removeQuestion(id ID) {
	c.mu.Lock()
	delete(c.questions, id)
	c.mu.Unlock()
}
