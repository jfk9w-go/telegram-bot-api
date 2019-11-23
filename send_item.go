package telegram

import (
	"encoding/json"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type genericSendItem interface {
	kind() string
	write(*flu.FormBody) (flu.BodyWriter, error)
}

type SendItem interface {
	genericSendItem
	self() SendItem
}

type Text struct {
	Text                  string    `url:"text"`
	ParseMode             ParseMode `url:"parse_mode,omitempty"`
	DisableWebPagePreview bool      `url:"disable_web_page_preview,omitempty"`
}

func (t *Text) kind() string {
	return "message"
}

func (t *Text) write(body *flu.FormBody) (flu.BodyWriter, error) {
	return body, nil
}

func (t *Text) self() SendItem {
	return t
}

type MediaType = string

const (
	Photo    = "photo"
	Video    = "video"
	Document = "document"
)

type Media struct {
	Type      MediaType        `url:"-" json:"type"`
	URL       string           `url:"-" json:"-"`
	Resource  flu.ReadResource `url:"-" json:"-"`
	Caption   string           `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode        `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m *Media) kind() string {
	return m.Type
}

func (m *Media) write(body *flu.FormBody) (flu.BodyWriter, error) {
	if m.URL != "" {
		return body.Set(m.Type, m.URL), nil
	} else if m.Resource != nil {
		return body.Multipart().Resource(m.Type, m.Resource), nil
	}

	return nil, errors.New("no URL or resource specified")
}

func (m *Media) self() SendItem {
	return m
}

type mediaJSON struct {
	Media
	MediaURL string `json:"media"`
}

type MediaGroup []Media

func (mg MediaGroup) kind() string {
	return "mediaGroup"
}

func (mg MediaGroup) write(body *flu.FormBody) (flu.BodyWriter, error) {
	multipart := true
	media := make([]mediaJSON, len(mg))
	for i, m := range mg {
		m := mediaJSON{m, ""}
		if m.Resource != nil {
			multipart = true
			id := "media" + strconv.Itoa(i)
			body.Multipart().Resource(id, m.Resource)
			m.MediaURL = "attach://" + id
		} else if m.URL != "" {
			m.MediaURL = m.URL
		} else {
			return nil, errors.New("no URL or resource specified")
		}

		media[i] = m
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
