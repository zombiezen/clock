// Package clock provides interfaces for obtaining the current time and
// sleeping.
package clock

import (
	"time"
)

// A type that implements Clock can get the current time and watch for
// changes in time.  A Clock is safe to use from multiple goroutines.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) Timer
	NewTicker(d time.Duration) Ticker
}

// A Timer represents a single event.
type Timer interface {
	C() <-chan time.Time
	Reset(d time.Duration) bool
	Stop() bool
}

// A Ticker holds a channel that delivers 'ticks' of a clock at intervals.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// System implements Clock by using the functions in the time package.
var System Clock = sys{}

type sys struct{}

// Now returns time.Now().
func (sys) Now() time.Time {
	return time.Now()
}

// NewTimer returns time.NewTimer(d)
func (sys) NewTimer(d time.Duration) Timer {
	return sysTimer{time.NewTimer(d)}
}

type sysTimer struct {
	*time.Timer
}

func (t sysTimer) C() <-chan time.Time {
	return t.Timer.C
}

// NewTicker returns time.NewTicker(d)
func (sys) NewTicker(d time.Duration) Ticker {
	return sysTicker{time.NewTicker(d)}
}

type sysTicker struct {
	*time.Ticker
}

func (t sysTicker) C() <-chan time.Time {
	return t.Ticker.C
}
