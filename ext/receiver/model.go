package receiver

import (
	"context"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w-go/flu/syncf"
)

type Media struct {
	MIMEType string
	Input    flu.Input
}

type Interface interface {
	SendText(ctx context.Context, text string) error
	SendMedia(ctx context.Context, ref syncf.Future[*Media], caption string) error
}
