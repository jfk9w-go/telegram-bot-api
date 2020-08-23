package feed

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type ID = int64

var (
	ErrNotFound        = errors.New("not found")
	ErrExists          = errors.New("exists")
	ErrForbidden       = errors.New("forbidden")
	ErrWrongVendor     = errors.New("wrong vendor")
	ErrSuspendedByUser = errors.New("suspended by user")
	ErrInvalidSubID    = errors.New("invalid sub ID")
)

type SubID struct {
	ID     string `db:"sub_id"`
	Vendor string `db:"vendor"`
	FeedID ID     `db:"feed_id"`
}

const SubIDSeparator = "+"

func (s SubID) String() string {
	return fmt.Sprintf("%v%s%s%s%s", s.FeedID, SubIDSeparator, s.Vendor, SubIDSeparator, s.ID)
}

func ParseSubID(value string) (SubID, error) {
	tokens := strings.Split(value, SubIDSeparator)
	if len(tokens) != 3 {
		return SubID{}, ErrInvalidSubID
	}
	feedID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return SubID{}, errors.Wrapf(err, "invalid string id: %s", tokens[2])
	}
	return SubID{
		ID:     tokens[2],
		Vendor: tokens[1],
		FeedID: feedID,
	}, nil
}

type Data string

var EmptyData Data = ""

func DataFrom(value interface{}) (Data, error) {
	if value == nil {
		return EmptyData, nil
	}
	buf := flu.NewBuffer()
	err := flu.EncodeTo(flu.JSON{value}, buf)
	return Data(buf.Bytes()), err
}

func (d Data) ReadTo(value interface{}) error {
	if d == EmptyData {
		return nil
	}
	return flu.DecodeFrom(flu.Bytes(d), flu.JSON{value})
}

func (d Data) String() string {
	return string(d)
}

type Sub struct {
	SubID
	Name      string     `db:"name"`
	Data      Data       `db:"data"`
	UpdatedAt *time.Time `db:"updated_at"`
}

type WriteHTML func(html *format.HTMLWriter) *format.HTMLWriter

type Update struct {
	Write WriteHTML
	Data  interface{}
	Error error
}

type Candidate struct {
	ID   string
	Name string
	Data interface{}
}

type Vendor interface {
	Parse(ctx context.Context, ref, options string) (Candidate, error)
	Load(ctx context.Context, data Data, queue chan<- Update)
}

type Store interface {
	io.Closer
	Init(ctx context.Context) ([]ID, error)
	Create(ctx context.Context, sub Sub) error
	Get(ctx context.Context, id SubID) (Sub, error)
	Advance(ctx context.Context, feedID ID) (Sub, error)
	List(ctx context.Context, feedID ID, active bool) ([]Sub, error)
	Clear(ctx context.Context, feedID ID, pattern string) (int64, error)
	Delete(ctx context.Context, id SubID) error
	Update(ctx context.Context, id SubID, value interface{}) error
}
