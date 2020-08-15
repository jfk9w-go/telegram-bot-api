package format

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"golang.org/x/exp/utf8string"
)

type Session struct {
	Context   context.Context
	Transport Transport
	Prefix    string
	Suffix    string
	PageSize  int
	PageCount int
	Overflow  bool
	curr      strings.Builder
	currSize  int
	currCount int
}

func (s *Session) Write(text string) {
	if s.Overflow {
		return
	}
	s.curr.WriteString(text)
	s.currSize += utf8.RuneCountInString(text)
}

func (s *Session) reset() {
	s.curr.Reset()
	s.currSize = 0
}

func (s *Session) Break() error {
	if s.Overflow {
		return nil
	}
	if s.currSize > utf8.RuneCountInString(s.Suffix) {
		s.Write(s.Suffix)
		if err := s.Transport.Text(s.Context, s.trim(s.curr.String()), false); err != nil {
			return err
		}
		s.reset()
		s.currCount++
		if s.PageCount > 0 && s.currCount >= s.PageCount {
			s.Overflow = true
		}
		if s.Overflow {
			return nil
		}
		s.Write(s.Prefix)
	}

	return nil
}

func (s *Session) Flush() error {
	return s.Break()
}

func (s *Session) Capacity() int {
	if s.PageSize < 1 {
		return math.MaxInt64
	}
	return s.PageSize - s.currSize - utf8.RuneCountInString(s.Suffix)
}

func (s *Session) trim(text string) string {
	return strings.Trim(text, " \t\n\v")
}

func (s *Session) Breakable(text string) error {
	if s.Overflow {
		return nil
	}
	utext := utf8string.NewString(text)
	length := utext.RuneCount()
	offset := 0
	capacity := s.Capacity()
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

		s.Write(s.trim(utext.Slice(offset, end)))
		if err := s.Break(); err != nil {
			return err
		}
		if s.Overflow {
			return nil
		}
		offset = nextOffset
		capacity = s.Capacity()
		end = offset + capacity
	}

	s.Write(utext.Slice(offset, length))
	return nil
}

func (s *Session) Unbreakable(text string) error {
	if s.Overflow {
		return nil
	}
	length := utf8.RuneCountInString(text)
	if length > s.Capacity() {
		if err := s.Break(); err != nil {
			return err
		}
		if length > s.Capacity() {
			return s.Breakable("BROKEN")
		}
	}

	s.Write(text)
	return nil
}

func (s *Session) Media(ref MediaRef, anchor string, collapsible bool) error {
	if s.Overflow {
		return nil
	}
	caption := anchor
	if collapsible && s.currCount == 0 && s.currSize+utf8.RuneCountInString(anchor)+1 <= telegram.MaxCaptionSize {
		if s.currSize > utf8.RuneCountInString(s.Suffix) {
			caption += "\n" + s.curr.String()
			s.reset()
		}
	} else {
		if err := s.Flush(); err != nil {
			return err
		}
	}

	media, mediaErr := ref.Get(s.Context)
	return s.Transport.Media(s.Context, media, mediaErr, caption)
}
