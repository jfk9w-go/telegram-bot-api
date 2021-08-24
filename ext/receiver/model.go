package receiver

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
)

var ErrUnsupportedMediaType = errors.New("unsupported media type")

type Interface interface {
	SendText(ctx context.Context, text string) error
	SendMedia(ctx context.Context, ref media.Ref, caption string) error
}
