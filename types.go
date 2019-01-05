package telegram

// Type of chat, can be either “private”, “group”, “supergroup” or “channel”
type ChatType string

const (
	PrivateChat ChatType = "private"
	GroupChat   ChatType = "group"
	Supergroup  ChatType = "supergroup"
	Channel     ChatType = "channel"
)

type (
	// See https://core.telegram.org/bots/api#user
	User struct {
		ID        ID       `json:"id"`
		IsBot     bool     `json:"is_bot"`
		FirstName string   `json:"first_name"`
		LastName  string   `json:"last_name"`
		Username  Username `json:"username"`
	}

	// See https://core.telegram.org/bots/api#chat
	Chat struct {
		ID                          ID       `json:"id"`
		Type                        ChatType `json:"type"`
		Title                       string   `json:"title"`
		Username                    Username `json:"username"`
		FirstName                   string   `json:"first_name"`
		LastName                    string   `json:"last_name"`
		AllMembersAreAdministrators bool     `json:"all_members_are_administrators"`
	}

	// See https://core.telegram.org/bots/api#message
	Message struct {
		ID       ID              `json:"message_id"`
		From     User            `json:"from"`
		Date     int             `json:"date"`
		Chat     Chat            `json:"chat"`
		Text     string          `json:"text"`
		Entities []MessageEntity `json:"entities"`
	}

	// See https://core.telegram.org/bots/api#messageentity
	MessageEntity struct {
		Type   string `json:"type"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
		URL    string `json:"url"`
		User   *User  `json:"user"`
	}

	// See https://core.telegram.org/bots/api#update
	Update struct {
		ID                ID       `json:"update_id"`
		Message           *Message `json:"message"`
		EditedMessage     *Message `json:"edited_message"`
		ChannelPost       *Message `json:"channel_post"`
		EditedChannelPost *Message `json:"edited_message_post"`
	}

	// See https://core.telegram.org/bots/api#chatmember
	ChatMember struct {
		User   User   `json:"user"`
		Status string `json:"status"`
	}
)
