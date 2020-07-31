package telegram

import (
	"context"
	"net/http"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
)

// BaseClient is the Telegram Bot API client implementation.
// It can not be instantiated by package users.
// Instead, it should be used as part of Bot.
type BaseClient struct {
	client  *fluhttp.Client
	baseURI string
}

var ValidStatusCodes = []int{
	http.StatusOK,
	http.StatusSeeOther,
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
}

func NewBaseClient(client *fluhttp.Client, token string) BaseClient {
	if token == "" {
		panic("token must not be empty")
	}
	if client == nil {
		client = fluhttp.NewClient(http.DefaultClient)
	}
	return BaseClient{
		client:  client.AcceptStatus(ValidStatusCodes...),
		baseURI: "https://api.telegram.org/bot" + token,
	}
}

// Use this method to receive incoming updates using long polling.
// An Array of Update objects is returned.
// See https://core.telegram.org/bots/api#getupdates
func (c BaseClient) GetUpdates(ctx context.Context, options *GetUpdatesOptions) ([]Update, error) {
	updates := make([]Update, 0)
	return updates, c.Execute(ctx, "/getUpdates", flu.JSON{options}, &updates)
}

// A simple method for testing your bot's auth token. Requires no parameters.
// Returns basic information about the bot in form of a User object.
// See https://core.telegram.org/bots/api#getme
func (c BaseClient) GetMe(ctx context.Context) (*User, error) {
	user := new(User)
	return user, c.Execute(ctx, "/getMe", nil, user)
}

// Use this method to delete a message, including service messages, with the following limitations:
// - A message can only be deleted if it was sent less than 48 hours ago.
// - Bots can delete outgoing messages in private chats, groups, and supergroups.
// - Bots granted can_post_messages permissions can delete outgoing messages in channels.
// - If the bot is an administrator of a group, it can delete any message there.
// - If the bot has can_delete_messages permission in a supergroup or a updateChannel, it can delete any message there.
// Returns True on success.
// See
//    https://core.telegram.org/bots/api#deletemessage
func (c BaseClient) DeleteMessage(ctx context.Context, chatID ChatID, messageID ID) (bool, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam()).
		Set("message_id", messageID.queryParam())
	var ok bool
	return ok, c.Execute(ctx, "/deleteMessage", body, &ok)
}

// Use this method to get up to date information about the chat (current name of
// the user for one-on-one conversations, current username of a user, group or updateChannel, etc.).
// Returns a Chat object on success.
// See https://core.telegram.org/bots/api#getchat
func (c BaseClient) GetChat(ctx context.Context, chatID ChatID) (*Chat, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())
	chat := new(Chat)
	return chat, c.Execute(ctx, "/getChat", body, chat)
}

// Use this method to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information about
// all chat administrators except other bots. If the chat is a group or a supergroup and
// no administrators were appointed, only the creator will be returned.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c BaseClient) GetChatAdministrators(ctx context.Context, chatID ChatID) ([]ChatMember, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())
	members := make([]ChatMember, 0)
	return members, c.Execute(ctx, "/getChatAdministrators", body, &members)
}

// Use this method to get information about a member of a chat.
// Returns a ChatMember object on success.
// See https://core.telegram.org/bots/api#getchatmember
func (c BaseClient) GetChatMember(ctx context.Context, chatID ChatID, userID ID) (*ChatMember, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam()).
		Set("user_id", userID.queryParam())
	member := new(ChatMember)
	return member, c.Execute(ctx, "/getChatMember", body, member)
}

// Use this method to send answers to callback queries sent from inline keyboards.
// The answer will be displayed to the user as a notification at the top of the chat screen or as an alert.
// On success, True is returned.
// https://core.telegram.org/bots/api#answercallbackquery
func (c BaseClient) AnswerCallbackQuery(ctx context.Context, id string, options *AnswerCallbackQueryOptions) (bool, error) {
	var ok bool
	return ok, c.Execute(ctx, "/answerCallbackQuery", options.body(id), &ok)
}

func (c BaseClient) Execute(ctx context.Context, method string, body flu.EncoderTo, resp interface{}) error {
	return c.client.
		POST(c.method(method)).
		BodyEncoder(body).
		Context(ctx).
		Execute().
		DecodeBody(newResponse(resp)).
		Error
}

func (c BaseClient) method(method string) string {
	return c.baseURI + method
}
