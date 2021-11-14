// tapp is designed for
package tapp

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	"gorm.io/gorm"
)

type Application interface {
	flu.Clock
	GetConfig() apfel.Config
	GetMetricsRegistry(ctx context.Context) (me3x.Registry, error)
	GetDatabase(driver, conn string) (*gorm.DB, error)
	GetBot(ctx context.Context) (*telegram.Bot, error)
	Manage(service interface{})
}

type Extension interface {
	ID() string
	Apply(ctx context.Context, app Application) (interface{}, error)
}
