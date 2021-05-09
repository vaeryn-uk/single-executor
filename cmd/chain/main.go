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

type signatureStorage struct {
	signatures []Signature
	size int
	listeners []chan Signature
}

func newStorage(size int) *signatureStorage {
	ret := new(signatureStorage)

	ret.size = size
	ret.signatures = make([]Signature, 0)
	ret.listeners = make([]chan Signature, 0)

	return ret
}

func (s *signatureStorage) store(signature Signature) {
	if len(s.signatures) >= s.size {
		for index, sig := range s.signatures {
			if index == 0 {
				// Skip. We'll overwrite in a second.
			} else {
				s.signatures[index - 1] = sig
			}
		}

		s.signatures[len(s.signatures) - 1] = signature
	} else {
		s.signatures = append(s.signatures, signature)
	}

	for _, l := range s.listeners {
		l <- signature
	}
}

func (s *signatureStorage) list() []Signature {
	return s.signatures
}

func (s *signatureStorage) length() int {
	return len(s.signatures)
}

func (s *signatureStorage) listen() <-chan Signature {
	listener := make(chan Signature)

	s.listeners = append(s.listeners, listener)

	return listener
}

func (s *signatureStorage) detach(channel <-chan Signature) {
	listenerIndex := -1

	for index, listener := range s.listeners {
		if channel == listener {
			close(listener)
			listenerIndex = index
		}
	}

	if listenerIndex >= 0 {
		s.listeners[listenerIndex] = s.listeners[len(s.listeners)-1]
		s.listeners = s.listeners[:len(s.listeners)-1]
	}
}

func main() {
	portEnv := os.Getenv("CHAIN_PORT")

	port, err := strconv.Atoi(portEnv)

	if err != nil {
		log.Fatalf("Could not resolve a UDP port from environment CHAIN_PORT %s\n", err.Error())
	}

	storage := newStorage(5)

	udpAddr := "0.0.0.0:" + strconv.Itoa(port)
	httpAddr := "0.0.0.0:80"

	pc, err := net.ListenPacket("udp", udpAddr)

	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)

	log.Printf("Listening for HTTP on %s", httpAddr)

	go func () {
		if err := http.ListenAndServe(httpAddr, &httpHandler{storage}); err != nil {
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

		storage.store(Signature{instanceId, nodeId, sig, time.Now()})
	}
}


type httpHandler struct {
	storage *signatureStorage
}

func (handler *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	messages, done, start := util.HandleSse(writer, request)

	listener := handler.storage.listen()

	sendSignature := func(s Signature) {
		data, err := json.Marshal(s)

		if err != nil {
			log.Printf("Failed to send signature: %s\n", err.Error())
			return
		}

		messages <- data	// Send it to the SSE.
	}

	go func() {
		for _, s := range handler.storage.list() {
			sendSignature(s)
		}

		for {
			select {
			case signature := <-listener:
				// A new signature has arrived.
				sendSignature(signature)
			case <- done:
				// SSE terminated.
				handler.storage.detach(listener)
				return
			}
		}
	}()

	start()
}
