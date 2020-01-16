package telegram

import (
	"encoding/json"
	"strconv"

	"github.com/jfk9w-go/flu"
)

type sendable interface {
	kind() string
	body(flu.Form) (flu.BodyWriter, error)
}

type Sendable interface {
	sendable
	self() Sendable
}

type Text struct {
	Text                  string    `url:"text"`
	ParseMode             ParseMode `url:"parse_mode,omitempty"`
	DisableWebPagePreview bool      `url:"disable_web_page_preview,omitempty"`
}

func (t *Text) kind() string {
	return "message"
}

func (t *Text) body(body flu.Form) (flu.BodyWriter, error) {
	return body, nil
}

func (t *Text) self() Sendable {
	return t
}

type MediaType = string

const (
	Photo    = "photo"
	Video    = "video"
	Document = "document"
)

type Media struct {
	Type      MediaType    `url:"-" json:"type"`
	Resource  flu.Readable `url:"-" json:"-"`
	Caption   string       `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode    `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m *Media) kind() string {
	return m.Type
}

func (m *Media) body(form flu.Form) (flu.BodyWriter, error) {
	switch r := m.Resource.(type) {
	case flu.URL:
		return form.Set(m.Type, r.URL()), nil
	default:
		return form.Multipart().File(m.Type, m.Resource), nil
	}
}

func (m *Media) self() Sendable {
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

func (mg MediaGroup) body(form flu.Form) (flu.BodyWriter, error) {
	var multipart flu.MultipartForm
	multipartInitialized := false
	media := make([]mediaJSON, len(mg))
	for i, m := range mg {
		m := mediaJSON{m, ""}
		switch r := m.Resource.(type) {
		case flu.URL:
			m.MediaURL = r.URL()
		default:
			if !multipartInitialized {
				multipart = form.Multipart()
				multipartInitialized = true
			}
			id := "media" + strconv.Itoa(i)
			multipart.File(id, m.Resource)
			m.MediaURL = "attach://" + id
		}
		media[i] = m
	}
	bytes, err := json.Marshal(media)
	if err != nil {
		return nil, err
	}
	form.Add("media", string(bytes))
	if multipartInitialized {
		return multipart, nil
	}
	return form, nil
}
