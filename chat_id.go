package telegram

import (
	"errors"
	"strconv"
	"strings"
)

// ID is an entity identifier (chat, message, user, etc.)
type ID int64

// ParseID tries to parse a value as ID.
func ParseID(value string) (ID, error) {
	var id, err = strconv.ParseInt(value, 10, 64)
	return ID(id), err
}

// MustParseID does what ParseID does, except on error it panics.
func MustParseID(value string) ID {
	var id, err = ParseID(value)
	if err != nil {
		panic(err)
	}

	return id
}

func (id ID) queryParam() string {
	return id.String()
}

// Increment increments the new ID value equal to the old one incremented by one.
func (id ID) Increment() ID {
	return ID(int64(id) + 1)
}

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Username represents a Telegram username (@userxxx...).
type Username string

// ParseUsername checks if a string value conforms to the username format.
func ParseUsername(str string) (Username, error) {
	if strings.HasPrefix(str, "@") {
		return Username(str), nil
	}

	return "", errors.New("username must begin with a '@'")
}

func (username Username) queryParam() string {
	return username.String()
}

func (username Username) String() string {
	return string(username)
}

// ChatID is either an ID or channel Username in various API calls.
type ChatID interface {
	queryParam() string
}
