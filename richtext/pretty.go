package richtext

import (
	"context"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Message struct {
	Notify bool
	Text   []string
	Format telegram.ParseMode
}

func (m Message) Send(ctx context.Context, client telegram.Client, chatID telegram.ChatID) error {
	for _, part := range m.Text {
		if _, err := client.Send(ctx, chatID,
			&telegram.Text{
				Text:                  part,
				ParseMode:             m.Format,
				DisableWebPagePreview: false},
			&telegram.SendOptions{DisableNotification: !m.Notify}); err != nil {
			return err
		}
	}

	return nil
}

func (m Message) First() string {
	if len(m.Text) > 0 {
		return m.Text[0]
	} else {
		return ""
	}
}
