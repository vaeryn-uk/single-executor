package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Signature struct {
	InstanceId uuid.UUID `json:"InstanceId"`
	NodeId     string    `json:"nodeId"`
	Signature  uuid.UUID `json:"signatureId"`
	Datetime   time.Time `json:"signedAt"`
}

var signatures []Signature

func main() {
	portEnv := os.Getenv("CHAIN_PORT")

	port, err := strconv.Atoi(portEnv)

	if err != nil {
		log.Fatalf("Could not resolve a UDP port from environment CHAIN_PORT %s\n", err.Error())
	}

	signatures = make([]Signature, 0)

	udpAddr := "0.0.0.0:" + strconv.Itoa(port)
	httpAddr := "0.0.0.0:80"

	pc, err := net.ListenPacket("udp", udpAddr)

	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)

	log.Printf("Listening for HTTP on %s", httpAddr)

	go func () {
		if err := http.ListenAndServe(httpAddr, &httpHandler{}); err != nil {
			panic(err)
		}
	}()

	log.Println("Listening for UDP on " + udpAddr)

	for {
		n, addr, err := pc.ReadFrom(buffer)

		if err != nil {
			log.Printf(err.Error())
			continue
		}

		// TODO: This doesn't handle character encoding.
		data := string(buffer[:n])

		parts := strings.Split(data, ".")

		log.Printf("Received packet from %s (length: %d): %s", addr.String(), n, buffer[:n])

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

		signatures = append(signatures, Signature{instanceId, nodeId, sig, time.Now()})

		log.Printf("Signature captured. Length: %d", len(signatures))
	}
}

type httpHandler struct {}

func (handler *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	data, err := json.Marshal(signatures)

	if err != nil {
		writer.WriteHeader(500)
		if _, err := fmt.Fprintf(writer, "Could not encode JSON: %s", err); err != nil {
			panic(err)
		}
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)

	if _, err := writer.Write(data); err != nil {
		panic(err)
	}
}