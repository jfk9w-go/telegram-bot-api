package app

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/app"
	"github.com/sirupsen/logrus"
)

func RunDefault(version, environPrefix string, extensions ...Extension) {
	configurer := app.DefaultConfigurer(environPrefix)
	app, err := Create(version, flu.DefaultClock, configurer)
	if err != nil {
		logrus.Fatal(err)
	}

	Run(app, extensions...)
}

func Run(app *Instance, extensions ...Extension) {
	if ok, err := app.Show(); err != nil {
		logrus.Fatal(err)
	} else if ok {
		return
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
