package format

import (
	"context"
	"log"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

var ErrSkipMedia = errors.New("skip")

type Transport interface {
	Text(ctx context.Context, text string, preview bool) error
	Media(ctx context.Context, media Media, mediaErr error, text string) error
}

const (
	parseModeKey   = "parse_mode"
	notifyKey      = "notify"
	replyMarkupKey = "reply_markup"
)

func WithParseMode(ctx context.Context, parseMode telegram.ParseMode) context.Context {
	return context.WithValue(ctx, parseModeKey, parseMode)
}

func getParseMode(ctx context.Context) telegram.ParseMode {
	value := ctx.Value(parseModeKey)
	if value != nil {
		return value.(telegram.ParseMode)
	} else {
		return telegram.None
	}
}

func WithNotify(ctx context.Context) context.Context {
	return context.WithValue(ctx, notifyKey, true)
}

func getNotify(ctx context.Context) bool {
	return ctx.Value(notifyKey) != nil
}

func WithReplyMarkup(ctx context.Context, markup telegram.ReplyMarkup) context.Context {
	return context.WithValue(ctx, replyMarkupKey, markup)
}

func getReplyMarkup(ctx context.Context) telegram.ReplyMarkup {
	value := ctx.Value(replyMarkupKey)
	if value != nil {
		return value.(telegram.ReplyMarkup)
	} else {
		return nil
	}
}

type TelegramTransport struct {
	Sender  telegram.Sender
	ChatIDs []telegram.ChatID
	Strict  bool
}

func (t *TelegramTransport) send(ctx context.Context, chatIDs []telegram.ChatID, sendable telegram.Sendable) error {
	options := &telegram.SendOptions{
		DisableNotification: !getNotify(ctx),
		ReplyMarkup:         getReplyMarkup(ctx)}
	for _, chatID := range chatIDs {
		if _, err := t.Sender.Send(ctx, chatID, sendable, options); err != nil {
			if t.Strict {
				return errors.Wrapf(err, "send text to %s", chatID)
			} else {
				log.Printf("[chat > %s] unable to send message due to %s:\n%s", chatID, err, sendable)
			}
		}
	}

	return nil
}

func (t *TelegramTransport) Text(ctx context.Context, text string, preview bool) error {
	if text == "" {
		return nil
	}

	return t.send(ctx, t.ChatIDs, telegram.Text{
		Text:                  text,
		ParseMode:             getParseMode(ctx),
		DisableWebPagePreview: !preview,
	})
}

func (t *TelegramTransport) Media(ctx context.Context, media Media, mediaErr error, caption string) error {
	if errors.Is(mediaErr, ErrSkipMedia) {
		log.Printf("[chat > %+v] skipping media: %s", t.ChatIDs, mediaErr)
		return nil
	}

	if mediaErr == nil {
		chatID := t.ChatIDs[0]
		if message, err := t.Sender.Send(ctx, chatID,
			telegram.Media{
				ParseMode: getParseMode(ctx),
				Caption:   caption,
				Input:     media.Input,
				Type:      telegram.MediaTypeByMIMEType(media.MIMEType)},
			&telegram.SendOptions{
				DisableNotification: !getNotify(ctx),
				ReplyMarkup:         getReplyMarkup(ctx),
			}); err != nil {
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
					ParseMode: getParseMode(ctx),
					Caption:   caption,
					Input:     flu.URL(file.ID),
					Type:      telegram.MediaTypeByMIMEType(media.MIMEType)})
			}
		}
	}

	log.Printf("[chat > %+v] unable to send media: %s\n%s", t.ChatIDs, mediaErr, caption)
	return t.Text(ctx, caption, true)
}

type BufferTransport struct {
	Pages []string
}

func NewBufferTransport() *BufferTransport {
	return &BufferTransport{
		Pages: make([]string, 0),
	}
}

func (b *BufferTransport) Text(ctx context.Context, text string, preview bool) error {
	b.Pages = append(b.Pages, text)
	return nil
}

func (b *BufferTransport) Media(ctx context.Context, media Media, mediaErr error, text string) error {
	if text != "" {
		return b.Text(ctx, text, false)
	}

	return nil
}
