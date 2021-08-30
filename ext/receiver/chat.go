package receiver

import (
	"context"
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	Sender      telegram.Sender
	ID          telegram.ChatID
	Silent      bool
	Preview     bool
	ParseMode   telegram.ParseMode
	ReplyMarkup telegram.ReplyMarkup
}

func (r *Chat) SendText(ctx context.Context, text string) error {
	return r.sendText(ctx, text, r.Preview)
}

func (r *Chat) SendMedia(ctx context.Context, ref media.Ref, caption string) error {
	media, err := ref.Get(ctx)
	if err == nil {
		if media == nil {
			return nil
		}

		payload := &telegram.Media{
			Type:      telegram.MediaTypeByMIMEType(media.MIMEType),
			Input:     media.Input,
			Caption:   caption,
			ParseMode: r.ParseMode,
		}

		_, err = r.Sender.Send(ctx, r.ID, payload, r.getSendOptions())
		if err == nil {
			return nil
		}

		logrus.WithField("chat", r.ID).Warnf("send media: %s", err)
	}

	return r.sendText(ctx, caption, true)
}

func (r *Chat) String() string {
	return fmt.Sprint(r.ID)
}

func (r *Chat) sendText(ctx context.Context, text string, preview bool) error {
	if text == "" {
		return nil
	}

	payload := &telegram.Text{
		Text:                  text,
		ParseMode:             r.ParseMode,
		DisableWebPagePreview: !preview,
	}

	_, err := r.Sender.Send(ctx, r.ID, payload, r.getSendOptions())
	return err
}

func (r *Chat) getSendOptions() *telegram.SendOptions {
	return &telegram.SendOptions{
		DisableNotification: r.Silent,
		ReplyMarkup:         r.ReplyMarkup,
	}
}
