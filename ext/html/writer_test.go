package html_test

import (
	"context"
	"testing"

	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

type textOnly struct{}

func (f textOnly) Format(text string, attrs []html.Attribute) string {
	return text
}

func TestWriter_Builder(t *testing.T) {
	buf := receiver.NewBuffer()
	writer := (&tghtml.Writer{
		Out:     &output.Paged{Receiver: buf},
		Anchors: textOnly{},
	}).WithContext(output.With(context.Background(), 72, 0))

	err := writer.
		Bold("A Study in Scarlet is an 1887 detective novel by Scottish author Arthur Conan Doyle.").
		Text(" ").
		Link("Wikipedia", "https://en.wikipedia.org/wiki/A_Study_in_Scarlet").
		Flush()
	assert.Nil(t, err)

	assert.Equal(t, []string{
		`<b>A Study in Scarlet is an 1887 detective novel by Scottish author</b>`,
		`<b>Arthur Conan Doyle.</b>`,
		`<a href="https://en.wikipedia.org/wiki/A_Study_in_Scarlet">Wikipedia</a>`,
	}, buf.Pages)
}

func TestWriter_Markup(t *testing.T) {
	buf := receiver.NewBuffer()
	writer := (&tghtml.Writer{
		Out:     &output.Paged{Receiver: buf},
		Anchors: textOnly{},
	}).WithContext(output.With(context.Background(), 45, 0))

	var markup = `<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <b><a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer"><i>https:&#47;&#47;pastebin.com&#47;i32h11vd</i></a></b>`
	assert.Nil(t, writer.MarkupString(markup).Flush())
	assert.Equal(t, []string{
		"<b>Музыкальный webm mp4 тред</b>\n<i>Не</i>",
		"<i>нашел - создал</i>\nДелимся вкусами, ищем",
		"музыку, создаем, нарезаем, постим свои либо",
		"чужие музыкальные видео.\nРекомендации для",
		"самостоятельного поиска соусов: <b></b>",
		"<b><i>https://pastebin.com/i32h11vd</i></b>",
	}, buf.Pages)
}

func TestWriter_Markup_Autofix(t *testing.T) {
	buf := receiver.NewBuffer()
	writer := (&tghtml.Writer{
		Out:     &output.Paged{Receiver: buf},
		Anchors: textOnly{},
	}).WithContext(output.With(context.Background(), 45, 0))

	var markup = `<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал<br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a></b>`
	assert.Nil(t, writer.MarkupString(markup).Flush())
	assert.Equal(t, []string{
		"<b>Музыкальный webm mp4 тред</b>\n<i>Не</i>",
		"<i>нашел - создал\nДелимся вкусами, ищем</i>",
		"<i>музыку, создаем, нарезаем, постим свои</i>",
		"<i>либо чужие музыкальные видео.\nРекоменд</i>",
		"<i>ации для самостоятельного поиска</i>",
		"<i>соусов: https://pastebin.com/i32h11vd</i>",
	}, buf.Pages)
}
