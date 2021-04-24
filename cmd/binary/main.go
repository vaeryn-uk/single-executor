package main

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"os"
	"time"
)

var instanceId uuid.UUID
var nodeId string

func main() {
	nodeId = os.Getenv("NODE_ID")

	if len(nodeId) == 0 {
		log.Fatalln("Must provide a node ID")
	}

	id, err := uuid.NewUUID()

	if err != nil {
		panic(err)
	}

	instanceId = id

	log.Printf("Running binary. Instance ID: %s", instanceId.String())

	duration, err := time.ParseDuration("300ms")

	if err != nil {
		panic(err)
	}

	addr, err := net.ResolveUDPAddr("udp", "localhost:6111")

	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		panic(err)
	}

	for {
		<- time.Tick(duration)

		doThing(conn)
	}
}

func doThing(conn *net.UDPConn) {
	signature, err := uuid.NewUUID()

	if err != nil {
		log.Printf("UUID could not be generated: %s", err.Error())
		return
	}

	n, err := conn.Write(bytes.NewBufferString(fmt.Sprintf("%s.%s.%s", nodeId, instanceId, signature)).Bytes())

	if err != nil {
		log.Printf("Could not write to UDP %s", err.Error())
	} else {
		log.Printf("Wrote %d bytes over UDP", n)
	}
}
