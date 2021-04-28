package watchdog

import (
	"fmt"
	"net"
)

type adapter struct {
	blacklist []Id
	cluster Cluster
}

func makeAdapter(cluster Cluster) *adapter {
	adapter := new(adapter)

	adapter.blacklist = make([]Id, 0)
	adapter.cluster = cluster

	return adapter
}

func (a *adapter) blacklistNode(id Id) {
	a.blacklist = append(a.blacklist, id)
}

func (a *adapter) send(addr string, m message) (error, string) {
	for _, id := range a.blacklist {
		if nodeAddr, err := a.cluster.AddressFor(id); err == nil && nodeAddr == addr {
			// This is a blacklisted address. Do not send.
			return fmt.Errorf("Ignoring request to send to blacklisted address: %s.\n", addr), ""
		}
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)

	if err != nil {
		return err, ""
	}

	if conn, err := net.DialUDP("udp", nil, udpAddr); err != nil {
		return err, ""
	} else {
		defer conn.Close()

		n, err := conn.Write(m.Serialize())

		if err != nil {
			return err, ""
		} else {
			return nil, fmt.Sprintf("NET: sent %d bytes (%s) to %s\n", n, m.String(), udpAddr)
		}
	}
}

func (a *adapter) receive(data []byte, addr net.Addr) (message, error) {
	// TODO: This should probably listen directly on the UDP connection
	// TODO: and only emit messages that are 1) valid and 2) not from an
	// TODO: address on the blacklist.
	var m message
	err, m := messageFromBytes(data)

	if err != nil {
		return m, err
	}

	for _, id := range a.blacklist {
		if m.id == id {
			return m, fmt.Errorf("NET: Ignoring %d bytes (%s) from %s as it is blacklisted\n", len(data), m.String(), addr)
		}
	}

	return m, nil
}