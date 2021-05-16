package watchdog

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

type Id uint8

const NullId Id = 0

func (id Id) IsNull() bool {
	return id == NullId
}

type state int

const (
	StateCreated state = iota
	StateIdle
	StateFollowing
	StateLeading
	StateElection
)

func (s state) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateIdle:
		return "idle"
	case StateElection:
		return "election"
	case StateLeading:
		return "leading"
	case StateFollowing:
		return "following"
	}

	return ""
}

type event struct {
	time time.Time
	event string
	term uint8
}

type Watchdog struct {
	// Node state.
	votes       votes
	currentTerm uint8
	state       state
	votedFor    Id
	leader      Id
	heartbeats  votes

	// Mechanics.
	process *os.Process
	timers  *timers
	canRunProcess bool

	// Network & configuration.
	id      Id
	config  Configuration
	cluster Cluster
	adapter *adapter

	// Monitoring & debug.
	Errors chan error
	Info   chan []byte
	events map[time.Time]event
}

func NewWatchdog(id Id, config Configuration, cluster Cluster) *Watchdog {
	w := Watchdog{
		id: id,
		config: config,
		cluster: cluster,
		Errors: make(chan error),
		Info: make(chan []byte),
		events: make(map[time.Time]event),
	}

	return &w
}

func (w *Watchdog) Start() error {
	w.event("start")

	w.timers = newTimers(
		w.config,
		rand.NewSource(time.Now().UnixNano()),
		w.onElectionTimeout,
		w.onLeadershipAwareTimeout,
		w.onHeartBeatInterval,
		w.onLeadershipGraceTimeout,
		w.onLeadershipTimeout,
	)

	w.votes = createVotes(w.cluster)
	w.heartbeats = createVotes(w.cluster)
	w.adapter = makeAdapter(w.cluster)

	if _, err := w.cluster.AddressFor(w.id); err != nil {
		// Throw if our ID isn't in the cluster.
		return err
	}

	err := w.adapter.listen(w.config.listenOn, w.handleMessage, w.error)

	if err != nil {
		return err
	}

	go func() {
		for {
			// Periodically check and start/stop process
			// depending on leader state.
			if w.state == StateLeading && w.canRunProcess {
				w.startProcess()
			} else {
				w.stopProcess()
			}

			time.Sleep(time.Millisecond * 100)
		}
	}()

	w.transition(StateIdle)

	return nil
}

func (w *Watchdog) onElectionTimeout() {
	w.transition(StateElection)

	w.currentTerm++
	w.votes = w.votes.vote(w.id)
	w.votedFor = w.id
	w.broadcast(MessageVoteRequest)
}

func (w *Watchdog) onLeadershipAwareTimeout() {
	w.transition(StateIdle)
}

func (w *Watchdog) onLeadershipTimeout() {
	w.transition(StateIdle)
}

func (w *Watchdog) onHeartBeatInterval() {
	switch w.state {
	case StateFollowing:
		if !w.leader.IsNull() {
			addr, err := w.cluster.AddressFor(w.leader)

			if err != nil {
				w.error(err)
			} else {
				w.sendMessage(addr, MessageHeartbeat)
			}
		}
	case StateLeading:
		// If leading, broadcast a heartbeat to all followers
		// to confirm we're still active (and elections should not occur).
		w.broadcast(MessageHeartbeat)
	}
}

func (w *Watchdog) onLeadershipGraceTimeout() {
	w.canRunProcess = true
}

func (w *Watchdog) transition(state state) {
	w.event(fmt.Sprintf("transition: %s", state.String()))

	// Reset everything.
	w.timers.stopAll()
	w.leader = NullId
	w.votes = w.votes.reset()
	w.heartbeats = w.heartbeats.reset()
	w.canRunProcess = false

	// Change state.
	w.state = state

	// Configure timers based on the state.
	switch state {
	case StateIdle:
		w.timers.election.start()
	case StateFollowing:
		w.timers.leadershipAware.start()
		w.timers.heartbeat.start()
	case StateLeading:
		w.timers.leadershipGrace.start()
		w.timers.heartbeat.start()
		w.timers.leadership.start()
		w.leader = w.id
	case StateElection:
		w.timers.election.start()
	}
}

func (w *Watchdog) broadcast(t messageType) {
	for _, node := range w.cluster.nodes {
		w.sendMessage(node.udpAddr, t)
	}
}

func (w *Watchdog) sendMessage(addr string, mtype messageType) {
	// Send this off the main thread to stop blocking if there are network issues.
	go func () {
		err, info := w.adapter.send(addr, message{w.id, w.currentTerm, mtype, w.leader})

		if err != nil {
			w.error(err)
		} else {
			w.info(info)
		}
	}()
}

func (w *Watchdog) error(err error) {
	go func() {
		w.Errors <- err
	}()
}

func (w *Watchdog) info(detail string) {
	go func() {
		w.Info <- []byte(detail)
	}()
}

func (w *Watchdog) event(name string) {
	e := event{
		time.Now(),
		name,
		w.currentTerm,
	}

	w.events[e.time] = e
}

func (w *Watchdog) handleMessage(m message) {
	if m.term < w.currentTerm {
		// Old term. Just ignore.
		return
	}

	// Do this synchronously with any other timer-based
	// triggers.
	w.timers.sync(func() {
		switch m.mtype {
		case MessageVoteRequest:
			w.handleVoteRequest(m.id, m.term)
		case MessageHeartbeat:
			w.handleHeartbeat(m.id, m.leader)
		case MessageVote:
			w.handleVote(m.id)
		}
	})
}

func (w *Watchdog) handleHeartbeat(id Id, leader Id) {
	if w.state == StateLeading && leader == w.id {
		w.info(fmt.Sprintf("Received follower heartbeat %d\n", id))

		w.heartbeats = w.heartbeats.vote(id)

		if w.heartbeats.isMajority() {
			w.timers.leadership.start()
			w.heartbeats = w.heartbeats.reset().vote(w.id)
		}
	} else if id == leader {
		w.info(fmt.Sprintf("Detected leader %d\n", id))
		w.leader = id
		w.timers.leadershipAware.start()

		if w.state != StateFollowing {
			w.transition(StateFollowing)
		}
	}
}

func (w *Watchdog) handleVote(id Id) {
	w.votes = w.votes.vote(id)

	if w.votes.isMajority() {
		// Majority reached! Let's go do leader things.
		w.transition(StateLeading)
	}
}

func (w *Watchdog) handleVoteRequest(id Id, term uint8) {
	if w.state == StateLeading || w.state == StateFollowing {
		// Nothing to do here.
		return
	}

	if term > w.currentTerm {
		w.newTerm(term)
	}

	if !w.votedFor.IsNull() {
		// We've already voted for something in this term.
		return
	}

	addr, err := w.cluster.AddressFor(id)

	if err != nil {
		w.error(err)
		return
	}

	w.event(fmt.Sprintf("voted for %d", id))

	w.sendMessage(addr, MessageVote)
	w.votedFor = id
}

func (w *Watchdog) newTerm(term uint8) {
	if term <= w.currentTerm {
		return
	}

	w.currentTerm = term
	w.votedFor = NullId
	w.leader = NullId
}

func (w *Watchdog) startProcess() {
	if w.isProcessRunning() {
		return
	}

	attr := new(os.ProcAttr)

	p, err := os.StartProcess(w.config.command.command, w.config.command.args, attr)

	if err != nil {
		w.error(err)
	}

	w.process = p
}

func (w *Watchdog) stopProcess() {
	if w.isProcessRunning() {
		go func() {
			// Need to Wait() to read exit status from the child process
			// otherwise it sits in a zombie state indefinitely.
			// Do this in a separate thread as we don't want to block the current.
			if _, err := w.process.Wait(); err != nil {
				w.error(err)
			}

			// Once it's gone, unset everything.
			w.process = nil
		}()

		if err := w.process.Kill(); err != nil {
			w.error(err)
		}
	}
}

func (w *Watchdog) isProcessRunning() bool {
	if w.process == nil {
		return false
	}

	_, err := os.FindProcess(w.process.Pid)

	return err == nil
}
