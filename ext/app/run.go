package app

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/app"
	"github.com/sirupsen/logrus"
)

var DefaultConfigurer = app.DefaultConfigurer

func RunDefault(version, environPrefix string, extensions ...Extension) {
	configurer := app.DefaultConfigurer(environPrefix, nil, "config.file", "config.stdin")
	app, err := Create(version, flu.DefaultClock, configurer)
	if err != nil {
		logrus.Fatal(err)
	}

	defer flu.CloseQuietly(app)
	if ok, err := app.Show(); err != nil {
		logrus.Fatal(err)
	} else if ok {
		return
	}

	Run(app, extensions...)
}

func Run(app *Instance, extensions ...Extension) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.ApplyExtensions(extensions...)
	if err := app.Run(ctx); err != nil {
		logrus.Fatal(err)
	}

	flu.AwaitSignal()
}
