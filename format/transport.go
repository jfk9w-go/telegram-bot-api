package format

import (
	"context"
	"log"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Transport interface {
	Text(ctx context.Context, text string, preview bool) error
	Media(ctx context.Context, media *Media, mediaErr error, text string) error
}

type TelegramTransport struct {
	Sender    telegram.Sender
	ChatIDs   []telegram.ChatID
	ParseMode telegram.ParseMode
	Strict    bool
	Notify    bool
}

func (t *TelegramTransport) send(ctx context.Context, chatIDs []telegram.ChatID, sendable telegram.Sendable) error {
	options := &telegram.SendOptions{DisableNotification: !t.Notify}
	for _, chatID := range chatIDs {
		if _, err := t.Sender.Send(ctx, chatID, sendable, options); err != nil {
			if t.Strict {
				return errors.Wrapf(err, "send text to %s", chatID)
			} else {
				log.Printf("Failed to send entity (below) to %s: %s\n%s", chatID, err, sendable)
			}
		}
	}

	return nil
}

func (t *TelegramTransport) Text(ctx context.Context, text string, preview bool) error {
	return t.send(ctx, t.ChatIDs, telegram.Text{
		Text:                  text,
		ParseMode:             t.ParseMode,
		DisableWebPagePreview: !preview,
	})
}

func (t *TelegramTransport) Media(ctx context.Context, media *Media, mediaErr error, caption string) error {
	if mediaErr == nil {
		chatID := t.ChatIDs[0]
		if message, err := t.Sender.Send(ctx, chatID,
			telegram.Media{
				ParseMode: t.ParseMode,
				Caption:   caption,
				Input:     media.Input,
				Type:      telegram.MediaTypeByMIMEType(media.MIMEType)},
			&telegram.SendOptions{DisableNotification: !t.Notify}); err != nil {
			mediaErr = err
		} else {
			var file *telegram.MessageFile
			switch {
			case len(message.Photo) > 0:
				file = &message.Photo[0]
			case message.Video != nil:
				file = message.Video
			case message.Animation != nil:
				file = message.Animation
			}

			if file == nil {
				mediaErr = errors.New("unsupported media type")
			} else {
				return t.send(ctx, t.ChatIDs[1:], telegram.Media{
					ParseMode: t.ParseMode,
					Caption:   caption,
					Input:     flu.URL(file.ID),
					Type:      telegram.MediaTypeByMIMEType(media.MIMEType)})
			}
		}
	}

	log.Printf("Failed to send initial HTML media message (below) to %s: %s\n%s", t.ChatIDs[0], mediaErr, caption)
	return t.Text(ctx, caption, true)
}
