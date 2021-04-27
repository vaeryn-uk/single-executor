package watchdog

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"math/rand"
	"net"
	"time"
)

type peerInput struct {
	Id      uint8 `yaml:"id"`
	Address string `yaml:"address"`
}

type cmdInput struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type configurationInput struct {
	MinElectionTimeout uint        `yaml:"minElectionTimeout"`
	MaxElectionTimeout uint        `yaml:"maxElectionTimeout"`
	NetworkInterval    uint        `yaml:"networkInterval"`
	Peers              []peerInput `yaml:"peers"`
	ListenOn           string      `yaml:"listenOn"`
	Id                 uint8       `yaml:"id"`
	Command            cmdInput    `yaml:"command"`
	HeartbeatInterval  uint        `yaml:"heartbeatInterval"`
}

type Peer struct {
	addr string
	id Id
}

type Cmd struct {
	command string
	args []string
}

type Configuration struct {
	id Id
	minElectionTimeout time.Duration
	maxElectionTimeout time.Duration
	networkInterval time.Duration
	peers map[Id]Peer
	listenOn *net.UDPAddr
	command Cmd
	heartbeatInterval time.Duration
}

func (c *Configuration) RandomElectionTimeout() time.Duration {
	min := int(c.minElectionTimeout.Milliseconds())
	max := int(c.maxElectionTimeout.Milliseconds())

	ms := rand.Intn(max - min) + min

	return msIntToDuration(uint(ms))
}

func (c *Configuration) NumberOfPeers() int {
	return len(c.peers)
}

func (c *Configuration) HalfInterval() time.Duration {
	return time.Duration(c.networkInterval.Nanoseconds() / 2)
}

func (c *Configuration) AddressFor(id Id) (string, error) {
	peer, ok := c.peers[id]

	if !ok {
		return "", fmt.Errorf("Unknown peer %s\n", id)
	}

	return peer.addr, nil
}

func ParseConfiguration(in []byte) (Configuration, error) {
	var raw configurationInput
	var result Configuration

	err := yaml.Unmarshal(in, &raw)

	if err != nil {
		return result, err
	}

	fmt.Printf("%+v\n", raw)

	if raw.MinElectionTimeout >= raw.MaxElectionTimeout {
		return result, fmt.Errorf("minElectionTimeout must be less than maxElectionTimeout")
	}

	result.networkInterval = msIntToDuration(raw.NetworkInterval)

	result.id = Id(raw.Id)
	result.minElectionTimeout = msIntToDuration(raw.MinElectionTimeout)
	result.maxElectionTimeout = msIntToDuration(raw.MaxElectionTimeout)
	result.heartbeatInterval  = msIntToDuration(raw.HeartbeatInterval)

	result.peers = make(map[Id]Peer)

	for _, peer := range raw.Peers {
		result.peers[Id(peer.Id)] = Peer{peer.Address, Id(peer.Id)}
	}

	result.listenOn, err = net.ResolveUDPAddr("udp", raw.ListenOn)

	if err != nil {
		return result, fmt.Errorf("Invalid UDP listen addr: %s\n", err.Error())
	}

	result.command = Cmd{raw.Command.Name, raw.Command.Args}

	return result, nil
}

func msIntToDuration(ms uint) time.Duration {
	return time.Duration(ms * 1e6)
}
