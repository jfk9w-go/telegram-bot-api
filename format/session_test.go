package format_test

import (
	"context"
	"testing"

	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/stretchr/testify/assert"
)

const LoremIpsum = "Lorem ipsum dolor sit amet, " +
	"consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
	"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo " +
	"consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat " +
	"nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt " +
	"mollit anim id est laborum."

type testTransport struct {
	pages []string
}

func newTestTransport() *testTransport {
	return &testTransport{pages: make([]string, 0)}
}

func (t *testTransport) Text(ctx context.Context, text string, preview bool) error {
	t.pages = append(t.pages, text)
	return nil
}

func (t *testTransport) Media(ctx context.Context, media *format.Media, mediaErr error, text string) error {
	panic("implement me")
}

func TestPageOutput_15x2(t *testing.T) {
	transport := newTestTransport()
	session := &format.Session{
		Transport: transport,
		PageSize:  15,
		PageCount: 2,
	}

	_ = session.Breakable(LoremIpsum)
	_ = session.Flush()

	assert.Equal(t, []string{"Lorem ipsum", "dolor sit amet,"}, transport.pages)
}

func TestPageOutput_15x2_Prefix_Suffix(t *testing.T) {
	transport := newTestTransport()
	session := &format.Session{
		Transport: transport,
		PageSize:  15,
		PageCount: 2,
		Prefix:    "<i>",
		Suffix:    "</i>",
	}

	_ = session.Unbreakable(session.Prefix)
	_ = session.Breakable(LoremIpsum)
	_ = session.Flush()

	assert.Equal(t, []string{"<i>Lorem</i>", "<i>ipsum</i>"}, transport.pages)
}

func TestPageOutput_Broken(t *testing.T) {
	transport := newTestTransport()
	session := &format.Session{
		Transport: transport,
		PageSize:  15,
		PageCount: 2,
	}

	_ = session.Unbreakable(LoremIpsum)
	_ = session.Flush()

	assert.Equal(t, []string{"BROKEN"}, transport.pages)
}
