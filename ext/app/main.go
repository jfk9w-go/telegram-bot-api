package app

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/app"
	"github.com/sirupsen/logrus"
)

func Run(version, environPrefix string, extensions ...Extension) {
	config, err := app.DefaultConfig(environPrefix, flu.YAML).Collect()
	if err != nil {
		logrus.Fatal(err)
	}

	app, err := Create(version, flu.DefaultClock, config)
	if err != nil {
		logrus.Fatal(err)
	}

	defer flu.CloseQuietly(app)
	if err := app.ConfigureLogging(); err != nil {
		logrus.Fatal(err)
	}

	app.ApplyExtensions(extensions...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx); err != nil {
		logrus.Fatal(err)
	}

	flu.AwaitSignal()
}
