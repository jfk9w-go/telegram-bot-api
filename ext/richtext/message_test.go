package richtext_test

import (
	"testing"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/stretchr/testify/assert"
)

const LoremIpsum = "Lorem ipsum dolor sit amet, " +
	"consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
	"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo " +
	"consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat " +
	"nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt " +
	"mollit anim id est laborum."

func TestMessage_15x2(t *testing.T) {
	output := richtext.NewBufferedOutput()
	message := &richtext.Message{
		Transport: output,
		PageSize:  15,
		PageCount: 2,
	}

	_ = message.Breakable(LoremIpsum)
	_ = message.Flush()

	assert.Equal(t, []string{"Lorem ipsum", "dolor sit amet,"}, output.Pages)
}

func TestMessage_15x2_Prefix_Suffix(t *testing.T) {
	output := richtext.NewBufferedOutput()
	message := &richtext.Message{
		Transport: output,
		PageSize:  15,
		PageCount: 2,
		Prefix:    "<i>",
		Suffix:    "</i>",
	}

	_ = message.Unbreakable(message.Prefix)
	_ = message.Breakable(LoremIpsum)
	_ = message.Flush()

	assert.Equal(t, []string{"<i>Lorem</i>", "<i>ipsum</i>"}, output.Pages)
}

func TestMessage_Broken(t *testing.T) {
	output := richtext.NewBufferedOutput()
	message := &richtext.Message{
		Transport: output,
		PageSize:  15,
		PageCount: 2,
	}

	_ = message.Unbreakable(LoremIpsum)
	_ = message.Flush()

	assert.Equal(t, []string{"BROKEN"}, output.Pages)
}
