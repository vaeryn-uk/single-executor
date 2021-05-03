package util

// Queue ensures that functions are executed synchronously.
// This is intended to be used without timeout-based code,
// where timers execute code on different goroutines.
// By adding this functions to a queue, we ensure that
// these critical functions do not execute at the same time.
//
// These currently cannot be stopped/cleaned up and assume
// they execute until the end of the process.
type Queue chan func()

// A Queue must be started before it does anything.
// This will start the queue processing on a separate
// goroutine.
func (q *Queue) Start() {
	go func() {
		for {
			(<- *q)()
		}
	}()
}

// Queues a function for execution.
// This will execute immediately in the Queue's goroutine (assuming Start has been called)
// or later if current functions are executing/queued.
func (q *Queue) Enqueue(in func()) {
	*q <- in
}

// A convenience version of Enqueue which returns
// a function that will call Enqueue. This can be
// used when you need to pass a function to time.Timer
// or time.Ticker.
//
// The in function will be added to the queue only
// when the returned function is called.
func (q *Queue) DeferredEnqueue(in func()) func() {
	return func() {
		q.Enqueue(in)
	}
}
