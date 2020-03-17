package telegram

import (
	"encoding/json"
	"time"

	fluhttp "github.com/jfk9w-go/flu/http"

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

func (o *SendOptions) body(chatID ChatID, item sendable) (flu.EncoderTo, error) {
	isMediaGroup := item.kind() == "mediaGroup"
	form := fluhttp.Form{}
	if !isMediaGroup {
		form = form.Value(item)
	}
	form = form.Set("chat_id", chatID.queryParam())
	if o != nil {
		if o.DisableNotification {
			form = form.Set("disable_notification", "1")
		}
		if o.ReplyToMessageID != 0 {
			form = form.Set("reply_to_message_id", o.ReplyToMessageID.queryParam())
		}
		if !isMediaGroup && o.ReplyMarkup != nil {
			bytes, err := json.Marshal(o.ReplyMarkup)
			if err != nil {
				return nil, errors.Wrap(err, "failed to serialize reply_markup")
			}
			form = form.Set("reply_markup", string(bytes))
		}
	}
	return item.body(form)
}

type AnswerCallbackQueryOptions struct {
	Text      string `url:"text,omitempty"`
	ShowAlert bool   `url:"show_alert,omitempty"`
	URL       string `url:"url,omitempty"`
	CacheTime int    `url:"cache_time,omitempty"`
}

func (o *AnswerCallbackQueryOptions) body(id string) flu.EncoderTo {
	return fluhttp.Form{}.Value(o).Add("callback_query_id", id)
}
