package watchdog

import (
	"math/rand"
	"single-executor/internal/util"
	"time"
)

type timer struct {
	q util.Queue
	repeat bool
	f func()
	t *time.Timer
	d time.Duration
}

func newTimer(queue util.Queue, repeat bool, duration time.Duration, fn func()) *timer {
	t := new(timer)

	t.q = queue
	t.repeat = repeat
	t.f = fn
	t.d = duration

	t.q.Start()

	return t
}

func (t *timer) start() {
	t.stop()

	if t.repeat {
		// interval timers work on the leading edge too.
		t.q.Enqueue(t.f)
	}

	t.t = time.AfterFunc(t.d, t.q.DeferredEnqueue(func() {
		t.f()

		if t.repeat {
			t.start()
		}
	}))
}

func (t *timer) stop() {
	if t.t != nil {
		t.t.Stop()
	}
}

type timers struct {
	election        *timer
	leadershipAware *timer
	heartbeat       *timer
	leadershipGrace *timer
	leadership      *timer
	q util.Queue
}

func (t *timers) sync(fn func()) {
	t.q.Enqueue(fn)
}

func (t *timers) stopAll() {
	t.election.stop()
	t.leadershipGrace.stop()
	t.leadershipAware.stop()
	t.heartbeat.stop()
	t.leadership.stop()
}

func newTimers(c Configuration, random rand.Source, onElectionTimeout func(), onLeadershipAwareTimeout func(), onHeartBeatInterval func(), onLeadershipGraceTimeout func(), onLeadershipTimeout func()) *timers {
	queue := make(util.Queue)
	min := int(c.minElectionTimeout.Milliseconds())
	max := int(c.maxElectionTimeout.Milliseconds())

	ms := (random.Int63() % int64(max-min)) + int64(min)

	duration := msIntToDuration(uint(ms))

	return &timers{
		newTimer(queue, false, duration, onElectionTimeout),
		newTimer(queue, false, c.networkInterval, onLeadershipAwareTimeout),
		newTimer(queue, true, c.heartbeatInterval, onHeartBeatInterval),
		newTimer(queue, false, c.networkInterval, onLeadershipGraceTimeout),
		newTimer(queue, false, c.networkInterval, onLeadershipTimeout),
		queue,
	}
}

