package telegram

import (
	"encoding/json"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type sendable interface {
	kind() string
	body(flu.Form) (flu.BodyEncoderTo, error)
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

func (t *Text) body(body flu.Form) (flu.BodyEncoderTo, error) {
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
	Type      MediaType          `url:"-" json:"type"`
	URL       string             `url:"-" json:"-"`
	Resource  flu.ResourceReader `url:"-" json:"-"`
	Caption   string             `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode          `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m *Media) kind() string {
	return m.Type
}

func (m *Media) body(form flu.Form) (flu.BodyEncoderTo, error) {
	if m.URL != "" {
		return form.Set(m.Type, m.URL), nil
	} else if m.Resource != nil {
		return form.Multipart().Resource(m.Type, m.Resource), nil
	}

	return nil, errors.New("no URL or resource specified")
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

var ErrNoURLOrResource = errors.New("no URL or resource specified")

func (mg MediaGroup) body(form flu.Form) (flu.BodyEncoderTo, error) {
	var multipart flu.MultipartForm
	multipartInitialized := false
	media := make([]mediaJSON, len(mg))
	for i, m := range mg {
		m := mediaJSON{m, ""}
		if m.Resource != nil {
			if !multipartInitialized {
				multipart = form.Multipart()
				multipartInitialized = true
			}
			id := "media" + strconv.Itoa(i)
			multipart.Resource(id, m.Resource)
			m.MediaURL = "attach://" + id
		} else if m.URL != "" {
			m.MediaURL = m.URL
		} else {
			return nil, ErrNoURLOrResource
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
