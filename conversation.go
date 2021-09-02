package telegram

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Question chan *Message

type Sender interface {
	Send(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error)
}

type ConversationAware struct {
	sender    Sender
	questions map[ID]Question
	mu        flu.RWMutex
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

func (c *ConversationAware) Answer(ctx context.Context, message *Message) error {
	if message.ReplyToMessage == nil {
		return errors.New("not a question")
	}

	defer c.mu.RLock().Unlock()
	question, ok := c.questions[message.ReplyToMessage.ID]
	if !ok {
		return errors.New("forgotten")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case question <- message:
		return nil
	}
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
