package format

import "time"

type Clock interface {
	Now() time.Time
}

type ClockFunc func() time.Time

func (fun ClockFunc) Now() time.Time {
	return fun()
}
