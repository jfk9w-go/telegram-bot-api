package telegram

import (
	"encoding/json"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// GatewaySendDelay is a delay between two consecutive /send* API calls per bot token.
var GatewaySendDelay = 30 * time.Millisecond

// SendDelays are delays between two consecutive /send* API calls per chat with a given type.
var SendDelays = map[ChatType]time.Duration{
	PrivateChat: 30 * time.Millisecond,
	GroupChat:   time.Second,
	Supergroup:  time.Second,
	Channel:     0,
}

type SendOptions struct {
	DisableNotification bool
	ReplyToMessageID    ID
	ReplyMarkup         ReplyMarkup
}

func (o *SendOptions) body(chatID ChatID, item genericSendItem) (flu.BodyWriter, error) {
	isMediaGroup := item.kind() == "mediaGroup"
	var form *flu.FormBody
	if isMediaGroup {
		form = flu.Form()
	} else {
		form = flu.Form(item)
	}

	form.Set("chat_id", chatID.queryParam())
	if o != nil {
		if o.DisableNotification {
			form.Set("disable_notification", "1")
		}

		if o.ReplyToMessageID != 0 {
			form.Set("reply_to_message_id", o.ReplyToMessageID.queryParam())
		}

		if !isMediaGroup && o.ReplyMarkup != nil {
			bytes, err := json.Marshal(o.ReplyMarkup)
			if err != nil {
				return nil, errors.Wrap(err, "failed to serialize reply_markup")
			}

			form.Set("reply_markup", string(bytes))
		}
	}

	return item.write(form)
}
