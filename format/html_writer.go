package format

import (
	"context"
	"fmt"
	"io"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"golang.org/x/net/html"
)

type HTMLTag struct {
	Open, Close string
	parent      *HTMLTag
}

type htmlAnchor struct {
	text   string
	attrs  HTMLAttributes
	parent *HTMLTag
}

var (
	Bold   = HTMLTag{Open: "<b>", Close: "</b>"}
	Italic = HTMLTag{Open: "<i>", Close: "</i>"}
	Code   = HTMLTag{Open: "<code>", Close: "</code>"}
	Pre    = HTMLTag{Open: "<pre>", Close: "</pre>"}
)

type HTMLAttributes []html.Attribute

func (attrs HTMLAttributes) Get(key string) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

type HTMLTagConverter interface {
	Get(tag string, attrs HTMLAttributes) (HTMLTag, bool)
}

type PlainHTMLTagConverter map[string]HTMLTag

func (c PlainHTMLTagConverter) Get(name string, _ HTMLAttributes) (HTMLTag, bool) {
	tag, ok := c[name]
	return tag, ok
}

var DefaultHTMLTagConverter = PlainHTMLTagConverter{
	"strong": Bold,
	"b":      Bold,
	"italic": Italic,
	"em":     Italic,
	"i":      Italic,
	"code":   Code,
	"pre":    Pre,
}

type HTMLAnchorFormat interface {
	Format(text string, attrs HTMLAttributes) string
}

type defaultHTMLAnchorFormat struct{}

func (f defaultHTMLAnchorFormat) Format(text string, attrs HTMLAttributes) string {
	href := attrs.Get("href")
	if href != "" {
		return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(href), text)
	}
	return ""
}

var DefaultHTMLAnchorFormat HTMLAnchorFormat = defaultHTMLAnchorFormat{}

func HTMLAnchor(text, href string) string {
	return DefaultHTMLAnchorFormat.Format(html.EscapeString(text), HTMLAttributes{{Key: "href", Val: href}})
}

type HTMLWriter struct {
	Session      *Session
	TagConverter HTMLTagConverter
	AnchorFormat HTMLAnchorFormat
	currTag      *HTMLTag
	currAnchor   *htmlAnchor
	err          error
}

var (
	DefaultLinkAllocSize  = 200
	DefaultMaxMessageSize = telegram.MaxMessageSize - DefaultLinkAllocSize
	DefaultMaxCaptionSize = telegram.MaxCaptionSize - DefaultLinkAllocSize
)

func HTMLWithTransport(ctx context.Context, transport Transport) *HTMLWriter {
	return &HTMLWriter{
		Session: &Session{
			Context:   ctx,
			Transport: transport,
			PageSize:  DefaultMaxMessageSize,
		},
		TagConverter: DefaultHTMLTagConverter,
		AnchorFormat: DefaultHTMLAnchorFormat,
	}
}

func HTML(ctx context.Context, sender telegram.Sender, notify bool, chatIDs ...telegram.ChatID) *HTMLWriter {
	if notify {
		ctx = WithNotify(ctx)
	}

	return HTMLWithTransport(
		WithParseMode(ctx, telegram.HTML),
		&TelegramTransport{
			Sender:  sender,
			ChatIDs: chatIDs,
			Strict:  true,
		})
}

func (w *HTMLWriter) StartTag(name string, attrs HTMLAttributes) *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return w
	}
	switch name {
	case "br":
		w.err = w.Session.Breakable("\n")
	case "a":
		w.currAnchor = &htmlAnchor{
			attrs:  attrs,
			parent: w.currTag,
		}
	default:
		if tag, ok := w.TagConverter.Get(name, attrs); ok {
			if len(tag.Open)+len(tag.Close)+3 >= w.Session.Capacity() {
				if err := w.Session.Break(); err != nil {
					w.err = err
					return w
				}
			}

			if w.currAnchor != nil {
				w.currAnchor.text += tag.Open
			} else {
				w.Session.Write(tag.Open)
				w.Session.Prefix += tag.Open
				w.Session.Suffix = tag.Close + w.Session.Suffix
			}

			tag.parent = w.currTag
			w.currTag = &tag
			return w
		} else {
			w.currTag = &HTMLTag{parent: w.currTag}
		}
	}

	return w
}

func (w *HTMLWriter) Text(text string) *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return w
	}
	text = html.EscapeString(text)
	if w.currAnchor != nil {
		w.currAnchor.text += html.EscapeString(text)
	} else {
		w.err = w.Session.Breakable(text)
	}

	return w
}

func (w *HTMLWriter) EndTag() *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return w
	}
	switch {
	case w.currAnchor != nil && w.currAnchor.parent == w.currTag:
		str := w.AnchorFormat.Format(w.currAnchor.text, w.currAnchor.attrs)
		if err := w.Session.Unbreakable(str); err != nil {
			w.err = err
		} else {
			w.currAnchor = nil
		}

	case w.currTag != nil:
		if w.currAnchor != nil {
			w.currAnchor.text += w.currTag.Close
		} else {
			w.Session.Write(w.currTag.Close)
			w.Session.Prefix = w.Session.Prefix[:len(w.Session.Prefix)-len(w.currTag.Open)]
			w.Session.Suffix = w.Session.Suffix[len(w.currTag.Close):]
		}

		w.currTag = w.currTag.parent
	}

	return w
}

func (w *HTMLWriter) Bold(text string) *HTMLWriter {
	return w.StartTag("b", nil).Text(text).EndTag()
}

func (w *HTMLWriter) Italic(text string) *HTMLWriter {
	return w.StartTag("i", nil).Text(text).EndTag()
}

func (w *HTMLWriter) Code(text string) *HTMLWriter {
	return w.StartTag("code", nil).Text(text).EndTag()
}

func (w *HTMLWriter) Pre(text string) *HTMLWriter {
	return w.StartTag("pre", nil).Text(text).EndTag()
}

func (w *HTMLWriter) Link(text, href string) *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return w
	}
	w.err = w.Session.Unbreakable(HTMLAnchor(text, href))
	return w
}

func (w *HTMLWriter) Markup(reader io.Reader) *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return w
	}
	tokenizer := html.NewTokenizer(reader)
	for {
		if w.err != nil || w.Session.Overflow {
			return w
		}
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return w
		}
		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			w.StartTag(token.Data, token.Attr)
		case html.TextToken:
			w.Text(token.Data)
		case html.EndTagToken:
			w.EndTag()
		}
	}
}

func (w *HTMLWriter) MarkupString(markup string) *HTMLWriter {
	return w.Markup(strings.NewReader(markup))
}

func (w *HTMLWriter) Media(url string, ref MediaRef, collapsible bool) *HTMLWriter {
	if w.err != nil || w.Session.Overflow {
		return nil
	}
	w.err = w.Session.Media(ref, HTMLAnchor("[media]", url), collapsible)
	return w
}

func (w *HTMLWriter) Flush() error {
	if w.err != nil || w.Session.Overflow {
		return w.err
	}
	for w.currAnchor != nil || w.currTag != nil {
		w.EndTag()
	}

	return w.Session.Flush()
}
