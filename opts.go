package telegram

import (
	"encoding/json"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// ParseMode is a parse_mode request parameter type.
type ParseMode string

const (
	// None is used for empty parse_mode.
	None ParseMode = ""
	// Markdown is "Markdown" parse_mode value.
	Markdown ParseMode = "Markdown"
	// HTML is "HTML" parse_mode value.
	HTML ParseMode = "HTML"

	// MaxMessageSize is maximum message character length.
	MaxMessageSize = 4096
	// MaxCaptionSize is maximum caption character length.
	MaxCaptionSize = 1024
)

// UpdateOpts is /getUpdates request options.
// See https://core.telegram.org/bots/api#getupdates
type UpdateOpts struct {
	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers of previously received updates.
	// By default, updates starting with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called with an offset
	// higher than its update_id. The negative offset can be specified to retrieve updates
	// starting from -offset update from the end of the updates queue.
	// All previous updates will be forgotten.
	Offset ID `json:"offset,omitempty"`
	// Limits the number of updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit int `json:"limit,omitempty"`
	// Timeout for long polling.
	TimeoutSecs int `json:"timeout,omitempty"`
	// List the types of updates you want your bot to receive.
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

func (o UpdateOpts) body() flu.BodyWriter {
	return flu.JSON(o)
}

type SendOpts struct {
	DisableNotification bool
	ReplyToMessageID    ID
	ReplyMarkup         ReplyMarkup
}

func (o *SendOpts) body(chatID ChatID, sendable baseSendable) (flu.BodyWriter, error) {
	isMediaGroup := sendable.kind() == "mediaGroup"
	var form *flu.FormBody
	if isMediaGroup {
		form = flu.Form()
	} else {
		form = flu.Form(sendable)
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

	return sendable.finalize(form)
}

type AnswerCallbackQueryOpts struct {
	Text      string `url:"text,omitempty"`
	ShowAlert bool   `url:"show_alert,omitempty"`
	URL       string `url:"url,omitempty"`
	CacheTime int    `url:"cache_time,omitempty"`
}

func (o *AnswerCallbackQueryOpts) body(id string) flu.BodyWriter {
	return flu.Form(o).Add("callback_query_id", id)
}
