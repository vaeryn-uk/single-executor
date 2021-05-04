package util

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func GetIntParam(name string, values url.Values) (int, error) {
	input := values.Get(name)

	val, err := strconv.Atoi(input)

	if err != nil {
		return 0, err
	}

	return val, nil
}

// HandleSse supports serve-sent-events, which are long running connections where the server
// will periodically send new data to the client.
// Send byte data to the first returned channel to send this data to the client.
// The second channel will receive a bool once the SSE is complete (because the client
// disconnected). This can be used to terminate a background goroutine that is sending
// data to the client.
// Finally, the func returned should be called in the current goroutine where the HTTP
// request is being handled. It is important the calling HandleFunc code does not return, as that
// will terminate the SSE conn prematurely.
func HandleSse(w http.ResponseWriter, r *http.Request) (chan<- []byte, <-chan bool, func()) {
	messages := make(chan []byte)
	done := make(chan bool)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher := w.(http.Flusher)

	return messages, done, func() {
		defer func() {
			close(messages)
			done <- true
		}()

		for {
			select {
			case <- r.Context().Done():
				return
			case message := <- messages:
				if _, err := fmt.Fprintf(w, "data: %s\n\n", message); err != nil {
					log.Printf("%s\n", err.Error())
				} else {
					flusher.Flush()
				}
			}
		}
	}
}

func ResponseWithJson(w http.ResponseWriter, data []byte) error {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write(data)

	return err
}
