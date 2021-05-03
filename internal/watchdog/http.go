package watchdog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
)

// Monitors a watchdog instance, returning its state
// via JSON HTTP response. Useful for human-readable
// diagnostics of a watchdog instance.
type httpMonitor struct {
	w *Watchdog
}

type watchdogReport struct {
	Id             Id       `json:"id"`
	State          string   `json:"state"`
	Leader         Id       `json:"leader"`
	VotedFor       Id       `json:"votedFor"`
	CurrentTerm    uint8    `json:"currentTerm"`
	Blacklist      []int    `json:"blacklist"`
	Events         []string `json:"events"`
	RunningProcess string   `json:"process"`
}

func (h httpMonitor) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Simple routing
	switch request.URL.Path {
	case "/state":
		h.reportState(writer)
	case "/blacklist":
		idInput  := request.URL.Query().Get("id")

		if id, err := strconv.Atoi(idInput); err != nil {
			http.Error(writer, "Must provide a numeric ID", http.StatusBadRequest)
		} else {
			h.blacklist(writer, Id(id))
		}
	case "/whitelist":
		idInput  := request.URL.Query().Get("id")

		if id, err := strconv.Atoi(idInput); err != nil {
			http.Error(writer, "Must provide a numeric ID", http.StatusBadRequest)
		} else {
			h.whitelist(writer, Id(id))
		}
	default:
		http.NotFound(writer, request)
	}
}

func (h *httpMonitor) blacklist(writer http.ResponseWriter, id Id) {
	h.w.adapter.blacklistNode(id)
	writer.WriteHeader(200)
}

func (h *httpMonitor) whitelist(writer http.ResponseWriter, id Id) {
	h.w.adapter.whitelistNode(id)
	writer.WriteHeader(200)
}

func (h *httpMonitor) reportState(writer http.ResponseWriter) {
	events := make([]string, 0)

	for timestamp, event := range h.w.events {
		events = append(events, fmt.Sprintf("%s: %s", timestamp.Format("15:04:05.000"), event))
	}

	sort.Strings(events)

	blacklist := make([]int, 0)

	for _, id := range h.w.adapter.blacklist {
		blacklist = append(blacklist, int(id))
	}

	report := watchdogReport{
		h.w.id,
		h.w.state.ToString(),
		h.w.leader,
		h.w.votedFor,
		h.w.currentTerm,
		blacklist,
		events,
		"",
	}

	if h.w.isProcessRunning() {
		report.RunningProcess = h.w.config.command.command
	}

	data, err := json.Marshal(report)

	if err != nil {
		http.Error(writer, err.Error(), 500)
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


