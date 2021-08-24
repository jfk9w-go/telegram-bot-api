package richtext

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"golang.org/x/exp/utf8string"
)

type Message struct {
	Context   context.Context
	Transport Output
	Prefix    string
	Suffix    string
	PageSize  int
	PageCount int
	Overflow  bool
	curr      strings.Builder
	currSize  int
	currCount int
}

func (m *Message) Write(text string) {
	if m.Overflow {
		return
	}
	m.curr.WriteString(text)
	m.currSize += utf8.RuneCountInString(text)
}

func (m *Message) reset() {
	m.curr.Reset()
	m.currSize = 0
}

func (m *Message) Break() error {
	if m.Overflow {
		return nil
	}
	if m.currSize > utf8.RuneCountInString(m.Suffix) {
		m.Write(m.Suffix)
		if err := m.Transport.Text(m.Context, m.trim(m.curr.String()), false); err != nil {
			return err
		}
		m.reset()
		m.currCount++
		if m.PageCount > 0 && m.currCount >= m.PageCount {
			m.Overflow = true
		}
		if m.Overflow {
			return nil
		}
		m.Write(m.Prefix)
	}

	return nil
}

func (m *Message) Flush() error {
	if err := m.Break(); err != nil {
		return err
	}

	m.currCount = 0
	return nil
}

func (m *Message) Capacity() int {
	if m.PageSize < 1 {
		return math.MaxInt32
	}
	return m.PageSize - m.currSize - utf8.RuneCountInString(m.Suffix)
}

func (m *Message) trim(text string) string {
	return strings.Trim(text, " \t\n\v")
}

func (m *Message) Breakable(text string) error {
	if m.Overflow {
		return nil
	}
	utext := utf8string.NewString(text)
	length := utext.RuneCount()
	offset := 0
	capacity := m.Capacity()
	end := offset + capacity
	for end < length {
		nextOffset := end
	search:
		for i := end; i >= 0; i-- {
			switch utext.At(i) {
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

		m.Write(m.trim(utext.Slice(offset, end)))
		if err := m.Break(); err != nil {
			return err
		}
		if m.Overflow {
			return nil
		}
		offset = nextOffset
		capacity = m.Capacity()
		end = offset + capacity
	}

	m.Write(utext.Slice(offset, length))
	return nil
}

func (m *Message) Unbreakable(text string) error {
	if m.Overflow {
		return nil
	}
	length := utf8.RuneCountInString(text)
	if length > m.Capacity() {
		if err := m.Break(); err != nil {
			return err
		}
		if length > m.Capacity() {
			return m.Breakable("BROKEN")
		}
	}

	m.Write(text)
	return nil
}

func (m *Message) Media(ref media.Ref, anchor string, collapsible bool) error {
	if m.Overflow {
		return nil
	}
	caption := anchor
	if collapsible && m.currCount == 0 && m.currSize+utf8.RuneCountInString(anchor)+1 <= telegram.MaxCaptionSize {
		if m.currSize > utf8.RuneCountInString(m.Suffix) {
			caption += "\n" + m.curr.String()
			m.reset()
		}
	} else {
		if err := m.Flush(); err != nil {
			return err
		}
	}

	media, mediaErr := ref.Get(m.Context)
	return m.Transport.Media(m.Context, media, mediaErr, caption)
}
