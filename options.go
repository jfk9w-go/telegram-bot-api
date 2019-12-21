package telegram

import (
	"github.com/jfk9w-go/flu"
)

type AnswerCallbackQueryOptions struct {
	Text      string `url:"text,omitempty"`
	ShowAlert bool   `url:"show_alert,omitempty"`
	URL       string `url:"url,omitempty"`
	CacheTime int    `url:"cache_time,omitempty"`
}

func (o *AnswerCallbackQueryOptions) body(id string) flu.BodyWriter {
	return flu.FormValue(o, true).Add("callback_query_id", id)
}
