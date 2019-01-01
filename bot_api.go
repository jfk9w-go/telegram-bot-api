package telegram

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
)

type ID int64

func ParseID(value string) (ID, error) {
	var id, err = strconv.ParseInt(value, 10, 64)
	return ID(id), err
}

func MustParseID(value string) ID {
	var id, err = ParseID(value)
	if err != nil {
		panic(err)
	}

	return id
}

func (id ID) Int64Value() int64 {
	return int64(id)
}

func (id ID) StringValue() string {
	return strconv.FormatInt(int64(id), 10)
}

type Username string

func ParseUsername(str string) (Username, error) {
	if strings.HasPrefix(str, "@") {
		return Username(str), nil
	}

	return "", errors.New("username must begin with a '@'")
}

func (username Username) StringValue() string {
	return string(username)
}

type ChatID interface {
	StringValue() string
}

type BotApi interface {
	GetMe() (*User, error)
	GetChat(ChatID) (*Chat, error)
	GetChatAdministrators(ChatID) ([]ChatMember, error)
	GetChatMember(ChatID, ID) (*ChatMember, error)
	GetUpdates(UpdatesOpts) ([]Update, error)
	Send(ChatID, interface{}, SendOpts) (*Message, error)
}

func NewBotApi(client *flu.Client, token string) BotApi {
	if client == nil {
		client = flu.NewClient(nil).
			ResponseHeaderTimeout(80 * time.Second)
	}

	return &BotApiImpl{
		client:  client,
		baseUri: "https://api.telegram.org/bot" + token,
	}
}
