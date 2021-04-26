package watchdog

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

type Id uint8

const NullId Id = 0

func (id Id) IsNull() bool {
	return id == NullId
}

type messageType byte

const (
	MessageVote        messageType = 0x01
	MessageVoteRequest messageType = 0x02
	MessageHeartbeat   messageType = 0x03
	MessageElectionWon messageType = 0x04
)

type state int

const (
	StateFollowing state = iota
	StateLeading
	StateElection
)

type Watchdog struct {
	config      Configuration
	state       state
	votes       map[Id]bool
	currentTerm uint8
	errs        chan error
	heartbeats  map[Id]bool
	electionTimer *time.Timer
	leadershipTimer *time.Timer
	heartbeatTicker *time.Ticker
	leader string
	process *os.Process
	votedFor Id
}

type message struct {
	id    Id
	term  uint8
	mtype messageType
}

func (m message) Serialize() []byte {
	return []byte{byte(m.id), m.term, byte(m.mtype)}
}

func messageFromBytes(data []byte) (err error, m message) {
	if len(data) != 3 {
		err = fmt.Errorf("Malformed UDP message %+v", data)
	} else {
		m = message{
			Id(data[0]),
			data[1],
			messageType(data[2]),
		}
	}

	return
}

func NewWatchdog(c Configuration) Watchdog {
	return Watchdog{
		c,
		StateElection,
		make(map[Id]bool),
		0,
		make(chan error),
		make(map[Id]bool),
		nil,
		nil,
		nil,
		"",
		nil,
		NullId,
	}
}

func (w *Watchdog) Start() (error, <-chan error) {
	listener, err := net.ListenUDP("udp", w.config.listenOn)

	if err != nil {
		return err, nil
	}

	go func() {
		var data []byte

		n, _, err := listener.ReadFrom(data)

		if err != nil {
			w.errs <- err
		}

		w.handleUdpData(data[:n])
	}()

	w.reset()

	return nil, w.errs
}

func (w *Watchdog) resetElectionTimer() {
	if w.electionTimer != nil {
		w.electionTimer.Stop()
	}

	timeout := rand.Intn(w.config.maxElectionTimeout - w.config.minElectionTimeout) + w.config.minElectionTimeout

	w.electionTimer = time.AfterFunc(time.Duration(timeout * 1000), w.startElection)
}

func (w *Watchdog) startElection() {
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
	w.votes[w.config.id] = true
}

func (w *Watchdog) makeLeader() {
	w.state = StateLeading
	w.resetLeadershipTimer()
	w.resetVotes()

	// A leader will not be checking to start a new election.
	if w.electionTimer != nil {
		w.electionTimer.Stop()
	}
}

func (w *Watchdog) resetLeadershipTimer() {
	if w.leadershipTimer != nil {
		w.leadershipTimer.Stop()
	}

	w.leadershipTimer = time.NewTimer(w.config.networkInterval)

	go func() {
		<- w.leadershipTimer.C

		w.reset()
	}()
}

func (w *Watchdog) reset() {
	w.state = StateFollowing
	w.resetVotes()
	w.resetElectionTimer()
}

func (w *Watchdog) resetVotes() {
	w.votes = make(map[Id]bool)

	for id := range w.config.peers {
		// Initialise to no votes from all peers.
		w.votes[id] = false
	}
}

func (w *Watchdog) broadcast(t messageType) {
	for _, peer := range w.config.peers {
		w.sendUdp(peer.addr, t)
	}
}

func (w *Watchdog) error(err error) {
	w.errs <- err
}

func (w *Watchdog) handleUdpData(data []byte) {
	err, m := messageFromBytes(data)

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
		w.heartbeats[id] = true
	} else {
		// When not leading, a heartbeat instructs us to restart our election timer.
		w.resetElectionTimer()
	}

}

func (w *Watchdog) handleVote(id Id) {
	w.votes[id] = true

	yes, no := 0, 0

	for _, voted := range w.votes {
		if voted {
			yes++
		} else {
			no++
		}
	}

	if yes > no {
		// Majority reached! Let's go do leader things.
		w.makeLeader()
	}
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

func (w *Watchdog) handleElectionWon(id Id) {
	addr, err := w.config.AddressFor(id)

	if err != nil {
		w.error(err)
	} else {
		w.leader = addr
	}

	w.state = StateLeading

	w.heartbeatTicker = time.NewTicker(w.config.HalfInterval())

	go func () {
		<- w.heartbeatTicker.C

		w.heartbeat()
	}()
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
		if w.leader != "" {
			w.sendUdp(w.leader, MessageHeartbeat)
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
		_, err = conn.Write(message{w.config.id, w.currentTerm, mtype}.Serialize())

		if err != nil {
			w.error(err)
		}
	}
}