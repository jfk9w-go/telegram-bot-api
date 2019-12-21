package telegram

import (
	"math"
	"time"
)

type Restraint struct {
	events   chan time.Time
	interval time.Duration
}

func NewConcurrencyRestraint(concurrency int) Restraint {
	if concurrency < 1 {
		concurrency = math.MaxInt32
	}
	event := make(chan time.Time, concurrency)
	moment := time.Unix(0, 0)
	for i := 0; i < concurrency; i++ {
		event <- moment
	}
	return Restraint{event, 0}
}

func NewIntervalRestraint(interval time.Duration) Restraint {
	events := make(chan time.Time, 1)
	events <- time.Unix(0, 0)
	return Restraint{events, interval}
}

func (fc Restraint) start() {
	prev := <-fc.events
	time.Sleep(fc.interval - time.Now().Sub(prev))
}

func (fc Restraint) complete() {
	fc.events <- time.Now()
}
