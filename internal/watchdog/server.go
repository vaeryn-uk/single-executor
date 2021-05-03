package watchdog

import (
	"fmt"
	"math/rand"
	"os"
	"single-executor/internal/util"
	"time"
)

type Id uint8

const NullId Id = 0

func (id Id) IsNull() bool {
	return id == NullId
}

type state int

const (
	StateFollowing state = iota
	StateLeading
	StateElection
)

func (s state) ToString() string {
	switch s {
	case StateElection:
		return "election"
	case StateLeading:
		return "leading"
	case StateFollowing:
		return "following"
	}

	return ""
}

type Watchdog struct {
	id              Id
	config          Configuration
	cluster         Cluster
	state           state
	votes           map[Id]bool
	currentTerm     uint8
	Errors          chan error
	Info            chan []byte
	heartbeats      map[Id]bool
	electionTimer   *time.Timer
	leadershipTimer       *time.Timer
	heartbeatTicker       *time.Ticker
	leader                Id
	process               *os.Process
	votedFor              Id
	queue                 util.Queue
	randomSource          rand.Source
	events                map[time.Time]string
	adapter               *adapter
	leaderAt              time.Time
	lastLeaderHeartbeatAt time.Time
}

func NewWatchdog(id Id, config Configuration, cluster Cluster) *Watchdog {
	w := Watchdog{
		id,
		config,
		cluster,
		StateElection,
		make(map[Id]bool),
		0,
		make(chan error),
		make(chan []byte),
		make(map[Id]bool),
		nil,
		nil,
		nil,
		NullId,
		nil,
		NullId,
		make(util.Queue),
		rand.NewSource(time.Now().UnixNano()),
		make(map[time.Time]string),
		makeAdapter(cluster),
		time.Now(),
		time.Unix(0, 0),
	}

	return &w
}

func (w *Watchdog) Start() error {
	w.event("start")

	if _, err := w.cluster.AddressFor(w.id); err != nil {
		// Throw if our ID isn't in the cluster.
		return err
	}

	err := w.adapter.listen(w.config.listenOn, w.handleMessage, w.error)

	if err != nil {
		return err
	}

	w.queue.Start()
	w.reset()

	go func() {
		for {
			// Periodically check and start/stop process
			// depending on leader state.
			if w.durationAsLeader() > w.config.networkInterval {
				w.startProcess()
			} else {
				w.stopProcess()
			}

			time.Sleep(time.Millisecond * 100)
		}
	}()

	return nil
}

func (w *Watchdog) durationAsLeader() time.Duration {
	if w.state != StateLeading {
		return time.Duration(0)
	}

	return time.Now().Sub(w.leaderAt)
}

func (w *Watchdog) durationSinceLastLeaderHeartbeat() time.Duration {
	return time.Now().Sub(w.lastLeaderHeartbeatAt)
}

func (w *Watchdog) resetElectionTimer() {
	if w.electionTimer != nil {
		w.electionTimer.Stop()
	}

	w.electionTimer = time.AfterFunc(w.randomElectionTimeout(), w.queue.DeferredEnqueue(w.startElection))
}

func (w *Watchdog) randomElectionTimeout() time.Duration {
	min := int(w.config.minElectionTimeout.Milliseconds())
	max := int(w.config.maxElectionTimeout.Milliseconds())

	ms := (w.randomSource.Int63() % int64(max-min)) + int64(min)

	duration := msIntToDuration(uint(ms))

	w.info(fmt.Sprintf("Random election timeout: %s\n", duration.String()))

	return duration
}

func (w *Watchdog) startElection() {
	if w.state == StateLeading {
		// A leader cannot start an election.
		return
	}

	w.event("start-election")

	w.state = StateElection
	w.setTerm(w.currentTerm + 1)
	w.votes[w.id] = true
	w.votedFor = w.id
	w.broadcast(MessageVoteRequest)

	// Start timer for another election in cases this once fails/stalls/whatever.
	w.resetElectionTimer()
}

func (w *Watchdog) setTerm(new uint8) {
	w.currentTerm = new
	w.votedFor = NullId
	w.resetVotes()
}

func (w *Watchdog) makeLeader() {
	if w.state == StateLeading {
		// If we're already leading, there's nothing to do here.
		return
	}

	w.info(fmt.Sprintf("%d is becoming leader for term %d\n", w.id, w.currentTerm))
	w.event("leadership")

	w.state = StateLeading
	w.leader = w.id

	w.resetLeadershipTimer()
	w.resetVotes()
	w.leaderAt = time.Now()

	// A leader will not be checking to start a new election.
	if w.electionTimer != nil {
		w.electionTimer.Stop()
	}

	w.resetHeartbeat()
}

func (w *Watchdog) resetLeadershipTimer() {
	// Clear any votes for the next round.
	w.resetLeaderHeartbeats()

	if w.leadershipTimer != nil {
		w.leadershipTimer.Stop()
	}

	w.leadershipTimer = time.AfterFunc(w.config.networkInterval, w.queue.DeferredEnqueue(w.reset))
}

func (w *Watchdog) reset() {
	w.state = StateFollowing
	w.leader = NullId
	w.resetVotes()
	w.resetElectionTimer()
	w.resetHeartbeat()
	w.resetLeaderHeartbeats()
}

func (w *Watchdog) resetVotes() {
	w.votes = w.initVotes()
}

func (w *Watchdog) initVotes() map[Id]bool {
	votes := make(map[Id]bool)

	for id := range w.cluster.nodes {
		// Initialise to no votes from all peers.
		votes[id] = false
	}

	return votes
}

func (w *Watchdog) broadcast(t messageType) {
	for _, node := range w.cluster.nodes {
		w.sendMessage(node.udpAddr, t)
	}
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
	w.events[time.Now()] = fmt.Sprintf("%s (term: %d)", name, w.currentTerm)
}

func (w *Watchdog) handleMessage(m message) {
	if m.term < w.currentTerm {
		// Old term. Just ignore.
		return
	}

	switch m.mtype {
	case MessageVoteRequest:
		w.handleVoteRequest(m.id, m.term)
	case MessageHeartbeat:
		w.handleHeartbeat(m.id)
	case MessageVote:
		w.handleVote(m.id)
	}
}

func (w *Watchdog) handleHeartbeat(id Id) {
	if w.state == StateLeading {
		w.info(fmt.Sprintf("Received follower heartbeat %d\n", id))
		w.heartbeats[id] = true

		if w.isMajority(w.heartbeats) {
			w.resetLeadershipTimer()
		}
	} else {
		// When not leading, a heartbeat instructs us to restart our election timer
		// (and follow a new leader if not already).
		w.info(fmt.Sprintf("Detected leader %d\n", id))
		w.reset()
		w.leader = id
		w.lastLeaderHeartbeatAt = time.Now()
	}
}

func (w *Watchdog) handleVote(id Id) {
	w.votes[id] = true

	if w.isMajority(w.votes) {
		// Majority reached! Let's go do leader things.
		w.makeLeader()
	}
}

func (w *Watchdog) isMajority(votes map[Id]bool) bool {
	yes, no := 0, 0

	for _, voted := range votes {
		if voted {
			yes++
		} else {
			no++
		}
	}

	w.info(fmt.Sprintf("Performing vote check. For: %d, against: %d", yes, no))

	return yes > no
}

func (w *Watchdog) handleVoteRequest(id Id, term uint8) {
	if w.durationSinceLastLeaderHeartbeat() <= w.config.networkInterval {
		// We've been notified of a leader too recently. Cannot participate
		// in a new election until we've not seen a leader for a while.
		return
	}

	if term > w.currentTerm {
		// A new term.
		// TODO: This nulls any previous votes. Is there a race condition here when we voted
		// TODO: for another node, which triggered its leadership. We then vote for a new
		// TODO: node, which will in the future get leadership and we have dual leader situation?
		w.setTerm(term)
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

	w.resetElectionTimer()
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

func (w *Watchdog) heartbeat() {
	switch w.state {
	case StateFollowing:
		// If following, only send a heartbeat to the leader.
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

func (w *Watchdog) resetHeartbeat() {
	if w.heartbeatTicker != nil {
		w.heartbeatTicker.Stop()
	}

	// Then queue one up every interval.
	w.heartbeatTicker = time.NewTicker(w.config.heartbeatInterval)

	go func() {
		for {
			// TODO: there is no exit from this function.
			<-w.heartbeatTicker.C

			w.queue.Enqueue(w.heartbeat)
		}
	}()
}

func (w *Watchdog) resetLeaderHeartbeats() {
	w.heartbeats = w.initVotes()
	w.heartbeats[w.id] = true
}
