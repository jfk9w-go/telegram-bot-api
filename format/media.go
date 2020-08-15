package format

import (
	"context"

	"github.com/jfk9w-go/flu"
)

type Media struct {
	MIMEType string
	Input    flu.Input
}

type MediaRef interface {
	URL() string
	Get(context.Context) (*Media, error)
}

type mediaVarItem struct {
	media *Media
	err   error
}

type MediaVar struct {
	url string
	c   chan mediaVarItem
}

func NewMediaVar(url string) *MediaVar {
	v := new(MediaVar)
	v.url = url
	v.c = make(chan mediaVarItem, 1)
	return v
}

func (v *MediaVar) Set(media *Media, err error) *MediaVar {
	v.c <- mediaVarItem{media, err}
	return v
}

func (v *MediaVar) URL() string {
	return v.url
}

func (v *MediaVar) Get(ctx context.Context) (*Media, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-v.c:
		v.c <- item
		return item.media, item.err
	}
}
