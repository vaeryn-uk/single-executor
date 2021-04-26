package watchdog

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"net"
	"time"
)

type peerInput struct {
	Id      uint8 `yaml:"id"`
	Address string `yaml:"address"`
}

type configurationInput struct {
	MinElectionTimeout int    `yaml:"minElectionTimeout"`
	MaxElectionTimeout int    `yaml:"maxElectionTimeout"`
	NetworkInterval   string `yaml:"networkInterval"`
	Peers []peerInput `yaml:"peers"`
	ListenOn string `yaml:"listenOn"`
	Id uint8 `yaml:"id"`
}

type Peer struct {
	addr string
	id Id
}

type Configuration struct {
	id Id
	minElectionTimeout int
	maxElectionTimeout int
	networkInterval time.Duration
	peers map[Id]Peer
	listenOn *net.UDPAddr
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

	result.networkInterval, err = time.ParseDuration(raw.NetworkInterval)

	if err != nil {
		return result, fmt.Errorf("Invalid runtimeIncrement string. Must be golang duration: %s\n", err.Error())
	}

	result.id = Id(raw.Id)
	result.minElectionTimeout = raw.MinElectionTimeout
	result.maxElectionTimeout = raw.MaxElectionTimeout
	result.peers = make(map[Id]Peer)

	for _, peer := range raw.Peers {
		result.peers[Id(peer.Id)] = Peer{peer.Address, Id(peer.Id)}
	}

	result.listenOn, err = net.ResolveUDPAddr("udp", raw.ListenOn)

	if err != nil {
		return result, fmt.Errorf("Invalid UDP listen addr: %s\n", err.Error())
	}

	return result, nil
}
