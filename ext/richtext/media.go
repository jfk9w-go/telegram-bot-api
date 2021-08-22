package richtext

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
)

type Media struct {
	MIMEType string
	Input    flu.Input
}

type MediaRef interface {
	Get(context.Context) (*Media, error)
}

type mediaVarItem struct {
	media *Media
	err   error
}

type MediaVar chan mediaVarItem

func NewMediaVar() MediaVar {
	return make(MediaVar, 1)
}

func MediaVarFrom(ctx context.Context, ref MediaRef) MediaVar {
	v := NewMediaVar()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		media, err := ref.Get(ctx)
		v.Set(media, err)
	}()

	return v
}

func (v MediaVar) Set(media *Media, err error) {
	v <- mediaVarItem{media, err}
}

func (v MediaVar) Get(ctx context.Context) (*Media, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-v:
		v <- item
		return item.media, item.err
	}
}
