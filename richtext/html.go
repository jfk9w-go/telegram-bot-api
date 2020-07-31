package richtext

import (
	"math"
	"strings"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"golang.org/x/exp/utf8string"
	"golang.org/x/net/html"
)

type HTMLPrinter struct {
	builder     strings.Builder
	tag         *htmlTag
	tags        HTMLTags
	link        *Link
	linkPrinter LinkPrinter
	pageSize    int
	maxPageSize int
	pages       []string
	maxPages    int
}

func HTML(maxPageSize, maxPages int, tags HTMLTags, linkPrinter LinkPrinter) *HTMLPrinter {
	if tags == nil {
		tags = DefaultSupportedTags
	}
	if linkPrinter == nil {
		linkPrinter = DefaultLinkPrinter
	}
	return &HTMLPrinter{
		builder:     strings.Builder{},
		tags:        tags,
		linkPrinter: linkPrinter,
		maxPageSize: maxPageSize,
		pages:       make([]string, 0),
		maxPages:    maxPages,
	}
}

func (p *HTMLPrinter) isOutOfBounds() bool {
	return p.maxPages >= 1 && len(p.pages) > p.maxPages
}

func (p *HTMLPrinter) write(text string) {
	p.builder.WriteString(text)
	p.pageSize += utf8.RuneCountInString(text)
}

func (p *HTMLPrinter) writeTagStart() bool {
	if p.tag != nil {
		return p.writeUnbreakable("<" + p.tag.name + ">")
	}
	return true
}

func (p *HTMLPrinter) writeTagEnd() {
	if p.tag != nil {
		p.write("</" + p.tag.name + ">")
	}
}

func (p *HTMLPrinter) breakPage() bool {
	if p.isOutOfBounds() {
		return false
	}
	if p.pageSize > p.tag.startLen() {
		p.write(p.tag.end())
		p.pages = append(p.pages, p.builder.String())
		p.builder.Reset()
		p.pageSize = 0
		if p.isOutOfBounds() {
			return false
		}
		p.write(p.tag.start())
	}
	return true
}

func (p *HTMLPrinter) capacity() int {
	if p.maxPageSize < 1 {
		return math.MaxInt32
	}
	capacity := p.maxPageSize - p.pageSize
	if p.tag != nil {
		capacity -= p.tag.endLen()
	}
	return capacity
}

func (p *HTMLPrinter) writeBreakable(text string) bool {
	if p.isOutOfBounds() {
		return false
	}
	utf8Text := utf8string.NewString(text)
	length := utf8Text.RuneCount()
	offset := 0
	capacity := p.capacity()
	end := offset + capacity
	for end < length {
		nextOffset := end
	search:
		for i := end; i >= 0; i-- {
			switch utf8Text.At(i) {
			case '\n', ' ', '\t', '\v':
				end, nextOffset = i, i+1
				break search
			case ',', '.', ':', ';':
				end, nextOffset = i+1, i+1
				break search
			default:
				continue
			}
		}
		p.write(trim(utf8Text, offset, end))
		if !p.breakPage() {
			return false
		}
		offset = nextOffset
		capacity = p.capacity()
		end = offset + capacity
	}
	p.write(utf8Text.Slice(offset, length))
	return true
}

func trim(str *utf8string.String, start, end int) string {
	return strings.Trim(str.Slice(start, end), " \t\n\v")
}

func (p *HTMLPrinter) writeUnbreakable(text string) bool {
	if p.isOutOfBounds() {
		return false
	}
	length := utf8.RuneCountInString(text)
	if length > p.capacity() {
		if !p.breakPage() {
			return false
		}
		if length > p.capacity() {
			return p.writeBreakable("BROKEN")
		} else {
			p.write(text)
		}
	} else {
		p.write(text)
	}
	return true
}

func (p *HTMLPrinter) Flush() Message {
	message := Message{Format: telegram.HTML}
	if p.breakPage() {
		message.Text = p.pages
	} else {
		if len(p.pages) > 0 {
			message.Text = p.pages[:len(p.pages)-1]
		} else {
			message.Text = p.pages
		}
	}

	return message
}

func (p *HTMLPrinter) StartTag(name string, attrs []html.Attribute) *HTMLPrinter {
	if p.isOutOfBounds() {
		return p
	}
	if name == "br" {
		return p.NewLine()
	}
	if name == "a" {
		if p.link == nil {
			p.link = &Link{Attrs: attrs, tag: p.tag}
			if p.tag != nil {
				p.writeTagEnd()
				p.tag = nil
			}
		}
		return p
	}
	if p.tag != nil {
		p.tag.depth++
		return p
	}
	if name, ok := p.tags.Get(name, attrs); ok {
		p.tag = &htmlTag{name, 0}
		p.writeTagStart()
	}
	return p
}

func (p *HTMLPrinter) EndTag() *HTMLPrinter {
	if p.isOutOfBounds() {
		return p
	}
	if p.link != nil {
		link := p.link
		p.link = nil
		p.writeUnbreakable(p.linkPrinter.Print(link))
		if link.tag != nil {
			p.tag = link.tag
			p.writeTagStart()
		}
		return p
	}
	if p.tag != nil {
		p.tag.depth--
		if p.tag.depth <= 0 {
			p.write(p.tag.end())
			p.tag = nil
		}
	}
	return p
}

func (p *HTMLPrinter) Tag(name string) *HTMLPrinter {
	p.StartTag(name, nil)
	return p
}

func (p *HTMLPrinter) Text(text string) *HTMLPrinter {
	text = html.EscapeString(text)
	if p.link != nil {
		p.link.Text = text
	} else {
		p.writeBreakable(text)
	}
	return p
}

func (p *HTMLPrinter) NewLine() *HTMLPrinter {
	if p.capacity() >= 5 {
		p.write("\n")
	} else {
		p.breakPage()
	}
	return p
}

func (p *HTMLPrinter) Link(text, href string) *HTMLPrinter {
	return p.StartTag("a", []html.Attribute{{Key: "href", Val: href}}).
		Text(text).
		EndTag()
}

func (p *HTMLPrinter) Parse(raw string) *HTMLPrinter {
	reader := strings.NewReader(raw)
	tokenizer := html.NewTokenizer(reader)
	for {
		if p.isOutOfBounds() {
			return p
		}
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		data := token.Data
		switch token.Type {
		case html.TextToken:
			p.Text(data)
		case html.StartTagToken:
			p.StartTag(data, token.Attr)
		case html.EndTagToken:
			p.EndTag()
		}
	}
	return p
}

func (p *HTMLPrinter) ParseMode() telegram.ParseMode {
	return telegram.HTML
}
