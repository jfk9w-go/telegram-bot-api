package richtext_test

import (
	"testing"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/stretchr/testify/assert"
)

type htmlAnchorFormatFunc func(text string, attrs richtext.HTMLAttributes) string

func (f htmlAnchorFormatFunc) Format(text string, attrs richtext.HTMLAttributes) string {
	return f(text, attrs)
}

func TestHTMLWriter_Builder(t *testing.T) {
	output := richtext.NewBufferedOutput()
	writer := &richtext.HTMLWriter{
		Session: &richtext.Message{
			Transport: output,
			PageSize:  72,
		},
		TagConverter: richtext.DefaultHTMLTagConverter,
		AnchorFormat: richtext.DefaultHTMLAnchorFormat,
	}

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
	}, output.Pages)
}

func TestHTMLWriter_Markup(t *testing.T) {
	output := richtext.NewBufferedOutput()
	writer := &richtext.HTMLWriter{
		Session: &richtext.Message{
			Transport: output,
			PageSize:  45,
		},
		TagConverter: richtext.DefaultHTMLTagConverter,
		AnchorFormat: htmlAnchorFormatFunc(func(text string, _ richtext.HTMLAttributes) string { return text }),
	}

	var markup = `<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <b><a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer"><i>https:&#47;&#47;pastebin.com&#47;i32h11vd</i></a></b>`
	assert.Nil(t, writer.MarkupString(markup).Flush())
	assert.Equal(t, []string{
		"<b>Музыкальный webm mp4 тред</b>\n<i>Не</i>",
		"<i>нашел - создал</i>\nДелимся вкусами, ищем",
		"музыку, создаем, нарезаем, постим свои либо",
		"чужие музыкальные видео.\nРекомендации для",
		"самостоятельного поиска соусов: <b></b>",
		"<b><i>https://pastebin.com/i32h11vd</i></b>",
	}, output.Pages)
}

func TestHTMLWriter_Markup_Autofix(t *testing.T) {
	output := richtext.NewBufferedOutput()
	writer := &richtext.HTMLWriter{
		Session: &richtext.Message{
			Transport: output,
			PageSize:  45,
		},
		TagConverter: richtext.DefaultHTMLTagConverter,
		AnchorFormat: htmlAnchorFormatFunc(func(text string, _ richtext.HTMLAttributes) string { return text }),
	}

	var markup = `<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал<br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a></b>`
	assert.Nil(t, writer.MarkupString(markup).Flush())
	assert.Equal(t, []string{
		"<b>Музыкальный webm mp4 тред</b>\n<i>Не</i>",
		"<i>нашел - создал\nДелимся вкусами, ищем</i>",
		"<i>музыку, создаем, нарезаем, постим свои</i>",
		"<i>либо чужие музыкальные видео.\nРекоменд</i>",
		"<i>ации для самостоятельного поиска</i>",
		"<i>соусов: https://pastebin.com/i32h11vd</i>",
	}, output.Pages)
}
