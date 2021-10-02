package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

type BaseClient func(string) *fluhttp.Request

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

type EndpointFunc func(token, method string) string

var DefaultEndpoint EndpointFunc = func(token, method string) string {
	return "https://api.telegram.org/bot" + token + "/" + method
}

func NewBaseClient(client *fluhttp.Client, token string) BaseClient {
	return NewBaseClientWithEndpoint(client, token, DefaultEndpoint)
}

func NewBaseClientWithEndpoint(client *fluhttp.Client, token string, endpoint EndpointFunc) BaseClient {
	if token == "" {
		panic("token must not be empty")
	}
	if client == nil {
		client = fluhttp.NewClient(http.DefaultClient)
	}
	client = client.AcceptStatus(ValidStatusCodes...)
	if endpoint == nil {
		endpoint = DefaultEndpoint
	}
	return func(method string) *fluhttp.Request { return client.POST(endpoint(token, method)) }
}

// GetUpdates is used to receive incoming updates using long polling.
// An Array of Update objects is returned.
// See https://core.telegram.org/bots/api#getupdates
func (c BaseClient) GetUpdates(ctx context.Context, options GetUpdatesOptions) ([]Update, error) {
	updates := make([]Update, 0)
	return updates, c.Execute(ctx, "getUpdates", flu.JSON(options), &updates)
}

// GetMe is a simple method for testing your bot's auth token. Requires no parameters.
// Returns basic information about the bot in form of a User object.
// See https://core.telegram.org/bots/api#getme
func (c BaseClient) GetMe(ctx context.Context) (*User, error) {
	user := new(User)
	return user, c.Execute(ctx, "getMe", nil, user)
}

func (c BaseClient) ForwardMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *SendOptions) (ID, error) {
	var messageID ID
	form, err := options.body(chatID, ref)
	if err != nil {
		return messageID, err
	}

	return messageID, c.Execute(ctx, "forwardMessage", form, &messageID)
}

func (c BaseClient) CopyMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *CopyOptions) (ID, error) {
	var resp struct {
		MessageID ID `json:"message_id"`
	}

	form, err := options.body(chatID, ref)
	if err != nil {
		return 0, err
	}

	return resp.MessageID, c.Execute(ctx, "copyMessage", form, &resp)
}

// DeleteMessage is used to delete a message, including service messages, with the following limitations:
// - A message can only be deleted if it was sent less than 48 hours ago.
// - Bots can delete outgoing messages in private chats, groups, and supergroups.
// - Bots granted can_post_messages permissions can delete outgoing messages in channels.
// - If the bot is an administrator of a group, it can delete any message there.
// - If the bot has can_delete_messages permission in a supergroup or a updateChannel, it can delete any message there.
// Returns True on success.
// See
//    https://core.telegram.org/bots/api#deletemessage
func (c BaseClient) DeleteMessage(ctx context.Context, ref MessageRef) error {
	var ok bool
	if err := c.Execute(ctx, "deleteMessage", ref.form(), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c BaseClient) EditMessageReplyMarkup(ctx context.Context, ref MessageRef, markup ReplyMarkup) (*Message, error) {
	markupJSON, err := json.Marshal(markup)
	if err != nil {
		return nil, err
	}

	form := ref.form().Set("reply_markup", string(markupJSON))
	message := new(Message)
	if err := c.Execute(ctx, "editMessageReplyMarkup", form, &message); err != nil {
		if tgerr := new(Error); errors.As(err, tgerr) &&
			strings.Contains(tgerr.Description, "message is not modified") {
			return nil, nil
		}

		return nil, err
	}

	return message, nil
}

func (c BaseClient) ExportChatInviteLink(ctx context.Context, chatID ChatID) (string, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())
	var inviteLink string
	return inviteLink, c.Execute(ctx, "exportChatInviteLink", body, &inviteLink)
}

// GetChat is used to get up to date information about the chat (current name of
// the user for one-on-one conversations, current username of a user, group or updateChannel, etc.).
// Returns a Chat object on success.
// See https://core.telegram.org/bots/api#getchat
func (c BaseClient) GetChat(ctx context.Context, chatID ChatID) (*Chat, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())
	chat := new(Chat)
	return chat, c.Execute(ctx, "getChat", body, chat)
}

// GetChatAdministrators is used to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information about
// all chat administrators except other bots. If the chat is a group or a supergroup and
// no administrators were appointed, only the creator will be returned.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c BaseClient) GetChatAdministrators(ctx context.Context, chatID ChatID) ([]ChatMember, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())
	members := make([]ChatMember, 0)
	return members, c.Execute(ctx, "getChatAdministrators", body, &members)
}

func (c BaseClient) GetChatMemberCount(ctx context.Context, chatID ChatID) (int64, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam())

	var count int64
	return count, c.Execute(ctx, "getChatMemberCount", body, &count)
}

// GetChatMember is used to get information about a member of a chat.
// Returns a ChatMember object on success.
// See https://core.telegram.org/bots/api#getchatmember
func (c BaseClient) GetChatMember(ctx context.Context, chatID ChatID, userID ID) (*ChatMember, error) {
	body := new(fluhttp.Form).
		Set("chat_id", chatID.queryParam()).
		Set("user_id", userID.queryParam())
	member := new(ChatMember)
	return member, c.Execute(ctx, "getChatMember", body, member)
}

// AnswerCallbackQuery is used to send answers to callback queries sent from inline keyboards.
// The answer will be displayed to the user as a notification at the top of the chat screen or as an alert.
// On success, True is returned.
// https://core.telegram.org/bots/api#answercallbackquery
func (c BaseClient) AnswerCallbackQuery(ctx context.Context, id string, options *AnswerOptions) error {
	var ok bool
	if err := c.Execute(ctx, "answerCallbackQuery", options.body(id), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c BaseClient) Execute(ctx context.Context, method string, body flu.EncoderTo, resp interface{}) error {
	return c(method).
		BodyEncoder(body).
		Context(ctx).
		Execute().
		DecodeBody(newResponse(resp)).
		Error
}
