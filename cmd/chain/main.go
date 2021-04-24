package main

import (
	"github.com/google/uuid"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type signature struct {
	instanceId uuid.UUID
	nodeId string
	signature uuid.UUID
	datetime time.Time
}

var port = 6111

func main() {
	signatures := make([]signature, 0)

	addr := "localhost:" + strconv.Itoa(port)

	pc, err := net.ListenPacket("udp", addr)

	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)

	log.Println("Listening for UDP on " + addr)

	for {
		n, addr, err := pc.ReadFrom(buffer)

		if err != nil {
			log.Printf(err.Error())
			continue
		}

		// TODO: This doesn't handle character encoding.
		data := string(buffer[:n])

		parts := strings.Split(data, ".")

		log.Printf("Received packet from %s (length: %d): %s", addr.String(), n, buffer)

		if len(parts) != 3 {
			log.Printf("Malformed UDP packet")
			continue
		}

		nodeId := parts[0]

		if len(nodeId) == 0 {
			log.Printf("No node ID found in packet")
			continue
		}

		instanceId, err := uuid.Parse(parts[1])

		if err != nil {
			log.Printf("Invalid instance ID: %s", err)
			continue
		}

		sig, err := uuid.Parse(parts[2])

		if err != nil {
			log.Printf("Invalid signature: %s", err)
			continue
		}

		signatures = append(signatures, signature{instanceId, nodeId, sig, time.Now()})

		log.Printf("Signature captured. Length: %d", len(signatures))
	}
}