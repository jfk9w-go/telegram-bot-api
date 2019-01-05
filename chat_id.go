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

// Int64Value returns the ID value as int64.
func (id ID) Int64Value() int64 {
	return int64(id)
}

// StringValue returns the ID value as string.
func (id ID) StringValue() string {
	return strconv.FormatInt(int64(id), 10)
}

// Increment increments the new ID value equal to the old one incremented by one.
func (id ID) Increment() ID {
	return ID(id.Int64Value() + 1)
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

// StringValue returns original string value.
func (username Username) StringValue() string {
	return string(username)
}

// ChatID is either an ID or channel Username in various API calls.
type ChatID interface {
	StringValue() string
}
