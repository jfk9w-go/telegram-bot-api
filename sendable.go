package telegram

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jfk9w-go/flu"
)

type baseSendable interface {
	kind() string
	finalize(*flu.FormBody) (flu.BodyWriter, error)
}

type Sendable interface {
	baseSendable
	toSendable() Sendable
}

type Text struct {
	Text                  string    `url:"text"`
	ParseMode             ParseMode `url:"parse_mode,omitempty"`
	DisableWebPagePreview bool      `url:"disable_web_page_preview,omitempty"`
}

func (t *Text) kind() string {
	return "message"
}

func (t *Text) isMediaGroup() bool {
	return true
}

func (t *Text) finalize(body *flu.FormBody) (flu.BodyWriter, error) {
	return body, nil
}

func (t *Text) toSendable() Sendable {
	return t
}

type MediaType = string

const (
	Photo    = "photo"
	Video    = "video"
	Document = "document"
)

type Media struct {
	Type      MediaType              `url:"-" json:"type"`
	URL       string                 `url:"-" json:"-"`
	Resource  flu.FileSystemResource `url:"-" json:"-"`
	Caption   string                 `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode              `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m *Media) kind() string {
	return m.Type
}

func (m *Media) finalize(body *flu.FormBody) (flu.BodyWriter, error) {
	if m.URL != "" {
		return body.Set(m.Type, m.URL), nil
	} else if m.Resource != "" {
		return body.Multipart().Resource(m.Type, m.Resource), nil
	}

	return nil, errors.New("no URL or resource specified")
}

func (m *Media) toSendable() Sendable {
	return m
}

type wrappedGroupMedia struct {
	Media
	MediaURL string `json:"media"`
}

type MediaGroup []Media

func (mg MediaGroup) kind() string {
	return "mediaGroup"
}

func (mg MediaGroup) finalize(body *flu.FormBody) (flu.BodyWriter, error) {
	multipart := true
	media := make([]wrappedGroupMedia, len(mg))
	for i, m := range mg {
		wm := wrappedGroupMedia{m, ""}
		if m.Resource != "" {
			multipart = true
			id := "media" + strconv.Itoa(i)
			body.Multipart().Resource(id, m.Resource)
			wm.MediaURL = "attach://" + id
		} else if m.URL != "" {
			wm.MediaURL = m.URL
		} else {
			return nil, errors.New("no URL or resource specified")
		}

		media[i] = wm
	}

	bytes, err := json.Marshal(media)
	if err != nil {
		return nil, err
	}

	body.Add("media", string(bytes))
	if multipart {
		return body.Multipart(), nil
	}

	return body, nil
}
