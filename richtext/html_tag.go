package richtext

import "golang.org/x/net/html"

type htmlTag struct {
	name  string
	depth int
}

func (t *htmlTag) start() string {
	if t == nil {
		return ""
	} else {
		return "<" + t.name + ">"
	}
}

func (t *htmlTag) startLen() int {
	if t == nil {
		return 0
	} else {
		return 2 + len(t.name)
	}
}

func (t *htmlTag) end() string {
	if t == nil {
		return ""
	} else {
		return "</" + t.name + ">"
	}
}

func (t *htmlTag) endLen() int {
	if t == nil {
		return 0
	} else {
		return 3 + len(t.name)
	}
}

type HTMLTags interface {
	Get(string, []html.Attribute) (string, bool)
}

type defaultSupportedTags map[string]string

func (d defaultSupportedTags) Get(tag string, attrs []html.Attribute) (string, bool) {
	tag, ok := d[tag]
	return tag, ok
}

var DefaultSupportedTags HTMLTags = defaultSupportedTags{
	"strong": "b",
	"b":      "b",
	"italic": "i",
	"em":     "i",
	"i":      "i",
	"code":   "code",
	"pre":    "pre",
}
