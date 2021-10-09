package app

import (
	"context"
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"
)

func Run(version string, extensions ...Extension) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := Create(version, flu.DefaultClock, flu.File(os.Args[1]))
	if err != nil {
		logrus.Fatal(err)
	}

	defer flu.CloseQuietly(app)
	if err := app.ConfigureLogging(); err != nil {
		logrus.Fatal(err)
	}

	app.ApplyExtensions(extensions...)

	if err := app.Run(ctx); err != nil {
		logrus.Fatal(err)
	}

	flu.AwaitSignal()
}
