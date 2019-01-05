package telegram

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jfk9w-go/flu"
)

// Client is the Telegram Bot API client implementation.
// It can not be instantiated by package users.
// Instead, it should be used as part of Bot.
type Client struct {
	http    *flu.Client
	baseURI string
}

func newClient(http *flu.Client, token string) *Client {
	if http == nil {
		http = flu.NewClient(nil).
			ResponseHeaderTimeout(80 * time.Second)
	}

	return &Client{
		http:    http,
		baseURI: "https://api.telegram.org/bot" + token,
	}
}

// Use this method to receive incoming updates using long polling.
// An Array of Update objects is returned.
// See https://core.telegram.org/bots/api#getupdates
func (c *Client) GetUpdates(opts *UpdatesOpts) ([]Update, error) {
	updates := make([]Update, 0)
	return updates, c.http.NewRequest().
		Endpoint(c.endpoint("/getUpdates")).
		Body(opts.body()).
		Post().
		ReadResponseFunc(defaultResponseProcessor(&updates)).
		Error
}

// A simple method for testing your bot's auth token. Requires no parameters.
// Returns basic information about the bot in form of a User object.
// See https://core.telegram.org/bots/api#getme
func (c *Client) GetMe() (*User, error) {
	user := new(User)
	return user, c.http.NewRequest().
		Endpoint(c.endpoint("/getMe")).
		Get().
		ReadResponseFunc(defaultResponseProcessor(user)).
		Error
}

// This is an umbrella method used for various /send* API calls.
// Generally, entity should be either string or flu.ReadResource.
// Via opts you can specify additional request options.
// The method is private since callers can hit API limits
// and get HTTP 429 error in case of intense usage.
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
func (c *Client) send(chatID ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	message := new(Message)
	return message, c.http.NewRequest().
		Endpoint(c.endpoint("/send" + opts.entityType())).
		Body(opts.body(chatID, entity)).
		Post().
		ReadResponseFunc(defaultResponseProcessor(message)).
		Error
}

// Use this method to get up to date information about the chat (current name of
// the user for one-on-one conversations, current username of a user, group or channel, etc.).
// Returns a Chat object on success.
// See https://core.telegram.org/bots/api#getchat
func (c *Client) GetChat(chatID ChatID) (*Chat, error) {
	chat := new(Chat)
	return chat, c.http.NewRequest().
		Endpoint(c.endpoint("/getChat")).
		QueryParam("chat_id", chatID.StringValue()).
		Get().
		ReadResponseFunc(defaultResponseProcessor(chat)).
		Error
}

// Use this method to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information about
// all chat administrators except other bots. If the chat is a group or a supergroup and
// no administrators were appointed, only the creator will be returned.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c *Client) GetChatAdministrators(chatID ChatID) ([]ChatMember, error) {
	members := make([]ChatMember, 0)
	return members, c.http.NewRequest().
		Endpoint(c.endpoint("/getChatAdministrators")).
		QueryParam("chat_id", chatID.StringValue()).
		Get().
		ReadResponseFunc(defaultResponseProcessor(&members)).
		Error
}

// Use this method to get information about a member of a chat.
// Returns a ChatMember object on success.
// See https://core.telegram.org/bots/api#getchatmember
func (c *Client) GetChatMember(chatID ChatID, userID ID) (*ChatMember, error) {
	member := new(ChatMember)
	return member, c.http.NewRequest().
		Endpoint(c.endpoint("/getChatMember")).
		QueryParam("chat_id", chatID.StringValue()).
		QueryParam("user_id", userID.StringValue()).
		Get().
		ReadResponseFunc(defaultResponseProcessor(member)).
		Error
}

func (c *Client) endpoint(method string) string {
	return c.baseURI + method
}

func defaultResponseProcessor(value interface{}) flu.ReadResponseFunc {
	return func(resp *http.Response) error {
		if _, ok := allowedStatusCodes[resp.StatusCode]; !ok {
			return fmt.Errorf("invalid status code: %d", resp.StatusCode)
		}

		data, err := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}

		r := new(response)
		err = json.Unmarshal(data, r)
		if err != nil {
			return err
		}

		return r.parse(value)
	}
}

var allowedStatusCodes = map[int]struct{}{
	http.StatusOK:                  {},
	http.StatusSeeOther:            {},
	http.StatusBadRequest:          {},
	http.StatusUnauthorized:        {},
	http.StatusForbidden:           {},
	http.StatusNotFound:            {},
	http.StatusTooManyRequests:     {},
	http.StatusInternalServerError: {},
}
