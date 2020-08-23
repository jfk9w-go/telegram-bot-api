package telegram

import "context"

type Client interface {
	GetMe(ctx context.Context) (*User, error)
	DeleteMessage(ctx context.Context, chatID ChatID, messageID ID) (bool, error)
	ExportChatInviteLink(ctx context.Context, chatID ChatID) (string, error)
	GetChat(ctx context.Context, chatID ChatID) (*Chat, error)
	GetChatAdministrators(ctx context.Context, chatID ChatID) ([]ChatMember, error)
	GetChatMember(ctx context.Context, chatID ChatID, userID ID) (*ChatMember, error)
	AnswerCallbackQuery(ctx context.Context, id string, options AnswerCallbackQueryOptions) (bool, error)
	Send(ctx context.Context, chatID ChatID, item Sendable, options *SendOptions) (*Message, error)
	SendMediaGroup(ctx context.Context, chatID ChatID, media []Media, options *SendOptions) ([]Message, error)
	Ask(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error)
	Answer(message *Message) bool
	Username() string
}

func InlineKeyboard(rows ...[][3]string) ReplyMarkup {
	keyboard := make([][]InlineKeyboardButton, len(rows))
	for i, row := range rows {
		keyboard[i] = make([]InlineKeyboardButton, len(row))
		for j, button := range row {
			keyboard[i][j] = InlineKeyboardButton{
				Text:         button[0],
				CallbackData: button[1] + " " + button[2],
			}
		}
	}
	return &InlineKeyboardMarkup{keyboard}
}
