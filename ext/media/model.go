package media

import (
	"context"

	"github.com/jfk9w-go/flu"
)

type Value struct {
	MIMEType string
	Input    flu.Input
}

type Ref interface {
	Get(context.Context) (*Value, error)
}

type varItem struct {
	media *Value
	err   error
}

type Var chan varItem

func NewVar() Var {
	return make(Var, 1)
}

func (v Var) Set(header *Value, err error) {
	v <- varItem{header, err}
}

func (v Var) Get(ctx context.Context) (*Value, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-v:
		v <- item
		return item.media, item.err
	}
}

func Get(ctx context.Context, ref Ref) Ref {
	v := NewVar()
	new(flu.WaitGroup).Go(ctx, func(ctx context.Context) {
		value, err := ref.Get(ctx)
		v.Set(value, err)
	})

	return v
}
