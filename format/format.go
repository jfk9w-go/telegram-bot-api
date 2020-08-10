package format

import (
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type NameSender interface {
	telegram.Sender
	Username() string
}

type HTML struct {
	NameSender
	ChatIDs []telegram.ChatID
	builder strings.Builder
}
