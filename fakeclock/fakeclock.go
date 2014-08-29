/*
	Copyright 2014 Google Inc. All rights reserved.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

// Package fakeclock provides a fake implementation of clock.
package fakeclock

import (
	"sync"
	"time"

	"bitbucket.org/zombiezen/clock"
)

// Clock implements clock.Clock by maintaining its own time.
type Clock struct {
	step time.Duration

	m     sync.Mutex
	state state
	watch func(time.Duration)
}

// New returns a new fake clock initialized to time t.
func New(t time.Time) *Clock {
	return NewWithStep(t, 0)
}

// NewWithStep returns a new fake clock that increases in time with each call to Now.
// It panics if step is negative.
func NewWithStep(start time.Time, step time.Duration) *Clock {
	if step < 0 {
		panic("fakeclock: NewWithStep with negative step")
	}
	return &Clock{
		state: state{t: start},
		step:  step,
	}
}

func (clock *Clock) do(f func(*state)) {
	clock.m.Lock()
	s := &clock.state
	t, w := s.t, clock.watch
	f(s)
	var args []time.Duration
	if s.t != t {
		s.notifyTimers()
		args = s.notifyTickers()
	}
	clock.m.Unlock()

	if w != nil {
		for _, d := range args {
			w(d)
		}
	}
}

func (clock *Clock) newWatcher(d time.Duration, f func(*state)) {
	clock.m.Lock()
	w := clock.watch
	f(&clock.state)
	clock.m.Unlock()

	if w != nil {
		w(d)
	}
}

// Now returns the clock's time, and increments the time if a step was given at creation.
func (clock *Clock) Now() time.Time {
	var now time.Time
	clock.do(func(s *state) {
		now = s.t
		s.t = s.t.Add(clock.step)
	})
	return now
}

// Add adds delta to the clock's time.  It panics if delta is negative.
func (clock *Clock) Add(delta time.Duration) {
	if delta < 0 {
		panic("fakeclock: Clock.Add with negative delta")
	}
	clock.do(func(s *state) {
		s.t = s.t.Add(delta)
	})
}

// Peek returns the clock's current time without incrementing it.
func (clock *Clock) Peek() time.Time {
	var now time.Time
	clock.do(func(s *state) {
		now = s.t
	})
	return now
}

// NewTimer creates a new timer that will send the clock's time on its channel once it has advanced by d or more.
// A zero or negative duration will send immediately.
func (clock *Clock) NewTimer(d time.Duration) clock.Timer {
	var t *timer
	clock.newWatcher(d, func(s *state) {
		t = &timer{
			clock: clock,
			c:     make(chan time.Time, 1),
		}
		if t.init(s.t, d) {
			s.timers = append(s.timers, t)
		}
	})
	return t
}

// NewTicker creates a new Ticker containing a channel that will send
// the time with a period specified by the duration argument.
// d must be greater than zero; if not, NewTicker will panic.
func (clock *Clock) NewTicker(d time.Duration) clock.Ticker {
	if d <= 0 {
		panic("fakeclock: NewTicker with non-positive duration")
	}

	var t *ticker
	clock.newWatcher(d, func(s *state) {
		t = &ticker{
			clock: clock,
			c:     make(chan time.Time, 1),
			d:     d,
			next:  s.t.Add(d),
		}
		s.tickers = append(s.tickers, t)
	})
	return t
}

// SetWatchFunc sets the watch callback to f, which may be nil.
// f will be called when a timer is created, when a timer is reset, when
// a ticker is created, and after a ticker fires.
func (clock *Clock) SetWatchFunc(f func(d time.Duration)) {
	clock.m.Lock()
	clock.watch = f
	clock.m.Unlock()
}

type timer struct {
	clock *Clock
	c     chan time.Time

	time  time.Time
	fired bool
}

func (t *timer) C() <-chan time.Time {
	return t.c
}

func (t *timer) Reset(d time.Duration) bool {
	var active bool
	t.clock.newWatcher(d, func(s *state) {
		active = !t.fired
		if t.init(s.t, d) {
			s.addTimer(t)
		} else {
			s.removeTimer(t)
		}
	})
	return active
}

func (t *timer) Stop() bool {
	var active bool
	t.clock.do(func(s *state) {
		active = !t.fired
		t.fired = true
		s.removeTimer(t)
	})
	return active
}

func (t *timer) update(now time.Time) (done bool) {
	if t.fired || now.Before(t.time) {
		return false
	}
	t.fire(now)
	return true
}

func (t *timer) fire(now time.Time) {
	select {
	case t.c <- now:
	default:
	}
	t.fired = true
}

func (t *timer) init(now time.Time, d time.Duration) bool {
	if d > 0 {
		t.time = now.Add(d)
		t.fired = false
	} else {
		t.time = now
		t.fire(now)
	}
	return !t.fired
}

type ticker struct {
	clock *Clock
	c     chan time.Time

	d    time.Duration
	next time.Time
}

func (t *ticker) C() <-chan time.Time {
	return t.c
}

func (t *ticker) Stop() {
	t.clock.do(func(s *state) {
		s.removeTicker(t)
	})
}

func (t *ticker) update(now time.Time) time.Duration {
	if now.Before(t.next) {
		return 0
	}
	select {
	case t.c <- now:
	default:
	}
	for !now.Before(t.next) {
		t.next = t.next.Add(t.d)
	}
	return t.d
}

type state struct {
	t       time.Time
	timers  []*timer
	tickers []*ticker
}

func (s *state) notifyTimers() {
	nleft := 0
	for _, t := range s.timers {
		done := t.update(s.t)
		if !done {
			s.timers[nleft] = t
			nleft++
		}
	}
	s.timers = s.timers[:nleft]
}

func (s *state) notifyTickers() (watches []time.Duration) {
	watches = make([]time.Duration, 0, len(s.tickers))
	for _, t := range s.tickers {
		d := t.update(s.t)
		if d != 0 {
			watches = append(watches, d)
		}
	}
	return
}

func (s *state) addTimer(t *timer) {
	for _, u := range s.timers {
		if u == t {
			return
		}
	}
	s.timers = append(s.timers, t)
}

func (s *state) removeTimer(t *timer) {
	for i, u := range s.timers {
		if u == t {
			copy(s.timers[i:], s.timers[i+1:])
			n := len(s.timers)
			s.timers[n-1] = nil
			s.timers = s.timers[:n-1]
		}
	}
}

func (s *state) removeTicker(t *ticker) {
	for i, u := range s.tickers {
		if u == t {
			copy(s.tickers[i:], s.tickers[i+1:])
			n := len(s.tickers)
			s.tickers[n-1] = nil
			s.tickers = s.tickers[:n-1]
		}
	}
}
