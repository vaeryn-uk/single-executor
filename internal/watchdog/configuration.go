package watchdog

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"net"
	"sort"
	"time"
)

type nodeInput struct {
	Id       uint8  `yaml:"id"`
	UdpAddr  string `yaml:"udpAddr"`
	HttpAddr string `yaml:"httpAddr"`
}

func (n nodeInput) validate() error {
	if n.Id == 0 {
		return fmt.Errorf("A node must have an id")
	}

	if len(n.UdpAddr) == 0 {
		return fmt.Errorf("A node must have a udpAddr")
	}

	if len(n.HttpAddr) == 0 {
		return fmt.Errorf("A node must have an httpAddr")
	}

	return nil
}

type Node struct {
	udpAddr  string
	httpAddr string
	id       Id
}

func (n Node) UdpAddr() string {
	return n.udpAddr
}

func (n Node) HttpAddr() string {
	return n.httpAddr
}

func (n Node) Id() Id {
	return n.id
}

type clusterInput struct {
	Nodes []nodeInput `yaml:"nodes"`
}

type Cluster struct {
	nodes map[Id]Node
}

func (c Cluster) Nodes() []Node {
	tmp := make([]Node, len(c.nodes))
	keys := make([]int, len(c.nodes))
	index := 0

	for id := range c.nodes {
		keys[index] = int(id)
		index++
	}

	sort.Ints(keys)

	for key, id := range keys {
		tmp[key] = c.nodes[Id(id)]
	}

	return tmp
}

type cmdInput struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type configurationInput struct {
	MinElectionTimeout uint     `yaml:"minElectionTimeout"`
	MaxElectionTimeout uint     `yaml:"maxElectionTimeout"`
	NetworkInterval    uint     `yaml:"networkInterval"`
	ListenOn           string   `yaml:"listenOn"`
	Command            cmdInput `yaml:"command"`
	HeartbeatInterval  uint     `yaml:"heartbeatInterval"`
}

type Cmd struct {
	command string
	args    []string
}

type Configuration struct {
	minElectionTimeout time.Duration
	maxElectionTimeout time.Duration
	networkInterval    time.Duration
	listenOn           *net.UDPAddr
	command            Cmd
	heartbeatInterval  time.Duration
}

func (c *Configuration) HalfInterval() time.Duration {
	return time.Duration(c.networkInterval.Nanoseconds() / 2)
}

func (c *Cluster) AddressFor(id Id) (string, error) {
	peer, ok := c.nodes[id]

	if !ok {
		return "", fmt.Errorf("Unknown node %d\n", id)
	}

	return peer.udpAddr, nil
}

func ParseConfiguration(config []byte) (Configuration, error) {
	var raw configurationInput
	var parsedConfig Configuration

	err := yaml.Unmarshal(config, &raw)

	if err != nil {
		return parsedConfig, err
	}

	if raw.MinElectionTimeout >= raw.MaxElectionTimeout {
		return parsedConfig, fmt.Errorf("minElectionTimeout must be less than maxElectionTimeout")
	}

	parsedConfig.networkInterval = msIntToDuration(raw.NetworkInterval)
	parsedConfig.minElectionTimeout = msIntToDuration(raw.MinElectionTimeout)
	parsedConfig.maxElectionTimeout = msIntToDuration(raw.MaxElectionTimeout)
	parsedConfig.heartbeatInterval = msIntToDuration(raw.HeartbeatInterval)

	parsedConfig.listenOn, err = net.ResolveUDPAddr("udp", raw.ListenOn)

	if err != nil {
		return parsedConfig, fmt.Errorf("Invalid UDP listen udpAddr: %s\n", err.Error())
	}

	parsedConfig.command = Cmd{raw.Command.Name, raw.Command.Args}

	return parsedConfig, nil
}

func ParseCluster(in []byte) (Cluster, error) {
	var cluster Cluster
	var input clusterInput

	err := yaml.Unmarshal(in, &input)

	if err != nil {
		return cluster, err
	}

	cluster.nodes = make(map[Id]Node)

	for _, nodeInput := range input.Nodes {
		if err = nodeInput.validate(); err != nil {
			return cluster, err
		}

		var node Node

		node.id = Id(nodeInput.Id)
		node.udpAddr = nodeInput.UdpAddr
		node.httpAddr = nodeInput.HttpAddr

		cluster.nodes[node.id] = node
	}

	if len(cluster.nodes) == 0 {
		return cluster, fmt.Errorf("Cluster file is invalid. Must specify at least one node.\n")
	}

	return cluster, nil
}

func msIntToDuration(ms uint) time.Duration {
	return time.Duration(ms * 1e6)
}
