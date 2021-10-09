package output

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

var levels = map[logrus.Level]string{
	logrus.InfoLevel:  "ℹ️",
	logrus.WarnLevel:  "⚠️",
	logrus.ErrorLevel: "🛑",
}

type Leveled struct {
	Context context.Context
	Paged   *Paged
	Error   error
}

func (l *Leveled) Errorf(message string, args ...interface{}) {
	l.log(logrus.ErrorLevel, message, args...)
}

func (l *Leveled) Warnf(message string, args ...interface{}) {
	l.log(logrus.WarnLevel, message, args...)
}

func (l *Leveled) Infof(message string, args ...interface{}) {
	l.log(logrus.InfoLevel, message, args...)
}

func (l *Leveled) Debugf(message string, args ...interface{}) {
	l.log(logrus.DebugLevel, message, args...)
}

func (l *Leveled) Close() error {
	if l.Error != nil {
		return l.Error
	}

	return l.Paged.Flush(l.Context)
}

func (l *Leveled) log(level logrus.Level, message string, args ...interface{}) {
	if l.Error != nil {
		return
	}

	var b strings.Builder
	if message == "" {
		return
	}

	b.WriteString("\n\n")
	icon := levels[level]
	if icon != "" {
		b.WriteString(icon)
		b.WriteRune(' ')
	}

	var err error
	if len(args) > 0 {
		lastIdx := len(args) - 1
		var ok bool
		if err, ok = args[lastIdx].(error); ok {
			args = args[:lastIdx]
		}
	}

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	b.WriteString(message)
	if err != nil {
		b.WriteRune('\n')
		b.WriteString(err.Error())
	}

	if err := l.Paged.WriteUnbreakable(l.Context, b.String()); err != nil {
		l.Error = err
	}
}
