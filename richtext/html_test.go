package richtext

import (
	"testing"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/stretchr/testify/assert"
)

func Test_PageWriter_SingleLetter(t *testing.T) {
	p := HTML(1, 0, nil, nil)
	p.writeBreakable("hello")
	p.writeBreakable("hello")
	p.writeUnbreakable("hello")
	assert.Equal(t,
		[]string{"h", "e", "l", "l", "o", "h", "e", "l", "l", "o", "B", "R", "O", "K", "E", "N"},
		p.Flush().Text)
}

func Test_PageWriter_OnePage(t *testing.T) {
	p := HTML(10, 1, nil, nil)
	p.writeBreakable("Hello, Mark. How do you do?")
	assert.Equal(t, []string{"Hello,"}, p.Flush().Text)
}

func Test_PageWriter_ManyPages(t *testing.T) {
	p := HTML(10, -1, nil, nil)
	p.writeBreakable("Hello, Mark. How do you do?")
	assert.Equal(t, []string{"Hello,", "Mark. How", "do you do?"}, p.Flush().Text)
}

func Test_PageWriter_Emojis(t *testing.T) {
	p := HTML(3, 0, nil, nil)
	p.writeBreakable("😭👌🎉😞😘😔")
	assert.Equal(t, []string{"😭👌🎉", "😞😘😔"}, p.Flush().Text)
}

func Test_PageWriter_Unbreakable(t *testing.T) {
	p := HTML(8, 0, nil, nil)
	p.writeUnbreakable("123")
	p.writeUnbreakable("😭👌🎉😞😘😔")
	assert.Equal(t, []string{"123", "😭👌🎉😞😘😔"}, p.Flush().Text)
}

type testLinkPrinter struct{}

func (testLinkPrinter) Print(link *Link) string {
	if href, ok := link.Attr("href"); ok {
		return href
	} else {
		return ""
	}
}

func Test_PageWriter_BasicHTML(t *testing.T) {
	parts := HTML(72, 0, nil, nil).
		Tag("b").
		Text("A Study in Scarlet is an 1887 detective novel by Scottish author Arthur Conan Doyle. ").
		EndTag().
		Link("Wikipedia", "https://en.wikipedia.org/wiki/A_Study_in_Scarlet").
		Flush().Text
	sample := []string{
		`<b>A Study in Scarlet is an 1887 detective novel by Scottish author</b>`,
		`<b>Arthur Conan Doyle. </b>`,
		`<a href="https://en.wikipedia.org/wiki/A_Study_in_Scarlet">Wikipedia</a>`,
	}
	assert.Equal(t, parts, sample)

	parts = HTML(0, 0, nil, nil).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Flush().Text
	sample = []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел - создал</i>
Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.
Рекомендации для самостоятельного поиска соусов: <a href="https://pastebin.com/i32h11vd">https://pastebin.com/i32h11vd</a>`,
	}
	assert.Equal(t, parts, sample)

	parts = HTML(75, 0, nil, nil).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Flush().Text
	sample = []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел - создал</i>
Делимся вкусами,`,
		`ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.`,
		`Рекомендации для самостоятельного поиска соусов: `,
		`<a href="https://pastebin.com/i32h11vd">https://pastebin.com/i32h11vd</a>`,
	}
	assert.Equal(t, parts, sample)
}

func Test_PageWriter_LinkPrinter(t *testing.T) {
	pages := HTML(50, 0, nil, testLinkPrinter{}).
		Parse(`<strong>Музыкальный webm mp4 тред</strong><br><em>Не нашел - создал</em><br>Делимся вкусами, ищем музыку, создаем, нарезаем, постим свои либо чужие музыкальные видео.<br>Рекомендации для самостоятельного поиска соусов: <a href="https:&#47;&#47;pastebin.com&#47;i32h11vd" target="_blank" rel="nofollow noopener noreferrer">https:&#47;&#47;pastebin.com&#47;i32h11vd</a>`).
		Flush().Text
	sample := []string{
		`<b>Музыкальный webm mp4 тред</b>
<i>Не нашел -</i>`,
		`<i>создал</i>
Делимся вкусами, ищем музыку,`,
		`создаем, нарезаем, постим свои либо чужие`,
		`музыкальные видео.
Рекомендации для`,
		`самостоятельного поиска соусов: `,
		`https://pastebin.com/i32h11vd`,
	}
	assert.Equal(t, pages, sample)
}

func Test_PageWriter_LinkInTag(t *testing.T) {
	pages := HTML(telegram.MaxMessageSize, 0, nil, nil).
		Parse(
			`<i>&gt;Почему товарищ Майор ничего не делает с рабочими домами, ведь следят они ужасно, в том же вконтактике <a href="https://vk.com/pedestrian111,">https://vk.com/pedestrian111,</a> да и симпатию питать к таким сложно? Заносят? Никто не жалуется, а план легче на наркошах выполнять?
    Делает, но о подвигах все молчат с 2011 года? Я чего-то не понимаю и это все норма? </i>`).
		Flush().Text
	sample := []string{
		`<i>&gt;Почему товарищ Майор ничего не делает с рабочими домами, ведь следят они ужасно, в том же вконтактике </i><a href="https://vk.com/pedestrian111,">https://vk.com/pedestrian111,</a><i> да и симпатию питать к таким сложно? Заносят? Никто не жалуется, а план легче на наркошах выполнять?
    Делает, но о подвигах все молчат с 2011 года? Я чего-то не понимаю и это все норма? </i>`,
	}
	assert.Equal(t, pages, sample)
}
