package telegram

import (
	"fmt"
	"os"
	"testing"
)

func TestBot(t *testing.T) {
	var (
		token  = os.Getenv("TOKEN")
		chatId = MustParseID(os.Getenv("CHAT"))
		bot    = NewBot(nil, token)
	)

	for i := 0; i < 20; i++ {
		_, _ = bot.Send(chatId, fmt.Sprintf("Hi %d", i), NewSendOpts().Message())
	}
}
