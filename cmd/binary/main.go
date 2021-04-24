package main

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"os"
	"time"
)

var instanceId uuid.UUID
var nodeId string
var duration time.Duration

func main() {
	nodeId = os.Getenv("NODE_ID")

	if len(nodeId) == 0 {
		log.Fatalln("Must provide env NODE_ID")
	}

	udpAddr := os.Getenv("CHAIN_UDP_ADDR")

	if len(udpAddr) == 0 {
		log.Fatalln("Must specify env CHAIN_UDP_ADDR")
	}

	id, err := uuid.NewUUID()

	if err != nil {
		panic(err)
	}

	instanceId = id

	log.Printf("Running binary. Instance ID: %s", instanceId.String())

	duration, err = time.ParseDuration(os.Getenv("SIGN_INTERVAL"))

	if err != nil {
		log.Fatalf("Invalid env SIGN_INTERVAL: %s\n", err.Error())
	}

	addr, err := net.ResolveUDPAddr("udp", udpAddr)

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

	timeout, err := time.ParseDuration("10ms")

	if err != nil {
		panic(err)
	}

	err = conn.SetDeadline(time.Now().Add(timeout))

	if err != nil {
		panic(err)
	}

	n, err := fmt.Fprintf(conn, "%s.%s.%s", nodeId, instanceId, signature)

	if err != nil {
		log.Printf("Could not write to UDP: %s", err.Error())
	} else {
		log.Printf("Wrote %d bytes over UDP", n)
	}
}
