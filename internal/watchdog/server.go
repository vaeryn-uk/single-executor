package watchdog

import (
	"fmt"
	"net"
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

type Watchdog struct {
	config          Configuration
	state           state
	votes           map[Id]bool
	currentTerm     uint8
	Errors          chan error
	Info            chan []byte
	heartbeats      map[Id]bool
	electionTimer   *time.Timer
	leadershipTimer *time.Timer
	heartbeatTicker *time.Ticker
	leader          Id
	process         *os.Process
	votedFor        Id
	queue util.Queue
}

func NewWatchdog(c Configuration) *Watchdog {
	w := Watchdog{
		c,
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
	}

	return &w
}

func (w *Watchdog) Start() error {
	listener, err := net.ListenUDP("udp", w.config.listenOn)

	if err != nil {
		return err
	}

	go func() {
		for {
			data := make([]byte, 1024)

			n, addr, err := listener.ReadFrom(data)

			if err != nil {
				w.Errors <- err
			} else {
				w.handleUdpData(data[:n], addr)
			}
		}
	}()

	w.reset()

	w.queue.Start()

	return nil
}

func (w *Watchdog) resetElectionTimer() {
	if w.electionTimer != nil {
		w.electionTimer.Stop()
	}

	w.electionTimer = time.AfterFunc(w.config.RandomElectionTimeout(), w.queue.DeferredEnqueue(w.startElection))
}

func (w *Watchdog) info(detail string) {
	w.Info <- []byte(detail)
}

func (w *Watchdog) startElection() {
	w.info("Starting election")
	w.state = StateElection
	w.setTerm(w.currentTerm + 1)
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
	w.info(fmt.Sprintf("%d is becoming leader for term %d\n", w.config.id, w.currentTerm))

	w.state = StateLeading
	w.resetLeadershipTimer()
	w.resetVotes()

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
	w.votes = w.initVotes(w.leader.IsNull())
}

func (w *Watchdog) initVotes(voteForSelf bool) map[Id]bool {
	votes := make(map[Id]bool)

	for id := range w.config.peers {
		// Initialise to no votes from all peers.
		votes[id] = false
	}

	if voteForSelf {
		votes[w.config.id] = true
	}

	return votes
}

func (w *Watchdog) broadcast(t messageType) {
	for _, peer := range w.config.peers {
		w.sendUdp(peer.addr, t)
	}
}

func (w *Watchdog) error(err error) {
	w.Errors <- err
}

func (w *Watchdog) handleUdpData(data []byte, addr net.Addr) {
	err, m := messageFromBytes(data)

	w.info(fmt.Sprintf("NET: Received %d bytes (%s) from %s\n", len(data), m.ToString(), addr))

	if err != nil {
		w.error(err)
		return
	}

	if m.term < w.currentTerm {
		// Old term. Just ignore.
		return
	}

	if m.term > w.currentTerm {
		w.setTerm(m.term)
		w.reset()
	}

	switch m.mtype {
	case MessageHeartbeat:
		w.handleHeartbeat(m.id)
	case MessageVoteRequest:
		w.handleVoteRequest(m.id)
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
		w.leader = id
		w.reset()
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

func (w *Watchdog) handleVoteRequest(id Id) {
	if w.votedFor.IsNull() {
		addr, err := w.config.AddressFor(id)

		if err != nil {
			w.error(err)
			return
		}

		w.sendUdp(addr, MessageVote)
	}
}

func (w *Watchdog) startProcess() {
	// TODO: real process.
	p, err := os.StartProcess("yes", []string{}, nil)

	if err != nil {
		w.error(err)
	}

	w.process = p
}

func (w *Watchdog) stopProcess() {
	if w.process != nil {
		err := w.process.Kill()

		if err != nil {
			w.error(err)
		}
	}
}

func (w *Watchdog) heartbeat() {
	switch w.state {
	case StateFollowing:
		// If following, only send a heartbeat to the leader.
		if w.leader != NullId {
			addr, err := w.config.AddressFor(w.leader)

			if err != nil {
				w.error(err)
			} else {
				w.sendUdp(addr, MessageHeartbeat)
			}
		}
	case StateLeading:
		// If leading, broadcast a heartbeat to all followers
		// to confirm we're still active (and elections should not occur).
		w.broadcast(MessageHeartbeat)
	}
}

func (w *Watchdog) sendUdp(addr string, mtype messageType) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)

	if err != nil {
		w.error(err)
		return
	}

	if conn, err := net.DialUDP("udp", nil, udpAddr); err != nil {
		w.error(err)
	} else {
		defer conn.Close()

		msg := message{w.config.id, w.currentTerm, mtype}
		n, err := conn.Write(msg.Serialize())

		if err != nil {
			w.error(err)
		} else {
			w.info(fmt.Sprintf("NET: sent %d bytes (%s) to %s\n", n, msg.ToString(), udpAddr))
		}
	}
}

func (w *Watchdog) resetHeartbeat() {
	if w.heartbeatTicker != nil {
		w.heartbeatTicker.Stop()
	}

	// Then queue one up every interval.
	w.heartbeatTicker = time.NewTicker(w.config.heartbeatInterval)

	go func () {
		for {
			// TODO: there is no exit from this function.
			<- w.heartbeatTicker.C

			w.queue.Enqueue(w.heartbeat)
		}
	}()
}

func (w *Watchdog) resetLeaderHeartbeats() {
	w.heartbeats = w.initVotes(true)
}