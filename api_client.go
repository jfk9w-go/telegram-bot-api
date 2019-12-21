package telegram

import (
	"net/http"
	"time"

	"github.com/jfk9w-go/flu"
)

// apiClient is the Telegram Bot API client implementation.
// It can not be instantiated by package users.
// Instead, it should be used as part of Bot.
type apiClient struct {
	http    *flu.Client
	baseURI string
}

var acceptedResponseCodes = []int{
	http.StatusOK,
	http.StatusSeeOther,
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
}

func newApiClient(http *flu.Client, token string) *apiClient {
	if token == "" {
		panic("token must not be empty")
	}
	if http == nil {
		http = flu.NewTransport().
			ResponseHeaderTimeout(2 * time.Minute).
			NewClient().
			AcceptResponseCodes(acceptedResponseCodes...)
	}
	return &apiClient{
		http:    http,
		baseURI: "https://api.telegram.org/bot" + token,
	}
}

// Use this method to receive incoming updates using long polling.
// An Array of Update objects is returned.
// See https://core.telegram.org/bots/api#getupdates
func (c *apiClient) GetUpdates(options *GetUpdatesOptions) ([]Update, error) {
	updates := make([]Update, 0)
	return updates, c.http.NewRequest().
		POST().
		Resource(c.method("/getUpdates")).
		Body(options.body()).
		Send().
		Read(newResponse(&updates)).
		Error
}

// A simple method for testing your bot's auth token. Requires no parameters.
// Returns basic information about the bot in form of a User object.
// See https://core.telegram.org/bots/api#getme
func (c *apiClient) GetMe() (*User, error) {
	user := new(User)
	return user, c.http.NewRequest().
		GET().
		Resource(c.method("/getMe")).
		Send().
		Read(newResponse(user)).
		Error
}

// This is an umbrella method used for various /send* API calls.
// The method is private since callers can hit API limits
// and get HTTP 429 error in case of intense usage.
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
//   https://core.telegram.org/bots/api#senddocument
//   https://core.telegram.org/bots/api#sendmediagroup
func (c *apiClient) send(url string, body flu.BodyWriter, resp interface{}) error {
	return c.http.NewRequest().
		POST().
		Resource(url).
		Body(body).
		Send().
		Read(newResponse(resp)).
		Error
}

// Use this method to delete a message, including service messages, with the following limitations:
//- A message can only be deleted if it was workers less than 48 hours ago.
//- Bots can delete outgoing messages in private chats, groups, and supergroups.
//- Bots granted can_post_messages permissions can delete outgoing messages in channels.
//- If the bot is an administrator of a group, it can delete any message there.
//- If the bot has can_delete_messages permission in a supergroup or a updateChannel, it can delete any message there.
// Returns True on success.
// See
//    https://core.telegram.org/bots/api#deletemessage
func (c *apiClient) DeleteMessage(chatID ChatID, messageID ID) (bool, error) {
	var r bool
	return r, c.http.NewRequest().
		GET().
		Resource(c.method("/deleteMessage")).
		QueryParam("chat_id", chatID.queryParam()).
		QueryParam("message_id", messageID.queryParam()).
		Send().
		Read(newResponse(&r)).
		Error
}

// Use this method to get up to date information about the chat (current name of
// the user for one-on-one conversations, current username of a user, group or updateChannel, etc.).
// Returns a Chat object on success.
// See https://core.telegram.org/bots/api#getchat
func (c *apiClient) GetChat(chatID ChatID) (*Chat, error) {
	chat := new(Chat)
	return chat, c.http.NewRequest().
		GET().
		Resource(c.method("/getChat")).
		QueryParam("chat_id", chatID.queryParam()).
		Send().
		Read(newResponse(chat)).
		Error
}

// Use this method to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information about
// all chat administrators except other bots. If the chat is a group or a supergroup and
// no administrators were appointed, only the creator will be returned.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c *apiClient) GetChatAdministrators(chatID ChatID) ([]ChatMember, error) {
	members := make([]ChatMember, 0)
	return members, c.http.NewRequest().
		GET().
		Resource(c.method("/getChatAdministrators")).
		QueryParam("chat_id", chatID.queryParam()).
		Send().
		Read(newResponse(&members)).
		Error
}

// Use this method to get information about a member of a chat.
// Returns a ChatMember object on success.
// See https://core.telegram.org/bots/api#getchatmember
func (c *apiClient) GetChatMember(chatID ChatID, userID ID) (*ChatMember, error) {
	member := new(ChatMember)
	return member, c.http.NewRequest().
		GET().
		Resource(c.method("/getChatMember")).
		QueryParam("chat_id", chatID.queryParam()).
		QueryParam("user_id", userID.queryParam()).
		Send().
		Read(newResponse(member)).
		Error
}

// Use this method to send answers to callback queries workers from inline keyboards.
// The answer will be displayed to the user as a notification at the top of the chat screen or as an alert.
// On success, True is returned.
// https://core.telegram.org/bots/api#answercallbackquery
func (c *apiClient) AnswerCallbackQuery(id string, options *AnswerCallbackQueryOptions) (bool, error) {
	var r bool
	return r, c.http.NewRequest().
		POST().
		Resource(c.method("/answerCallbackQuery")).
		Body(options.body(id)).
		Send().
		Read(newResponse(&r)).
		Error
}

func (c *apiClient) method(method string) string {
	return c.baseURI + method
}
