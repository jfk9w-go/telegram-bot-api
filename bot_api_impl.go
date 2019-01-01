package telegram

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/jfk9w-go/flu"
)

type BotApiImpl struct {
	client  *flu.Client
	baseUri string
}

func (api *BotApiImpl) endpoint(method string) string {
	return api.baseUri + method
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

func defaultResponseProcessor(value interface{}) flu.ResponseProcessor {
	return func(resp *http.Response) (err error) {
		if _, ok := allowedStatusCodes[resp.StatusCode]; !ok {
			return fmt.Errorf("invalid status code: %d", resp.StatusCode)
		}

		var data []byte
		data, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return
		}

		var r = new(Response)
		err = json.Unmarshal(data, r)
		if err != nil {
			return
		}

		return r.Parse(value)
	}
}

func (api *BotApiImpl) GetMe() (*User, error) {
	var user = new(User)
	return user, api.client.NewRequest().
		Endpoint(api.endpoint("/getMe")).
		Get().Retrieve().
		ProcessResponse(defaultResponseProcessor(user)).Done()
}

func (api *BotApiImpl) GetChat(chatId ChatID) (*Chat, error) {
	var chat = new(Chat)
	return chat, api.client.NewRequest().
		Endpoint(api.endpoint("/getChat")).
		QueryParam("chat_id", chatId.StringValue()).
		Get().Retrieve().
		ProcessResponse(defaultResponseProcessor(chat)).
		Done()
}

func (api *BotApiImpl) GetChatAdministrators(chatId ChatID) ([]ChatMember, error) {
	var members = make([]ChatMember, 0)
	return members, api.client.NewRequest().
		Endpoint(api.endpoint("/getChatAdministrators")).
		QueryParam("chat_id", chatId.StringValue()).
		Get().Retrieve().
		ProcessResponse(defaultResponseProcessor(&members)).
		Done()
}

func (api *BotApiImpl) GetChatMember(chatId ChatID, userId ID) (*ChatMember, error) {
	var member = new(ChatMember)
	return member, api.client.NewRequest().
		Endpoint(api.endpoint("/getChatMember")).
		QueryParam("chat_id", chatId.StringValue()).
		QueryParam("user_id", userId.StringValue()).
		Get().Retrieve().
		ProcessResponse(defaultResponseProcessor(member)).
		Done()
}

func (api *BotApiImpl) GetUpdates(opts UpdatesOpts) ([]Update, error) {
	var updates = make([]Update, 0)
	return updates, api.client.NewRequest().
		Endpoint(api.endpoint("/getUpdates")).
		Post().Body(opts.body()).Retrieve().
		ProcessResponse(defaultResponseProcessor(&updates)).
		Done()
}

func (api *BotApiImpl) Send(chatId ChatID, entity interface{}, opts SendOpts) (*Message, error) {
	var message = new(Message)
	return message, api.client.NewRequest().
		Endpoint(api.endpoint("/send" + opts.entityType())).
		Post().Body(opts.body(chatId, entity)).Retrieve().
		ProcessResponse(defaultResponseProcessor(message)).
		Done()
}
