package telegram

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jfk9w-go/lego/json"

	"github.com/jfk9w-go/flu"
)

type ParseMode string

const (
	None     ParseMode = ""
	Markdown ParseMode = "Markdown"
	HTML     ParseMode = "HTML"

	MaxMessageSize = 4096
	MaxCaptionSize = 200
)

type BaseOpts url.Values

func (opts BaseOpts) values() url.Values {
	return url.Values(opts)
}

func (opts BaseOpts) Add(key, value string) BaseOpts {
	opts.values().Add(key, value)
	return opts
}

func (opts BaseOpts) AddAll(key string, values ...string) BaseOpts {
	for _, value := range values {
		opts.values().Add(key, value)
	}

	return opts
}

func (opts BaseOpts) Set(key, value string) BaseOpts {
	opts.values().Set(key, value)
	return opts
}

type UpdatesOpts struct {
	Offset         *ID            `json:"offset"`
	Limit          *int           `json:"limit"`
	Timeout        *json.Duration `json:"timeout"`
	AllowedUpdates []string       `json:"allowed_updates"`
}

func (opts *UpdatesOpts) SetOffset(offset ID) *UpdatesOpts {
	opts.Offset = &offset
	return opts
}

func (opts *UpdatesOpts) SetLimit(limit int) *UpdatesOpts {
	opts.Limit = &limit
	return opts
}

func (opts *UpdatesOpts) SetTimeout(timeout time.Duration) *UpdatesOpts {
	opts.Timeout = (*json.Duration)(&timeout)
	return opts
}

func (opts *UpdatesOpts) SetAllowedUpdates(allowedUpdates ...string) *UpdatesOpts {
	opts.AllowedUpdates = allowedUpdates
	return opts
}

func (opts *UpdatesOpts) body() flu.BodyWriter {
	form := flu.Form()
	if opts.Offset != nil {
		form.Add("offset", opts.Offset.StringValue())
	}
	if opts.Limit != nil {
		form.Add("limit", strconv.Itoa(*opts.Limit))
	}
	if opts.Timeout != nil {
		form.Add("timeout", strconv.Itoa(int(opts.Timeout.Value().Seconds())))
	}
	if opts.AllowedUpdates != nil {
		form.AddAll("allowed_updates", opts.AllowedUpdates...)
	}

	return form
}

type SendOpts interface {
	body(ChatID, interface{}) flu.BodyWriter
	entityType() string
}

type BaseSendOpts BaseOpts

func NewSendOpts() BaseSendOpts {
	return BaseSendOpts{}
}

func (opts BaseSendOpts) base() BaseOpts {
	return BaseOpts(opts)
}

func (opts BaseSendOpts) ParseMode(parseMode ParseMode) BaseSendOpts {
	opts.base().Add("parse_mode", string(parseMode))
	return opts
}

func (opts BaseSendOpts) DisableNotification(disableNotification bool) BaseSendOpts {
	if disableNotification {
		opts.base().Add("disable_notification", "true")
	}

	return opts
}

func (opts BaseSendOpts) ReplyToMessageId(replyToMessageID ID) BaseSendOpts {
	opts.base().Add("reply_to_message_id", replyToMessageID.StringValue())
	return opts
}

func (opts BaseSendOpts) Message() MessageOpts {
	return MessageOpts(opts)
}

func (opts BaseSendOpts) Media() MediaOpts {
	return MediaOpts(opts)
}

type MessageOpts BaseSendOpts

func (opts MessageOpts) base() BaseSendOpts {
	return BaseSendOpts(opts)
}

func (opts MessageOpts) DisableWebPagePreview(disableWebPagePreview bool) MessageOpts {
	if disableWebPagePreview {
		opts.base().base().Add("disable_web_page_preview", "true")
	}

	return opts
}

func (opts MessageOpts) body(chatID ChatID, entity interface{}) flu.BodyWriter {
	return flu.FormWith(opts.base().base().values()).
		Add("text", entity.(string)).
		Add("chat_id", chatID.StringValue())
}

func (opts MessageOpts) entityType() string {
	return "Message"
}

type MediaOpts BaseSendOpts

func (opts MediaOpts) send() BaseSendOpts {
	return BaseSendOpts(opts)
}

func (opts MediaOpts) Caption(caption string) MediaOpts {
	if caption != "" {
		opts.send().base().Add("caption", caption)
	}

	return opts
}

func (opts MediaOpts) Photo() PhotoOpts {
	return PhotoOpts(opts)
}

func (opts MediaOpts) Video() VideoOpts {
	return VideoOpts(opts)
}

func (opts MediaOpts) body(chatID ChatID, entityType string, entity interface{}) flu.BodyWriter {
	opts.send().base().Add("chat_id", chatID.StringValue())
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

type VideoOpts MediaOpts

func (opts VideoOpts) media() MediaOpts {
	return MediaOpts(opts)
}

func (opts VideoOpts) Duration(duration int) VideoOpts {
	opts.media().send().base().Add("duration", strconv.Itoa(duration))
	return opts
}

func (opts VideoOpts) Height(height int) VideoOpts {
	opts.media().send().base().Add("height", strconv.Itoa(height))
	return opts
}

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
