package tapp

import (
	"context"
	"log"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Instance struct {
	*apfel.Core
	extensions []Extension
	bot        *telegram.Bot
}

func Create(version string, clock flu.Clock) *Instance {
	return &Instance{
		Core:       apfel.New(version, clock),
		extensions: make([]Extension, 0),
	}
}

func (app *Instance) ApplyExtensions(extensions ...Extension) {
	app.extensions = append(app.extensions, extensions...)
}

func (app *Instance) GetBot(ctx context.Context) (*telegram.Bot, error) {
	if app.bot != nil {
		return app.bot, nil
	}

	config := new(struct{ Telegram struct{ Token string } })
	if err := app.GetConfig().As(config); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	app.bot = telegram.NewBot(ctx, nil, config.Telegram.Token)
	return app.bot, nil
}

func (app *Instance) Run(ctx context.Context) error {
	global := make(telegram.CommandRegistry)
	commands := make(Commands)
	for _, extension := range app.extensions {
		id := extension.ID()
		listener, err := extension.Apply(ctx, app)
		if err != nil {
			return errors.Wrapf(err, "apply extension %s", id)
		}

		if listener == nil {
			log.Printf("extension %s disabled", id)
			continue
		}

		local := telegram.CommandRegistryFrom(listener)
		scope := Public
		if scoped, ok := listener.(Scoped); ok {
			scope = scoped.CommandScope()
		}

		for key, listener := range local {
			scope.Transform(func(scope telegram.BotCommandScope) { commands.AddAll(scope, key) })
			global.Add(key, scope.Wrap(listener))
			log.Printf("registered command %s%s", key, scope.Labels())
		}

		log.Printf("extension %s init ok", id)
	}

	bot, err := app.GetBot(ctx)
	if err != nil {
		return errors.Wrap(err, "get bot")
	}

	AddDefaultStart(commands, global, app.GetVersion())
	if err := commands.Set(ctx, bot); err != nil {
		return errors.Wrap(err, "set commands")
	}

	app.Manage(bot.CommandListener(global))
	flu.AwaitSignal(ctx)

	return nil
}

func humanizeKey(key string) string {
	return strings.Replace(strings.Title(strings.Trim(key, "/")), "_", " ", -1)
}
