package receiver

import (
	"context"
	"strings"

	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Chat struct {
	Sender           telegram.Sender
	ID               telegram.ChatID
	Silent           bool
	Preview          bool
	ParseMode        telegram.ParseMode
	ReplyMarkup      telegram.ReplyMarkup
	SkipOnMediaError bool
}

func (r *Chat) String() string {
	return "tgbot.chat." + r.ID.String()
}

func (r *Chat) SendText(ctx context.Context, text string) error {
	return r.sendText(ctx, text, r.Preview)
}

func (r *Chat) SendMedia(ctx context.Context, ref syncf.Future[*Media], caption string) error {
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
		logf.Get(r).Resultf(ctx, logf.Debug, logf.Warn, "send media [%s] failed: %v", err)
		if err == nil {
			return nil
		}
	} else if r.SkipOnMediaError {
		logf.Get(r).Debugf(ctx, "send media failed (skipping): %v", err)
		return nil
	}

	return r.sendText(ctx, caption, true)
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
	logf.Get(r).Resultf(ctx, logf.Debug, logf.Warn, "send text message [%s]: %v", cut(text, 50), err)
	return err
}

func cut(value string, size int) string {
	if len(value) < size {
		return value
	}

	newLine := strings.Index(value, "\n")
	if newLine >= 0 && newLine < size {
		size = newLine
	}

	return value[:size] + "..."
}

func (r *Chat) getSendOptions() *telegram.SendOptions {
	return &telegram.SendOptions{
		DisableNotification: r.Silent,
		ReplyMarkup:         r.ReplyMarkup,
	}
}
