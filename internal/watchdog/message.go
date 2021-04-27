package watchdog

import "fmt"

type messageType byte

const (
	MessageVote        messageType = 0x01
	MessageVoteRequest messageType = 0x02
	MessageHeartbeat   messageType = 0x03
)

func (t messageType) ToString() string {
	switch t {
	case MessageVote:
		return "vote-for"
	case MessageVoteRequest:
		return "vote-for-me"
	case MessageHeartbeat:
		return "heartbeat"
	}

	return ""
}

type message struct {
	id    Id
	term  uint8
	mtype messageType
}

func (m message) Serialize() []byte {
	return []byte{byte(m.id), m.term, byte(m.mtype)}
}

func (m message) ToString() string {
	return fmt.Sprintf("source: %d, term: %d, type: %s", m.id, m.term, m.mtype.ToString())
}

func messageFromBytes(data []byte) (err error, m message) {
	if len(data) != 3 {
		err = fmt.Errorf("Malformed UDP message %x\n", data)
	} else {
		m = message{
			Id(data[0]),
			data[1],
			messageType(data[2]),
		}
	}

	return
}