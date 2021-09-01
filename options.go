package telegram

import (
	"encoding/json"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type SendOptions struct {
	DisableNotification bool
	ReplyToMessageID    ID
	ReplyMarkup         ReplyMarkup
}

func (o *SendOptions) body(chatID ChatID, item sendable) (flu.EncoderTo, error) {
	mediaGroup := item.kind() == "mediaGroup"
	form := new(fluhttp.Form)
	if !mediaGroup {
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
		if !mediaGroup && o.ReplyMarkup != nil {
			bytes, err := json.Marshal(o.ReplyMarkup)
			if err != nil {
				return nil, errors.Wrap(err, "serialize reply_markup")
			}
			form = form.Set("reply_markup", string(bytes))
		}
	}
	return item.body(form)
}

type CopyOptions struct {
	*SendOptions
	Caption   string    `url:"caption,omitempty"`
	ParseMode ParseMode `url:"parse_mode,omitempty"`
}

func (o *CopyOptions) body(chatID ChatID, ref MessageRef) (flu.EncoderTo, error) {
	form := new(fluhttp.Form).Value(o)
	form.Add("chat_id", chatID.queryParam())
	return ref.body(form)
}

type AnswerOptions struct {
	Text      string `url:"text,omitempty"`
	ShowAlert bool   `url:"show_alert,omitempty"`
	URL       string `url:"url,omitempty"`
	CacheTime int    `url:"cache_time,omitempty"`
}

func (o *AnswerOptions) body(id string) flu.EncoderTo {
	return new(fluhttp.Form).Value(o).Add("callback_query_id", id)
}
