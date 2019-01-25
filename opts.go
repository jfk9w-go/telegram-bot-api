package telegram

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/lego"

	"github.com/jfk9w-go/flu"
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

// UpdatesOpts is /getUpdates request options.
// See https://core.telegram.org/bots/api#getupdates
type UpdatesOpts struct {
	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers of previously received updates.
	// By default, updates starting with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called with an offset
	// higher than its update_id. The negative offset can be specified to retrieve updates
	// starting from -offset update from the end of the updates queue.
	// All previous updates will be forgotten.
	Offset_ ID `json:"offset,omitempty"`
	// Limits the number of updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit_ int `json:"limit,omitempty"`
	// Timeout for long polling.
	Timeout_ int `json:"timeout,omitempty"`
	// List the types of updates you want your bot to receive.
	AllowedUpdates_ []string `json:"allowed_updates,omitempty"`
}

// Offset sets the update offset and returns itself.
func (opts *UpdatesOpts) Offset(offset ID) *UpdatesOpts {
	opts.Offset_ = offset
	return opts
}

// Limit sets the limit and returns itself.
func (opts *UpdatesOpts) Limit(limit int) *UpdatesOpts {
	opts.Limit_ = limit
	return opts
}

// Timeout sets the timeout and returns itself.
func (opts *UpdatesOpts) Timeout(timeout time.Duration) *UpdatesOpts {
	opts.Timeout_ = int(timeout.Seconds())
	return opts
}

// AllowedUpdates sets the allowed updates and returns itself.
func (opts *UpdatesOpts) AllowedUpdates(allowedUpdates ...string) *UpdatesOpts {
	opts.AllowedUpdates_ = allowedUpdates
	return opts
}

func (opts *UpdatesOpts) body() flu.BodyWriter {
	return flu.JSON(opts)
}

type SendOpts interface {
	body(ChatID, interface{}) flu.BodyWriter
	entityType() string
}

type BaseSendOpts struct {
	DisableNotification_ bool `url:"disable_notification,omitempty"`
	ReplyToMessageID_    ID   `url:"reply_to_message_id,omitempty"`
}

func NewSendOpts() *BaseSendOpts {
	return new(BaseSendOpts)
}

func (opts *BaseSendOpts) DisableNotification(disableNotification bool) *BaseSendOpts {
	opts.DisableNotification_ = disableNotification
	return opts
}

func (opts *BaseSendOpts) ReplyToMessageID(replyToMessageID ID) *BaseSendOpts {
	opts.ReplyToMessageID_ = replyToMessageID
	return opts
}

func (opts *BaseSendOpts) Message() *MessageOpts {
	return &MessageOpts{
		BaseSendOpts: opts,
	}
}

func (opts *BaseSendOpts) Media() *MediaOpts {
	return &MediaOpts{
		BaseSendOpts: opts,
	}
}

func (opts *BaseSendOpts) MediaGroup() *MediaGroupOpts {
	return &MediaGroupOpts{
		BaseSendOpts: opts,
	}
}

type MessageOpts struct {
	*BaseSendOpts
	ParseMode_             ParseMode `url:"parse_mode,omitempty"`
	DisableWebPagePreview_ bool      `url:"disable_web_page_preview,omitempty"`
	ReplyMarkup_           ReplyMarkup
}

func (opts *MessageOpts) ParseMode(parseMode ParseMode) *MessageOpts {
	opts.ParseMode_ = parseMode
	return opts
}

func (opts *MessageOpts) DisableWebPagePreview(disableWebPagePreview bool) *MessageOpts {
	opts.DisableWebPagePreview_ = disableWebPagePreview
	return opts
}

func (opts *MessageOpts) ReplyMarkup(replyMarkup ReplyMarkup) *MessageOpts {
	opts.ReplyMarkup_ = replyMarkup
	return opts
}

func (opts *MessageOpts) entityType() string {
	return "Message"
}

func (opts *MessageOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	form := flu.Form(opts).
		Add("chat_id", chatID.queryParam()).
		Add("text", entity.(string))

	if opts.ReplyMarkup_ != nil {
		replyMarkupJSON, err := json.Marshal(opts.ReplyMarkup_)
		lego.Check(err)
		form.Add("reply_markup", string(replyMarkupJSON))
	}

	return form
}

type MediaOpts struct {
	*BaseSendOpts
	media_     interface{}
	Type_      string    `json:"type,omitempty"`
	Media_     string    `json:"media,omitempty"`
	Caption_   string    `json:"caption,omitempty" url:"caption,omitempty"`
	ParseMode_ ParseMode `json:"parse_mode,omitempty" url:"parse_mode,omitempty"`
}

func NewMediaOpts(entity interface{}) *MediaOpts {
	return &MediaOpts{
		media_: entity,
	}
}

func (opts *MediaOpts) Photo() *MediaOpts {
	opts.Type_ = "photo"
	return opts
}

func (opts *MediaOpts) Video() *MediaOpts {
	opts.Type_ = "video"
	return opts
}

func (opts *MediaOpts) Document() *MediaOpts {
	opts.Type_ = "document"
	return opts
}

func (opts *MediaOpts) Caption(caption string) *MediaOpts {
	opts.Caption_ = caption
	return opts
}

func (opts *MediaOpts) ParseMode(parseMode ParseMode) *MediaOpts {
	opts.ParseMode_ = parseMode
	return opts
}

func (opts *MediaOpts) entityType() string {
	return strings.ToTitle(opts.Type_)
}

func (opts *MediaOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	form := flu.Form(opts).Add("chat_id", chatID.queryParam())
	switch entity := entity.(type) {
	case string:
		return form.Add(opts.Type_, entity)

	case flu.ReadResource:
		return form.Multipart().Resource(opts.Type_, entity)
	}

	log.Panicf("invalid entity type: %T", entity)
	return nil
}

type MediaGroupOpts struct {
	*BaseSendOpts
}

func (opts *MediaGroupOpts) entityType() string {
	return "MediaGroup"
}

func (opts *MediaGroupOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	form := flu.Form(opts).Add("chat_id", chatID.queryParam())
	media := entity.([]*MediaOpts)
	isMultipart := false
	for i, media := range media {
		switch entity := media.media_.(type) {
		case string:
			media.Media_ = entity

		case flu.ReadResource:
			id := "media" + strconv.Itoa(i)
			form.Multipart().Resource(id, entity)
			media.Media_ = "attach://" + id
			isMultipart = true
		}
	}

	mediaJSON, err := json.Marshal(media)
	lego.Check(err)
	form.Add("media", string(mediaJSON))

	if isMultipart {
		return form.Multipart()
	} else {
		return form
	}
}

type AnswerCallbackQueryOpts struct {
	Text_      string `url:"text,omitempty"`
	ShowAlert_ bool   `url:"show_alert,omitempty"`
	URL_       string `url:"url,omitempty"`
	CacheTime_ int    `url:"cache_time,omitempty"`
}

func NewAnswerCallbackQueryOpts() *AnswerCallbackQueryOpts {
	return new(AnswerCallbackQueryOpts)
}

func (opts *AnswerCallbackQueryOpts) Text(text string) *AnswerCallbackQueryOpts {
	opts.Text_ = text
	return opts
}

func (opts *AnswerCallbackQueryOpts) ShowAlert(showAlert bool) *AnswerCallbackQueryOpts {
	opts.ShowAlert_ = showAlert
	return opts
}

func (opts *AnswerCallbackQueryOpts) URL(url string) *AnswerCallbackQueryOpts {
	opts.URL_ = url
	return opts
}

func (opts *AnswerCallbackQueryOpts) CacheTime(cacheTime time.Duration) *AnswerCallbackQueryOpts {
	opts.CacheTime_ = int(cacheTime.Minutes())
	return opts
}

func (opts *AnswerCallbackQueryOpts) body(id string) flu.BodyWriter {
	return flu.Form(opts).Add("callback_query_id", id)
}
