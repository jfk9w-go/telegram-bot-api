package feed

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jfk9w-go/flu"
)

type SubID = int64

type ID struct {
	ID    string `db:"id"`
	Type  string `db:"type"`
	SubID SubID  `db:"sub_id"`
}

type Data string

const ZeroData Data = ""

func DataFrom(value interface{}) (Data, error) {
	buf := flu.NewBuffer()
	err := flu.EncodeTo(flu.JSON{value}, buf)
	return Data(buf.Bytes()), err
}

func (d Data) ReadTo(value interface{}) error {
	return flu.DecodeFrom(flu.Bytes(d), flu.JSON{value})
}

func (d Data) String() string {
	return string(d)
}

type Feed struct {
	ID
	Name      string     `db:"name"`
	Data      Data       `db:"data"`
	UpdatedAt *time.Time `db:"updated_at"`
}

type State struct {
	Data  Data
	Error error
}

var (
	ErrNotFound  = errors.New("not found")
	ErrExists    = errors.New("exists")
	ErrForbidden = errors.New("forbidden")
)

type Store interface {
	io.Closer
	Init(ctx context.Context) ([]SubID, error)
	Create(ctx context.Context, feed Feed) error
	Get(ctx context.Context, id ID) (Feed, error)
	Advance(ctx context.Context, subID SubID) (Feed, error)
	List(ctx context.Context, subID SubID, active bool) ([]Feed, error)
	Clear(ctx context.Context, subID SubID, pattern string) (int64, error)
	Delete(ctx context.Context, id ID) error
	Update(ctx context.Context, id ID, state State) error
}
