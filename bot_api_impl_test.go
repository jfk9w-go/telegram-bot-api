package telegram

import (
	"log"
	"os"
	"testing"

	"github.com/jfk9w-go/flu"
)

//noinspection GoUnhandledErrorResult
func TestBotApiImpl(t *testing.T) {
	var (
		token  = os.Getenv("TELEGRAM_BOT_API_TOKEN")
		chatId = MustParseID(os.Getenv("TELEGRAM_CHAT"))
		api    = NewBotApi(nil, token)

		user    *User
		chat    *Chat
		message *Message

		err error
	)

	user, err = api.GetMe()
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Received %+v", user)

	chat, err = api.GetChat(chatId)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Received %+v", chat)

	message, err = api.Send(chatId, "Hi", NewSendOpts().Message())
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Received %+v", message)

	message, err = api.Send(chatId,
		flu.NewFileSystemResource("testdata/check.png"),
		NewSendOpts().Media().Caption("Hi").Photo())

	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Received %+v", message)
}
