package app

import (
	"context"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Scoped interface {
	CommandScope() CommandScope
}

type CommandScope struct {
	ChatIDs, UserIDs map[telegram.ID]bool
}

func (s CommandScope) Wrap(listener telegram.CommandListener) telegram.CommandListenerFunc {
	return func(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
		if s.allow(cmd.Chat.ID, cmd.User.ID) {
			return listener.OnCommand(ctx, client, cmd)
		}

		return errors.New("forbidden")
	}
}

func (s CommandScope) allow(chatID, userID telegram.ID) bool {
	if s.UserIDs != nil {
		return userID == chatID && s.UserIDs[userID]
	}

	if s.ChatIDs != nil {
		return userID != chatID && s.ChatIDs[chatID]
	}

	return false
}

type Commands map[telegram.BotCommandScope]map[string]string

func (c Commands) Add(scope CommandScope, commands ...string) {
	if len(scope.UserIDs) > 0 {
		c.add(telegram.BotCommandScope{Type: telegram.BotCommandScopeAllPrivateChats}, commands...)
	}

	for chatID := range scope.ChatIDs {
		c.add(telegram.BotCommandScope{Type: telegram.BotCommandScopeChat, ChatID: chatID}, commands...)
	}
}

func (c Commands) AddDefault(commands ...string) {
	c.add(telegram.BotCommandScope{Type: telegram.BotCommandScopeDefault}, commands...)
}

func (c Commands) add(scope telegram.BotCommandScope, commands ...string) {
	sc, ok := c[scope]
	if !ok {
		sc = make(map[string]string)
		c[scope] = sc
	}

	sc["start"] = "Get user & chat ID"
	for _, command := range commands {
		if strings.HasPrefix(command, "/") {
			command := command[1:]
			if _, ok := sc[command]; ok {
				logrus.Fatalf("duplicate command handler for %s", command)
			}

			sc[command] = humanizeKey(command)
		}
	}
}

func (c Commands) Set(ctx context.Context, client telegram.Client) error {
	for scope, commands := range c {
		scope := scope
		botCommands := make([]telegram.BotCommand, len(commands))
		i := 0
		for command, description := range commands {
			botCommands[i] = telegram.BotCommand{
				Command:     command,
				Description: description,
			}

			i++
		}

		if err := client.DeleteMyCommands(ctx, &scope); err != nil {
			return errors.Wrap(err, "delete commands")
		}

		if err := client.SetMyCommands(ctx, &scope, botCommands); err != nil {
			return errors.Wrap(err, "set commands")
		}
	}

	return nil
}
