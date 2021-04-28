package watchdog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
)

// Monitors a watchdog instance, returning its state
// via JSON HTTP response. Useful for human-readable
// diagnostics of a watchdog instance.
type httpMonitor struct {
	w *Watchdog
}

type watchdogReport struct {
	Id Id `json:"id"`
	State string `json:"state"`
	Leader Id `json:"leader"`
	VotedFor Id `json:"votedFor"`
	CurrentTerm uint8 `json:"currentTerm"`
	Events []string `json:"events"`
}

func (h httpMonitor) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	events := make([]string, 0)

	for timestamp, event := range h.w.events {
		events = append(events, fmt.Sprintf("%s: %s", timestamp.Format("15:04:05.000"), event))
	}

	sort.Strings(events)

	report := watchdogReport{
		h.w.id,
		h.w.state.ToString(),
		h.w.leader,
		h.w.votedFor,
		h.w.currentTerm,
		events,
	}

	data, err := json.Marshal(report)

	if err != nil {
		writer.WriteHeader(500)
		_, err = writer.Write([]byte(err.Error()))

		if err != nil {
			log.Fatalln(err)
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	_, err = writer.Write(data)

	if err != nil {
		log.Fatalln(err)
	}
}

func HttpMonitor(w *Watchdog) error {
	monitor := httpMonitor{w}

	err := http.ListenAndServe("0.0.0.0:80", monitor)

	if err != nil {
		return err
	}

	return nil
}


