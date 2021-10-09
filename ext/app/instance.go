package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/app"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var GormDialects = app.GormDialects

type Instance struct {
	*app.Base
	extensions []Extension
	bot        *telegram.Bot
}

func Create(version string, clock flu.Clock, config flu.File) (*Instance, error) {
	base, err := app.New(version, clock, config, flu.YAML)
	if err != nil {
		return nil, err
	}

	return &Instance{
		Base:       base,
		extensions: make([]Extension, 0),
	}, nil
}

func (app *Instance) ApplyExtensions(extensions ...Extension) {
	app.extensions = append(app.extensions, extensions...)
}

func (app *Instance) GetBot(ctx context.Context) (*telegram.Bot, error) {
	if app.bot != nil {
		return app.bot, nil
	}

	config := new(struct{ Telegram struct{ Token string } })
	if err := app.GetConfig(config); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	app.bot = telegram.NewBot(ctx, nil, config.Telegram.Token)
	return app.bot, nil
}

func (app *Instance) Run(ctx context.Context) error {
	global := make(telegram.CommandRegistry).
		AddFunc("/start", func(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
			return cmd.Reply(ctx, client, fmt.Sprintf("hi, %d @ %d", cmd.User.ID, cmd.Chat.ID))
		})

	commands := make(Commands)
	for _, extension := range app.extensions {
		id := extension.ID()
		listener, err := extension.Apply(ctx, app)
		if err != nil {
			return errors.Wrapf(err, "apply extension %s", id)
		}

		log := logrus.WithField("extension", id)
		if listener != nil {
			local := telegram.CommandRegistryFrom(listener)
			scoped, ok := listener.(Scoped)
			if ok {
				scope := scoped.CommandScope()
				for key, listener := range local {
					commands.Add(scope, key)
					global.Add(key, scope.Wrap(listener))
					log.WithFields(logrus.Fields{
						"key":     key,
						"chatIDs": scope.ChatIDs,
						"userIDs": scope.UserIDs,
					}).Infof("registered scoped command")
				}
			} else {
				for key, listener := range global {
					commands.AddDefault(key)
					global.Add(key, listener)
					log.WithField("key", key).Infof("registered public command")
				}
			}
		}

		log.Infof("init ok")
	}

	bot, err := app.GetBot(ctx)
	if err != nil {
		return errors.Wrap(err, "get bot")
	}

	if err := commands.Set(ctx, bot); err != nil {
		return errors.Wrap(err, "set commands")
	}

	app.Manage(bot.CommandListener(global))
	return nil
}

func humanizeKey(key string) string {
	return strings.Replace(strings.Title(strings.Trim(key, "/")), "_", " ", -1)
}
