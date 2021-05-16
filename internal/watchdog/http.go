package watchdog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// Monitors a watchdog instance, returning its state
// via JSON HTTP response. Useful for human-readable
// diagnostics of a watchdog instance.
type httpMonitor struct {
	w *Watchdog
}

type watchdogReportEvent struct {
	Node  Id        `json:"nodeId"`
	Event string    `json:"event"`
	Term  uint8     `json:"term"`
	Time  time.Time `json:"time"`
}

type sortableEvents []watchdogReportEvent

func (s sortableEvents) Len() int {
	return len(s)
}

func (s sortableEvents) Less(i, j int) bool {
	return s[i].Time.String() < s[j].Time.String()
}

func (s sortableEvents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type watchdogReport struct {
	Id             Id       `json:"id"`
	State          string   `json:"state"`
	Leader         Id       `json:"leader"`
	VotedFor       Id       `json:"votedFor"`
	CurrentTerm    uint8    `json:"currentTerm"`
	Blacklist      []int    `json:"blacklist"`
	Events         sortableEvents `json:"events"`
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
	h.w.event(fmt.Sprintf("blacklist node %d", id))
	writer.WriteHeader(200)
}

func (h *httpMonitor) whitelist(writer http.ResponseWriter, id Id) {
	h.w.adapter.whitelistNode(id)
	h.w.event(fmt.Sprintf("blacklist node %d", id))
	writer.WriteHeader(200)
}

func (h *httpMonitor) reportState(writer http.ResponseWriter) {
	events := make(sortableEvents, 0)

	for timestamp, event := range h.w.events {
		events = append(events, watchdogReportEvent{
			h.w.id,
			event.event,
			event.term,
			timestamp,
		})
	}

	sort.Sort(events)

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


