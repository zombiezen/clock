package fakeclock

import (
	"testing"
	"time"
)

var baseTime = time.Unix(1136214245, 0)

func TestClock_NowAlwaysReturnsTime(t *testing.T) {
	clock := New(baseTime)

	now1 := clock.Now()
	now2 := clock.Now()

	if !now1.Equal(baseTime) {
		t.Errorf("1st clock.Now() call = %v; want %v", now1, baseTime)
	}
	if !now2.Equal(baseTime) {
		t.Errorf("2nd clock.Now() call = %v; want %v", now2, baseTime)
	}
}

func TestClock_NowAdvancesByStep(t *testing.T) {
	const step = 1 * time.Minute
	clock := NewWithStep(baseTime, step)

	now1 := clock.Now()
	now2 := clock.Now()

	if !now1.Equal(baseTime) {
		t.Errorf("1st clock.Now() call = %v; want %v", now1, baseTime)
	}
	if want := baseTime.Add(step); !now2.Equal(want) {
		t.Errorf("2nd clock.Now() call = %v; want %v", now2, want)
	}
}

func TestClock_PeekDoesNotAdvanceNow(t *testing.T) {
	const step = 1 * time.Minute
	clock := NewWithStep(baseTime, step)

	if now := clock.Peek(); !now.Equal(baseTime) {
		t.Errorf("1st clock.Peek() call = %v; want %v", now, baseTime)
	}
	if now := clock.Peek(); !now.Equal(baseTime) {
		t.Errorf("2nd clock.Peek() call = %v; want %v", now, baseTime)
	}
	if now := clock.Now(); !now.Equal(baseTime) {
		t.Errorf("clock.Now() = %v; want %v", now, baseTime)
	}
}

func TestTimer_ExactAddFiresTimer(t *testing.T) {
	const d = 50 * time.Millisecond
	endTime := baseTime.Add(d)
	clock := New(baseTime)
	timer := clock.NewTimer(d)

	clock.Add(d)
	select {
	case fireTime := <-timer.C():
		if !fireTime.Equal(endTime) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, endTime)
		}
	default:
		t.Error("<-timer.C() has nothing")
	}
}

func TestTimer_MultipleAddsOnlyFiresAtEnd(t *testing.T) {
	const (
		d0 = 24 * time.Millisecond
		d1 = 26 * time.Millisecond

		dTotal = d0 + d1
	)
	endTime := baseTime.Add(dTotal)
	clock := New(baseTime)
	timer := clock.NewTimer(dTotal)

	clock.Add(d0)
	clock.Add(d1)
	fireTime := <-timer.C()

	if !fireTime.Equal(endTime) {
		t.Errorf("<-timer.C() = %v; want %v", fireTime, endTime)
	}
}

func TestTimer_DoesNotFireAfterExpire(t *testing.T) {
	const d = 50 * time.Millisecond
	clock := New(baseTime)
	timer := clock.NewTimer(d)

	clock.Add(d)
	<-timer.C()
	clock.Add(d)
	select {
	case fireTime := <-timer.C():
		t.Errorf("<-timer.C() fired at %v after expiry", fireTime)
	default:
		// expected
	}
}

func TestTimer_StopPreventsFiring(t *testing.T) {
	const d = 50 * time.Millisecond
	clock := New(baseTime)
	timer := clock.NewTimer(d)

	timer.Stop()
	clock.Add(d)
	select {
	case fireTime := <-timer.C():
		t.Errorf("<-timer.C() fired at %v after stop", fireTime)
	default:
		// expected
	}
}

func TestTimer_ResetFiresAtDifferentTime(t *testing.T) {
	const (
		firstDelta = 50 * time.Millisecond
		d0         = 5 * time.Millisecond

		resetDelta = 1 * time.Second
		d1         = firstDelta - d0
		d2         = resetDelta - d1
	)
	clock := New(baseTime)
	timer := clock.NewTimer(firstDelta)

	clock.Add(d0)
	timer.Reset(resetDelta)
	clock.Add(d1)
	select {
	case fireTime := <-timer.C():
		t.Errorf("<-timer.C() fired at %v after reset", fireTime)
	default:
		// expected
	}
	clock.Add(d2)
	select {
	case fireTime := <-timer.C():
		if want := baseTime.Add(d0 + d1 + d2); !fireTime.Equal(want) {
			t.Errorf("<-timer.C() fired at %v; want %v", fireTime, want)
		}
	default:
		t.Error("<-timer.C() did not fire after reset")
	}
}

func TestTimer_ResetAfterExpiryFiresAgain(t *testing.T) {
	const (
		delta = 50 * time.Millisecond

		d0 = 60 * time.Millisecond
		d1 = 5 * time.Millisecond
		d2 = 40 * time.Millisecond
		d3 = 20 * time.Millisecond
	)
	clock := New(baseTime)
	timer := clock.NewTimer(delta)
	clock.Add(d0)
	<-timer.C()
	clock.Add(d1)

	timer.Reset(delta)
	clock.Add(d2)
	select {
	case fireTime := <-timer.C():
		t.Errorf("<-timer.C() fired at %v after reset", fireTime)
	default:
		// expected
	}
	clock.Add(d3)
	select {
	case fireTime := <-timer.C():
		if want := baseTime.Add(d0 + d1 + d2 + d3); !fireTime.Equal(want) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, want)
		}
	default:
		t.Error("<-timer.C() never fired after reset")
	}
}

func TestTimer_ZeroFiresImmediately(t *testing.T) {
	clock := New(baseTime)
	timer := clock.NewTimer(0)

	select {
	case fireTime := <-timer.C():
		if !fireTime.Equal(baseTime) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, baseTime)
		}
	default:
		t.Error("<-timer.C() blocked")
	}
}

func TestTimer_NegativeFiresImmediately(t *testing.T) {
	clock := New(baseTime)
	timer := clock.NewTimer(-1 * time.Second)

	select {
	case fireTime := <-timer.C():
		if !fireTime.Equal(baseTime) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, baseTime)
		}
	default:
		t.Error("<-timer.C() blocked")
	}
}

func TestTimer_ResetZeroFiresImmediately(t *testing.T) {
	const d0 = 42 * time.Millisecond
	clock := New(baseTime)
	timer := clock.NewTimer(d0)
	clock.Add(d0)
	<-timer.C()

	timer.Reset(0)

	select {
	case fireTime := <-timer.C():
		if want := baseTime.Add(d0); !fireTime.Equal(want) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, want)
		}
	default:
		t.Error("<-timer.C() blocked")
	}
}

func TestTimer_ResetNegativeFiresImmediately(t *testing.T) {
	const d0 = 42 * time.Millisecond
	clock := New(baseTime)
	timer := clock.NewTimer(d0)
	clock.Add(d0)
	<-timer.C()

	timer.Reset(-1 * time.Second)

	select {
	case fireTime := <-timer.C():
		if want := baseTime.Add(d0); !fireTime.Equal(want) {
			t.Errorf("<-timer.C() = %v; want %v", fireTime, want)
		}
	default:
		t.Error("<-timer.C() blocked")
	}
}

func TestTicker_TicksEveryPeriod(t *testing.T) {
	const (
		tick = 25 * time.Millisecond

		d0 = 27 * time.Millisecond
		d1 = 4 * time.Millisecond
		d2 = 21 * time.Millisecond
	)
	clock := New(baseTime)
	ticker := clock.NewTicker(tick)

	clock.Add(d0)
	t1 := <-ticker.C()
	if want := baseTime.Add(d0); !t1.Equal(want) {
		t.Errorf("1st <-timer.C() = %v; want %v", t1, want)
	}

	clock.Add(d1)
	clock.Add(d2)
	t2 := <-ticker.C()
	if want := baseTime.Add(d0 + d1 + d2); !t2.Equal(want) {
		t.Errorf("2nd <-timer.C() = %v; want %v", t2, want)
	}
}

func TestTicker_Stops(t *testing.T) {
	const (
		tick = 25 * time.Millisecond

		d0 = 27 * time.Millisecond
		d1 = 4 * time.Millisecond
		d2 = 21 * time.Millisecond
	)
	clock := New(baseTime)
	ticker := clock.NewTicker(tick)
	clock.Add(d0)
	<-ticker.C()

	ticker.Stop()
	clock.Add(d1)
	clock.Add(d2)
	select {
	case t2 := <-ticker.C():
		t.Errorf("<-timer.C() fired at %v after stop", t2)
	default:
		// expected
	}
}

func TestClock_NewTimerCallsWatchFunc(t *testing.T) {
	const duration = 42 * time.Second
	clock := New(baseTime)
	called := false

	clock.SetWatchFunc(func(d time.Duration) {
		if d != duration {
			t.Errorf("d = %v; want %v", d, duration)
		}
		called = true
	})
	clock.NewTimer(duration)

	if !called {
		t.Error("Clock.NewTimer did not call watch func")
	}
}

func TestTimer_ResetCallsWatchFunc(t *testing.T) {
	const (
		d0 = 1 * time.Second
		d1 = 42 * time.Second
	)
	clock := New(baseTime)
	timer := clock.NewTimer(d0)
	called := false

	clock.SetWatchFunc(func(d time.Duration) {
		if d != d1 {
			t.Errorf("d = %v; want %v", d, d1)
		}
		called = true
	})
	timer.Reset(d1)

	if !called {
		t.Error("Timer.Reset did not call watch func")
	}
}

func TestClock_NewTickerCallsWatchFunc(t *testing.T) {
	const duration = 42 * time.Second
	clock := New(baseTime)
	called := false

	clock.SetWatchFunc(func(d time.Duration) {
		if d != duration {
			t.Errorf("d = %v; want %v", d, duration)
		}
		called = true
	})
	clock.NewTicker(duration)

	if !called {
		t.Error("Clock.NewTicker did not call watch func")
	}
}

func TestTicker_FireCallsWatchFunc(t *testing.T) {
	const tick = 42 * time.Second
	clock := New(baseTime)
	clock.NewTicker(tick)
	called := false

	clock.SetWatchFunc(func(d time.Duration) {
		if d != tick {
			t.Errorf("d = %v; want %v", d, tick)
		}
		called = true
	})
	clock.Add(tick)

	if !called {
		t.Error("Ticker fire did not call watch func")
	}
}
