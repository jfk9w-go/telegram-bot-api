package telegram

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

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

// BaseOpts is a base type used for building various request options.
type BaseOpts url.Values

func (opts BaseOpts) values() url.Values {
	return url.Values(opts)
}

// Add adds a key-value pair to the map and returns itself.
func (opts BaseOpts) Add(key, value string) BaseOpts {
	opts.values().Add(key, value)
	return opts
}

// AddAll adds all key-value pairs for value in values to the map and returns itself.
func (opts BaseOpts) AddAll(key string, values ...string) BaseOpts {
	for _, value := range values {
		opts.values().Add(key, value)
	}

	return opts
}

// Set sets a key-value pair in the underlying map.
func (opts BaseOpts) Set(key, value string) BaseOpts {
	opts.values().Set(key, value)
	return opts
}

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
	Offset *ID
	// Limits the number of updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit *int
	// Timeout for long polling.
	Timeout *time.Duration
	// List the types of updates you want your bot to receive.
	AllowedUpdates []string
}

// SetOffset sets the update offset and returns itself.
func (opts *UpdatesOpts) SetOffset(offset ID) *UpdatesOpts {
	opts.Offset = &offset
	return opts
}

// SetLimit sets the limit and returns itself.
func (opts *UpdatesOpts) SetLimit(limit int) *UpdatesOpts {
	opts.Limit = &limit
	return opts
}

// SetTimeout sets the timeout and returns itself.
func (opts *UpdatesOpts) SetTimeout(timeout time.Duration) *UpdatesOpts {
	opts.Timeout = &timeout
	return opts
}

// SetAllowedUpdates sets the allowed updates and returns itself.
func (opts *UpdatesOpts) SetAllowedUpdates(allowedUpdates ...string) *UpdatesOpts {
	opts.AllowedUpdates = allowedUpdates
	return opts
}

func (opts *UpdatesOpts) body() flu.BodyWriter {
	form := flu.Form()
	if opts.Offset != nil {
		form.Add("offset", opts.Offset.queryParam())
	}
	if opts.Limit != nil {
		form.Add("limit", strconv.Itoa(*opts.Limit))
	}
	if opts.Timeout != nil {
		form.Add("timeout", strconv.Itoa(int(opts.Timeout.Seconds())))
	}
	if opts.AllowedUpdates != nil {
		form.AddAll("allowed_updates", opts.AllowedUpdates...)
	}

	return form
}

// SendOpts represents request options provided for /send* API call.
type SendOpts interface {
	body(ChatID, interface{}) flu.BodyWriter
	entityType() string
}

// BaseSendOpts is a base type used for building SendOpts.
type BaseSendOpts BaseOpts

// NewSendOpts creates an empty BaseSendOpts instance.
func NewSendOpts() BaseSendOpts {
	return BaseSendOpts{}
}

func (opts BaseSendOpts) base() BaseOpts {
	return BaseOpts(opts)
}

// ParseMode sets the parse_mode request parameter and returns itself.
// Send Markdown or HTML, if you want Telegram apps to show bold,
// italic, fixed-width text or inline URLs in your bot's message.
func (opts BaseSendOpts) ParseMode(parseMode ParseMode) BaseSendOpts {
	opts.base().Add("parse_mode", string(parseMode))
	return opts
}

// DisableNotifications sets the disable_notification request parameter and returns itself.
// Sends the message silently. Users will receive a notification with no sound.
func (opts BaseSendOpts) DisableNotification(disableNotification bool) BaseSendOpts {
	if disableNotification {
		opts.base().Add("disable_notification", "true")
	}

	return opts
}

// ReplyToMessageID sets the reply_to_message_id request parameter and returns itself.
// If the message is a reply, ID of the original message
func (opts BaseSendOpts) ReplyToMessageID(replyToMessageID ID) BaseSendOpts {
	opts.base().Add("reply_to_message_id", replyToMessageID.queryParam())
	return opts
}

// Message converts this instance to MessageOpts for /sendMessage API call.
func (opts BaseSendOpts) Message() MessageOpts {
	return MessageOpts(opts)
}

// Media converts this instance to MediaOpts for setting common /send* media API calls.
func (opts BaseSendOpts) Media() MediaOpts {
	return MediaOpts(opts)
}

// MessageOpts is used for setting options for /sendMessage API call.
// See https://core.telegram.org/bots/api#sendmessage
type MessageOpts BaseSendOpts

func (opts MessageOpts) base() BaseSendOpts {
	return BaseSendOpts(opts)
}

// DisableWebPagePreview sets the disable_web_page_preview request parameter and returns itself.
// Disables link previews for links in this message
func (opts MessageOpts) DisableWebPagePreview(disableWebPagePreview bool) MessageOpts {
	if disableWebPagePreview {
		opts.base().base().Add("disable_web_page_preview", "true")
	}

	return opts
}

// Additional interface options. A JSON-serialized object for an inline keyboard, custom reply keyboard,
// instructions to remove reply keyboard or to force a reply from the user.
func (opts MessageOpts) ReplyMarkup(markup ReplyMarkup) MessageOpts {
	markupData, _ := json.Marshal(markup)
	opts.base().base().Set("reply_markup", string(markupData))
	return opts
}

func (opts MessageOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	return flu.FormWith(opts.base().base().values()).
		Add("text", entity.(string)).
		Add("chat_id", chatID.queryParam())
}

func (opts MessageOpts) entityType() string {
	return "Message"
}

// MediaOpts is used for setting options for /send* media API call.
type MediaOpts BaseSendOpts

func (opts MediaOpts) send() BaseSendOpts {
	return BaseSendOpts(opts)
}

// Caption sets the caption request parameter and returns itself.
// Media caption (may also be used when resending media files by file_id), 0-1024 characters
func (opts MediaOpts) Caption(caption string) MediaOpts {
	if caption != "" {
		opts.send().base().Add("caption", caption)
	}

	return opts
}

// Document converts MediaOpts to PhotoOpts.
func (opts MediaOpts) Document() DocumentOpts {
	return DocumentOpts(opts)
}

// Photo converts MediaOpts to PhotoOpts.
func (opts MediaOpts) Photo() PhotoOpts {
	return PhotoOpts(opts)
}

// Video converts MediaOpts to VideoOpts.
func (opts MediaOpts) Video() VideoOpts {
	return VideoOpts(opts)
}

func (opts MediaOpts) body(chatID ChatID, entityType string, entity interface{}) flu.BodyWriter {
	opts.send().base().Add("chat_id", chatID.queryParam())
	switch entity := entity.(type) {
	case string:
		return flu.FormWith(opts.send().base().values()).
			Add(entityType, entity)

	case flu.ReadResource:
		return flu.MultipartFormWith(opts.send().base().values()).
			Resource(entityType, entity)

	default:
		panic(fmt.Errorf("unrecognized entity type: %T", entity))
	}
}

// DocumentOpts is used to pass the options to /sendDocument API call.
// See https://core.telegram.org/bots/api#senddocument
type DocumentOpts MediaOpts

func (opts DocumentOpts) media() MediaOpts {
	return MediaOpts(opts)
}

func (opts DocumentOpts) body(chatId ChatID, entity interface{}) flu.BodyWriter {
	return opts.media().body(chatId, "document", entity)
}

func (opts DocumentOpts) entityType() string {
	return "Document"
}

// PhotoOpts is used to pass the options to /sendPhoto API call.
// See https://core.telegram.org/bots/api#sendphoto
type PhotoOpts MediaOpts

func (opts PhotoOpts) media() MediaOpts {
	return MediaOpts(opts)
}

func (opts PhotoOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	return opts.media().body(chatID, "photo", entity)
}

func (opts PhotoOpts) entityType() string {
	return "Photo"
}

// VideoOpts is used to pass the options to /sendVideo API call.
// See https://core.telegram.org/bots/api#sendvideo
type VideoOpts MediaOpts

func (opts VideoOpts) media() MediaOpts {
	return MediaOpts(opts)
}

// Duration sets the duration request parameter and returns itself.
// Duration of sent video in seconds
func (opts VideoOpts) Duration(duration int) VideoOpts {
	opts.media().send().base().Add("duration", strconv.Itoa(duration))
	return opts
}

// Height sets the height request parameter and returns itself.
// Video height
func (opts VideoOpts) Height(height int) VideoOpts {
	opts.media().send().base().Add("height", strconv.Itoa(height))
	return opts
}

// Width sets the width request parameter and returns itself.
// Video width
func (opts VideoOpts) Width(width int) VideoOpts {
	opts.media().send().base().Add("width", strconv.Itoa(width))
	return opts
}

func (opts VideoOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	return opts.media().body(chatID, "video", entity)
}

func (opts VideoOpts) entityType() string {
	return "Video"
}

type AnswerCallbackQueryOpts BaseOpts

func NewAnswerCallbackQueryOpts() AnswerCallbackQueryOpts {
	return AnswerCallbackQueryOpts{}
}

func (opts AnswerCallbackQueryOpts) base() BaseOpts {
	return BaseOpts(opts)
}

func (opts AnswerCallbackQueryOpts) Text(text string) AnswerCallbackQueryOpts {
	if text != "" {
		opts.base().Set("text", text)
	}

	return opts
}

func (opts AnswerCallbackQueryOpts) ShowAlert(showAlert bool) AnswerCallbackQueryOpts {
	if showAlert {
		opts.base().Set("show_alert", "true")
	}

	return opts
}

func (opts AnswerCallbackQueryOpts) URL(url string) AnswerCallbackQueryOpts {
	if url != "" {
		opts.base().Set("url", url)
	}

	return opts
}

func (opts AnswerCallbackQueryOpts) CacheTime(cacheTime time.Duration) AnswerCallbackQueryOpts {
	if cacheTime.Seconds() > 0 {
		opts.base().Set("cache_time", fmt.Sprintf("%.0f", cacheTime.Seconds()))
	}

	return opts
}

func (opts AnswerCallbackQueryOpts) body() flu.FormBody {
	return flu.FormWith(opts.base().values())
}
