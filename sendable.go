package telegram

import (
	"encoding/json"
	"strconv"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w-go/flu"
)

type sendable interface {
	kind() string
	body(fluhttp.Form) (flu.EncoderTo, error)
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

func (t Text) kind() string {
	return "message"
}

func (t Text) body(form fluhttp.Form) (flu.EncoderTo, error) {
	return form, nil
}

func (t Text) self() Sendable {
	return t
}

type MediaType string

const (
	Photo     MediaType = "photo"
	Animation MediaType = "animation"
	Video     MediaType = "video"
	Document  MediaType = "document"
	Audio     MediaType = "audio"
	Sticker   MediaType = "sticker"
	Voice     MediaType = "voice"
)

func (mt MediaType) RemoteMaxSize() int64 {
	if mt == Photo {
		return 5 << 20
	} else {
		return 20 << 20
	}
}

func (mt MediaType) AttachMaxSize() int64 {
	if mt == Photo {
		return 10 << 20
	} else {
		return 50 << 20
	}
}

var (
	DefaultMediaType   = Document
	MIMEType2MediaType = map[string]MediaType{
		"image/jpeg":               Photo,
		"image/png":                Photo,
		"image/bmp":                Photo,
		"image/gif":                Animation,
		"video/mp4":                Video,
		"application/pdf":          Document,
		"application/octet-stream": Document,
		"audio/mpeg":               Audio,
		"audio/ogg":                Voice,
		"image/webp":               Sticker,
	}
)

func MediaTypeByMIMEType(mimeType string) MediaType {
	if mediaType, ok := MIMEType2MediaType[mimeType]; ok {
		return mediaType
	} else {
		return DefaultMediaType
	}
}

type Media struct {
	Type      MediaType `url:"-" json:"type"`
	Input     flu.Input `url:"-" json:"-"`
	Caption   string    `url:"caption,omitempty" json:"caption,omitempty"`
	ParseMode ParseMode `url:"parse_mode,omitempty" json:"parse_mode,omitempty"`
}

func (m Media) kind() string {
	return string(m.Type)
}

func (m Media) body(form fluhttp.Form) (flu.EncoderTo, error) {
	switch r := m.Input.(type) {
	case flu.URL:
		return form.Set(string(m.Type), r.URL()), nil
	default:
		return form.Multipart().File(string(m.Type), m.Input), nil
	}
}

func (m Media) self() Sendable {
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

func (mg MediaGroup) body(form fluhttp.Form) (flu.EncoderTo, error) {
	var multipart fluhttp.MultipartForm
	multipartInitialized := false
	media := make([]mediaJSON, len(mg))
	for i, m := range mg {
		m := mediaJSON{m, ""}
		switch r := m.Input.(type) {
		case flu.URL:
			m.MediaURL = r.URL()
		default:
			if !multipartInitialized {
				multipart = form.Multipart()
				multipartInitialized = true
			}
			id := "media" + strconv.Itoa(i)
			multipart = multipart.File(id, m.Input)
			m.MediaURL = "attach://" + id
		}
		media[i] = m
	}
	bytes, err := json.Marshal(media)
	if err != nil {
		return nil, err
	}
	form = form.Add("media", string(bytes))
	if multipartInitialized {
		return multipart, nil
	}
	return form, nil
}
