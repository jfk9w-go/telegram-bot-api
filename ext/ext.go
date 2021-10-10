package ext

import (
	"context"
	"fmt"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

func HTML(ctx context.Context, sender telegram.Sender, chatID telegram.ID) *html.Writer {
	return &html.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    sender,
				ID:        chatID,
				ParseMode: telegram.HTML,
			},
			PageSize: telegram.MaxMessageSize,
		},
	}
}

func DefaultStart(version string) telegram.CommandListenerFunc {
	return func(ctx context.Context, tgclient telegram.Client, cmd *telegram.Command) error {
		text := fmt.Sprintf("User ID: %d\nChat ID: %s\nBot: %s\nVersion: %s",
			cmd.User.ID, cmd.Chat.ID, tgclient.Username(), version)
		return cmd.Reply(ctx, tgclient, text)
	}
}
