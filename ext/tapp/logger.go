package tapp

import (
	"context"

	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api"
)

type Logger struct {
	telegram.Sender
}

func (l *Logger) Logf(ctx context.Context, level logf.Level, pattern string, values ...any) {

}
