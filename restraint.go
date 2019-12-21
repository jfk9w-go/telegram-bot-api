package telegram

import (
	"time"
)

type Restraint interface {
	Start()
	Complete()
}

type concurrencyRestraint chan struct{}

func NewConcurrencyRestraint(concurrency int) Restraint {
	if concurrency < 1 {
		return NoRestraint{}
	} else {
		r := make(concurrencyRestraint, concurrency)
		for i := 0; i < concurrency; i++ {
			r.Complete()
		}
		return r
	}
}

func (r concurrencyRestraint) Start() {
	<-r
}

var unit struct{}

func (r concurrencyRestraint) Complete() {
	r <- unit
}

type intervalRestraint struct {
	event    chan time.Time
	interval time.Duration
}

func NewIntervalRestraint(interval time.Duration) Restraint {
	if interval <= 0 {
		return NoRestraint{}
	} else {
		event := make(chan time.Time, 1)
		event <- time.Unix(0, 0)
		return intervalRestraint{event, interval}
	}
}

func (r intervalRestraint) Start() {
	prev := <-r.event
	time.Sleep(r.interval - time.Now().Sub(prev))
}

func (r intervalRestraint) Complete() {
	r.event <- time.Now()
}

type NoRestraint struct{}

func (NoRestraint) Start() {

}

func (NoRestraint) Complete() {

}
