package app

import (
	"context"
	"strings"

	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Scoped interface {
	CommandScope() CommandScope
}

type CommandScope struct {
	all              bool
	ChatIDs, UserIDs map[telegram.ID]bool
}

var Public = CommandScope{all: true}

func (s CommandScope) Wrap(listener telegram.CommandListener) telegram.CommandListenerFunc {
	return func(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
		if s.allow(cmd.Chat.ID, cmd.User.ID) {
			return listener.OnCommand(ctx, client, cmd)
		}

		return nil
	}
}

func (s CommandScope) allow(chatID, userID telegram.ID) bool {
	if s.all {
		return true
	}

	if s.ChatIDs != nil {
		return userID != chatID && s.ChatIDs[chatID]
	}

	if s.UserIDs != nil {
		return userID == chatID && s.UserIDs[userID]
	}

	return false
}

func (s CommandScope) Transform(body func(scope telegram.BotCommandScope)) {
	if s.all {
		body(telegram.BotCommandScope{Type: telegram.BotCommandScopeDefault})
		return
	}

	for chatID := range s.ChatIDs {
		body(telegram.BotCommandScope{Type: telegram.BotCommandScopeChat, ChatID: chatID})
	}

	if len(s.UserIDs) > 0 {
		body(telegram.BotCommandScope{Type: telegram.BotCommandScopeAllPrivateChats})
	}
}

func slice(ids map[telegram.ID]bool) []telegram.ID {
	slice := make([]telegram.ID, len(ids))
	i := 0
	for id := range ids {
		slice[i] = id
		i++
	}

	return slice
}

func (s CommandScope) Labels() metrics.Labels {
	labels := metrics.Labels{}
	if s.all {
		return labels.Add("scope.all", true)
	}

	if len(s.ChatIDs) > 0 {
		labels = labels.Add("scope.chatIDs", slice(s.ChatIDs))
	}

	if len(s.UserIDs) > 0 {
		labels = labels.Add("scope.userIDs", slice(s.UserIDs))
	}

	return labels
}

type Commands map[telegram.BotCommandScope]map[string]string

func (c Commands) DefaultStart(version string) {
	Public.Transform(func(scope telegram.BotCommandScope) {
		c.Add(scope, "/start", "Get debug info.")
	})
}

func (c Commands) AddAll(scope telegram.BotCommandScope, commands ...string) {
	all := make(map[string]string, len(commands))
	for _, command := range commands {
		all[command] = humanizeKey(command)
	}

	c.add(scope, all)
}

func (c Commands) Add(scope telegram.BotCommandScope, command, description string) {
	c.add(scope, map[string]string{command: description})
}

func (c Commands) add(scope telegram.BotCommandScope, commands map[string]string) {
	sc, ok := c[scope]
	if !ok {
		sc = make(map[string]string)
		c[scope] = sc
	}

	for command, description := range commands {
		if strings.HasPrefix(command, "/") {
			command := command[1:]
			if _, ok := sc[command]; ok {
				logrus.Fatalf("duplicate command handler for %s", command)
			}

			sc[command] = description
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
