package receiver

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type receiverFunc func(ctx context.Context, receiver Interface) error

type Broadcast struct {
	Receivers []Interface
	Strict    bool
}

func (r *Broadcast) SendText(ctx context.Context, text string) error {
	return r.broadcast(ctx, "text", func(ctx context.Context, receiver Interface) error {
		return receiver.SendText(ctx, text)
	})
}

func (r *Broadcast) SendMedia(ctx context.Context, ref media.Ref, caption string) error {
	return r.broadcast(ctx, "media", func(ctx context.Context, receiver Interface) error {
		return receiver.SendMedia(ctx, ref, caption)
	})
}

func (r *Broadcast) broadcast(ctx context.Context, description string, body receiverFunc) error {
	for _, receiver := range r.Receivers {
		if err := body(ctx, receiver); err != nil {
			if r.Strict {
				return errors.Wrapf(err, "send %s to %s", description, receiver)
			}

			logrus.WithField("receiver", receiver).
				Warnf("send %s: %s", description, err)
		}
	}

	return nil
}
