package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net"
	"net/http"
	"os"
	"single-executor/internal/util"
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

type signatureListener struct {
	ch   chan Signature
	open bool
}

var signatureListeners []signatureListener

func main() {
	portEnv := os.Getenv("CHAIN_PORT")

	port, err := strconv.Atoi(portEnv)

	if err != nil {
		log.Fatalf("Could not resolve a UDP port from environment CHAIN_PORT %s\n", err.Error())
	}

	signatures = make([]Signature, 0)
	signatureListeners = make([]signatureListener, 0)

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

		s := Signature{instanceId, nodeId, sig, time.Now()}
		signatures = append(signatures, s)

		for _, l := range signatureListeners {
			log.Println("About to send to channel")
			if l.open {
				log.Println("Sending...")
				l.ch <- s
			} else {
				log.Println("Channel closed. Skipping.")
			}
		}

		log.Printf("Signature captured. Length: %d", len(signatures))
	}
}


type httpHandler struct {}

func (handler *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	messages, done, start := util.HandleSse(writer, request)

	listener := signatureListener{make(chan Signature), true}
	signatureListeners = append(signatureListeners, listener)

	sendSignature := func(s Signature) {
		data, err := json.Marshal(s)

		if err != nil {
			log.Printf("Failed to send signature: %s\n", err.Error())
			return
		}

		messages <- data	// Send it to the SSE.
	}

	go func() {
		for _, s := range signatures {
			sendSignature(s)
		}

		for {
			select {
			case signature := <-listener.ch:
				// A new signature has arrived.
				sendSignature(signature)
			case <- done:
				// SSE terminated.
				log.Println("Closing channel")
				close(listener.ch)
				listener.open = false
				return
			}
		}
	}()

	start()
}
