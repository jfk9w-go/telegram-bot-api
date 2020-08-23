package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/jfk9w-go/flu"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type WriteHTMLWithChatHref func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter

type Management interface {
	CheckAccess(ctx context.Context, userID telegram.ID, chatID telegram.ID) (context.Context, error)
	NotifyAdmins(ctx context.Context, chatID telegram.ID, markup telegram.ReplyMarkup, writeHTML WriteHTMLWithChatHref) error
	ChatHref(ctx context.Context, chatID telegram.ID) string
}

type Supervisor struct {
	Client      telegram.Client
	ChatID      telegram.ID
	inviteLinks map[telegram.ID]string
	flu.RWMutex
}

func NewSupervisorManagement(client telegram.Client, chatID telegram.ID) *Supervisor {
	return &Supervisor{
		Client:      client,
		ChatID:      chatID,
		inviteLinks: make(map[telegram.ID]string),
	}
}

func (s *Supervisor) ChatHref(ctx context.Context, chatID telegram.ID) string {
	if chatID > 0 {
		return "tg://resolve?domain=" + s.Client.Username()
	} else {
		s.RLock()
		if inviteLink, ok := s.inviteLinks[chatID]; ok {
			s.RUnlock()
			return inviteLink
		}

		s.RUnlock()
		defer s.Lock().Unlock()
		chat, err := s.Client.GetChat(ctx, chatID)
		if err != nil {
			return fmt.Sprintf("getchat:%s", err)
		}

		inviteLink := chat.InviteLink
		if inviteLink == "" {
			if chat.Username != nil {
				inviteLink = "https://t.me/" + chat.Username.String()
			} else {
				inviteLink, err = s.Client.ExportChatInviteLink(ctx, chatID)
				if err != nil {
					return fmt.Sprintf("exportchatinvitelink:%s", err)
				}
			}
		}

		s.inviteLinks[chatID] = inviteLink
		return inviteLink
	}
}

func (s *Supervisor) CheckAccess(ctx context.Context, userID telegram.ID, chatID telegram.ID) (context.Context, error) {
	if userID != s.ChatID {
		return nil, ErrForbidden
	}
	return ctx, nil
}

func (s *Supervisor) NotifyAdmins(ctx context.Context, chatID telegram.ID, markup telegram.ReplyMarkup, writeHTML WriteHTMLWithChatHref) error {
	chatHref := s.ChatHref(ctx, chatID)
	transport := format.NewBufferTransport()
	if err := writeHTML(format.HTMLWithTransport(ctx, transport), chatHref).Flush(); err != nil {
		return errors.Wrap(err, "flush")
	}

	lastIdx := len(transport.Pages) - 1
	pages := transport.Pages
	ttransport := &format.TelegramTransport{
		Sender:  s.Client,
		ChatIDs: []telegram.ChatID{s.ChatID},
		Strict:  true,
	}

	for i, page := range pages {
		markup := markup
		if i != lastIdx {
			markup = nil
		}

		if err := ttransport.Text(
			format.WithParseMode(format.WithReplyMarkup(ctx, markup), telegram.HTML),
			page, true); err != nil {
			return err
		}
	}

	return nil
}

type CommandListener struct {
	Aggregator *Aggregator
	Management Management
	Aliases    map[string]telegram.ID
}

func (c *CommandListener) OnCommand(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	var fun func(context.Context, telegram.Client, telegram.Command) error
	switch cmd.Key {
	case "/sub", "/subscribe":
		fun = c.Subscribe
	case "s":
		fun = c.Suspend
	case "r":
		fun = c.Resume
	case "d":
		fun = c.Delete
	case "/clear":
		fun = c.Clear
	case "/list":
		fun = c.List
	default:
		return errors.New("invalid command")
	}

	if err := fun(ctx, client, cmd); err != nil {
		return err
	}

	if len(cmd.Key) == 1 {
		// callback query
		return cmd.Reply(ctx, client, "OK")
	}

	return nil
}

func (c *CommandListener) listenerContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}

var (
	ErrSubscribeUsage = errors.Errorf("" +
		"Invalid command syntax. Usage: /subscribe SUB [CHAT_ID] [OPTIONS]\n\n" +
		"SUB – subscription string (for example, a link).\n" +
		"CHAT_ID – target chat username or '.' to use this chat. Optional, this chat by default.\n" +
		"OPTIONS – subscription-specific options string. Optional, empty by default.")

	ErrClearUsage = errors.Errorf("" +
		"Invalid command syntax. Usage: /clear PATTERN [CHAT_ID]\n\n" +
		"PATTERN – pattern to match subscription error.\n" +
		"CHAT_ID – target chat username or '.' to use this chat.",
	)

	ErrListUsage = errors.Errorf("" +
		"Invalid command syntax. Usage: /list [CHAT_ID] [STATUS]\n\n" +
		"CHAT_ID – target chat username or '.' to use this chat. Optional, this chat by default.\n" +
		"STATUS – status subscriptions to list for, 's' for suspended. Optional, active by default.")
)

func (c *CommandListener) resolveChatID(ctx context.Context, client telegram.Client, cmd telegram.Command, argumentIndex int) (context.Context, telegram.ID, error) {
	chatID := cmd.Chat.ID
	if len(cmd.Args) > argumentIndex && cmd.Args[argumentIndex] != "." {
		if id, ok := c.Aliases[cmd.Args[argumentIndex]]; ok {
			chatID = id
		} else {
			chat, err := client.GetChat(ctx, telegram.Username(cmd.Args[1]))
			if err != nil {
				return nil, 0, errors.Wrap(err, "get chat")
			}

			chatID = chat.ID
		}
	}

	ctx, err := c.Management.CheckAccess(ctx, cmd.User.ID, chatID)
	if err != nil {
		return nil, 0, err
	}

	return ctx, chatID, nil
}

func (c *CommandListener) parseSubID(ctx context.Context, cmd telegram.Command, argumentIndex int) (context.Context, SubID, error) {
	subID, err := ParseSubID(cmd.Args[argumentIndex])
	if err != nil {
		return nil, SubID{}, errors.Wrap(err, "parse subID")
	}

	ctx, err = c.Management.CheckAccess(ctx, cmd.User.ID, telegram.ID(subID.FeedID))
	if err != nil {
		return nil, SubID{}, err
	}

	return ctx, subID, nil
}

func (c *CommandListener) Subscribe(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	if len(cmd.Args) == 0 {
		return ErrSubscribeUsage
	}

	ref := cmd.Args[0]
	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}

	options := ""
	if len(cmd.Args) > 2 {
		options = cmd.Args[2]
	}

	sub, err := c.Aggregator.Subscribe(ctx, ID(chatID), ref, options)
	if err != nil {
		return err
	}

	go c.OnSubscribe(sub)
	return nil
}

func (c *CommandListener) OnSubscribe(sub Sub) {
	ctx, cancel := c.listenerContext()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		telegram.InlineKeyboard([][3]string{
			{"Suspend", "s", sub.SubID.String()},
		}),
		func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatHref).
				Text(" 🔥")
		})
}

func (c *CommandListener) Suspend(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}

	sub, err := c.Aggregator.Suspend(ctx, subID, ErrSuspendedByUser)
	if err != nil {
		return err
	}

	go c.OnSuspend(sub, ErrSuspendedByUser)
	return nil
}

func (c *CommandListener) OnSuspend(sub Sub, err error) {
	ctx, cancel := c.listenerContext()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		// by column
		telegram.InlineKeyboard([][3]string{
			{"Resume", "r", sub.SubID.String()},
			{"Delete", "d", sub.SubID.String()},
		}),
		func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatHref).
				Text(" 🛑\n" + err.Error())
		})
}

func (c *CommandListener) Resume(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}

	sub, err := c.Aggregator.Resume(ctx, subID)
	if err != nil {
		return err
	}

	go c.OnResume(sub)
	return nil
}

func (c *CommandListener) OnResume(sub Sub) {
	ctx, cancel := c.listenerContext()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		telegram.InlineKeyboard([][3]string{
			{"Suspend", "s", sub.SubID.String()},
		}),
		func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatHref).
				Text(" 🔥")
		})
}

func (c *CommandListener) Delete(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}

	sub, err := c.Aggregator.Delete(ctx, subID)
	if err != nil {
		return err
	}

	go c.OnDelete(sub)
	return nil
}

func (c *CommandListener) OnDelete(sub Sub) {
	ctx, cancel := c.listenerContext()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID), nil,
		func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatHref).
				Text(" 🗑")
		})
}

func (c *CommandListener) Clear(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	if len(cmd.Args) != 2 {
		return ErrClearUsage
	}

	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}

	pattern := cmd.Args[0]
	count, err := c.Aggregator.Clear(ctx, ID(chatID), pattern)
	if err != nil {
		return err
	}

	go c.OnClear(ID(chatID), pattern, count)
	return nil
}

func (c *CommandListener) OnClear(feedID ID, pattern string, count int64) {
	ctx, cancel := c.listenerContext()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(feedID), nil,
		func(html *format.HTMLWriter, chatHref string) *format.HTMLWriter {
			return html.Text(fmt.Sprintf("%d subs @ ", count)).
				Link("chat", chatHref).
				Text(" 🗑 (" + pattern + ")")
		})
}

func (c *CommandListener) List(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 0)
	if err != nil {
		return err
	}

	active := len(cmd.Args) <= 1 || cmd.Args[1] != "s"
	subs, err := c.Aggregator.List(ctx, ID(chatID), active)
	if err != nil {
		return err
	}

	status, changeCmd := "🔥", "s"
	if !active {
		status, changeCmd = "🛑", "r"
	}

	// by row
	keyboard := make([][][3]string, len(subs))
	for i, sub := range subs {
		keyboard[i] = [][3]string{{sub.Name, changeCmd, sub.SubID.String()}}
	}

	_, err = client.Send(ctx, chatID,
		telegram.Text{
			ParseMode: telegram.HTML,
			Text: fmt.Sprintf(
				"%d subs @ %s %s",
				len(subs),
				format.HTMLAnchor("chat", c.Management.ChatHref(ctx, chatID)),
				status),
			DisableWebPagePreview: true},
		&telegram.SendOptions{
			ReplyMarkup: telegram.InlineKeyboard(keyboard...)})
	return err
}
