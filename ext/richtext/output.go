package richtext

import (
	"context"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var ErrSkipMedia = errors.New("skip")

type Output interface {
	Text(ctx context.Context, text string, preview bool) error
	Media(ctx context.Context, media *Media, mediaErr error, text string) error
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

type TelegramOutput struct {
	Sender  telegram.Sender
	ChatIDs []telegram.ChatID
	Strict  bool
}

func (o *TelegramOutput) send(ctx context.Context, chatIDs []telegram.ChatID, sendable telegram.Sendable) error {
	options := &telegram.SendOptions{
		DisableNotification: !getNotify(ctx),
		ReplyMarkup:         getReplyMarkup(ctx)}
	for _, chatID := range chatIDs {
		if _, err := o.Sender.Send(ctx, chatID, sendable, options); err != nil {
			if o.Strict {
				return errors.Wrapf(err, "send text to %s", chatID)
			} else {
				logrus.WithField("chat", chatID).
					Warnf("failed to send message: %s", err)
			}
		}
	}

	return nil
}

func (o *TelegramOutput) Text(ctx context.Context, text string, preview bool) error {
	if text == "" {
		return nil
	}

	return o.send(ctx, o.ChatIDs, telegram.Text{
		Text:                  text,
		ParseMode:             getParseMode(ctx),
		DisableWebPagePreview: !preview,
	})
}

func (o *TelegramOutput) Media(ctx context.Context, media *Media, mediaErr error, caption string) error {
	if errors.Is(mediaErr, ErrSkipMedia) {
		return nil
	}

	if mediaErr == nil {
		chatID := o.ChatIDs[0]
		if message, err := o.Sender.Send(ctx, chatID,
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
				return o.send(ctx, o.ChatIDs[1:], telegram.Media{
					ParseMode: getParseMode(ctx),
					Caption:   caption,
					Input:     flu.URL(file.ID),
					Type:      telegram.MediaTypeByMIMEType(media.MIMEType)})
			}
		}
	}

	logrus.WithField("chats", o.ChatIDs).
		Warnf("failed to send media: %s", mediaErr)
	return o.Text(ctx, caption, true)
}

type BufferedOutput struct {
	Pages []string
}

func NewBufferedOutput() *BufferedOutput {
	return &BufferedOutput{
		Pages: make([]string, 0),
	}
}

func (o *BufferedOutput) Text(ctx context.Context, text string, preview bool) error {
	o.Pages = append(o.Pages, text)
	return nil
}

func (o *BufferedOutput) Media(ctx context.Context, media *Media, mediaErr error, text string) error {
	if text != "" {
		return o.Text(ctx, text, false)
	}

	return nil
}
