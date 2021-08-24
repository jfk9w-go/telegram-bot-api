package html

import (
	"context"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"golang.org/x/net/html"
)

var (
	DefaultLinkAllocSize  = 200
	DefaultMaxMessageSize = telegram.MaxMessageSize - DefaultLinkAllocSize
	DefaultMaxCaptionSize = telegram.MaxCaptionSize - DefaultLinkAllocSize
)

type AnchorFormat interface {
	Format(text string, attrs []html.Attribute) string
}

type Tag struct {
	Open, Close string
	parent      *Tag
}

var (
	Bold   = Tag{Open: "<b>", Close: "</b>"}
	Italic = Tag{Open: "<i>", Close: "</i>"}
	Code   = Tag{Open: "<code>", Close: "</code>"}
	Pre    = Tag{Open: "<pre>", Close: "</pre>"}
)

type TagConverter interface {
	Get(tag string, attrs []html.Attribute) (Tag, bool)
}

type Output interface {
	IsOverflown() bool
	UpdatePrefix(update func(prefix string) string)
	UpdateSuffix(update func(suffix string) string)
	Write(text string)
	WriteBreakable(ctx context.Context, text string) error
	WriteUnbreakable(ctx context.Context, text string) error
	AddMedia(ctx context.Context, ref media.Ref, anchor string, collapsible bool) error
	BreakPage(ctx context.Context) error
	Flush(ctx context.Context) error
	PageCapacity() int
}
