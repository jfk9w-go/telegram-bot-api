package telegram

import (
	"encoding/json"
	"fmt"
	"time"
)

type ResponseParameters struct {
	MigrateToChatID ID  `json:"migrate_to_chat_id"`
	RetryAfter      int `json:"retry_after"`
}

type Response struct {
	Ok          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      json.RawMessage     `json:"result"`
	Parameters  *ResponseParameters `json:"parameters"`
}

func (r *Response) Parse(value interface{}) error {
	if !r.Ok {
		if r.Parameters != nil && r.Parameters.RetryAfter > 0 {
			return &TooManyMessages{time.Duration(r.Parameters.RetryAfter) * time.Second}
		}

		return &Error{r.ErrorCode, r.Description}
	}

	if value == nil {
		return nil
	}

	data, err := r.Result.MarshalJSON()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

type Error struct {
	ErrorCode   int
	Description string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s", e.ErrorCode, e.Description)
}

type TooManyMessages struct {
	RetryAfter time.Duration
}

func (e *TooManyMessages) Error() string {
	return fmt.Sprintf("too many messages, retry after %.0f seconds", e.RetryAfter.Seconds())
}
