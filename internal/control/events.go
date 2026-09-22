package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	release := s.c.ClientConnected()
	defer release()
	changes, cancel := s.c.Subscribe()
	defer cancel()
	ping := time.NewTicker(5 * time.Second)
	defer ping.Stop()

	send := func() error {
		b, _ := json.Marshal(s.c.State())
		if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", b); err != nil {
			return err
		}
		return rc.Flush()
	}
	if send() != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.c.Done():
			fmt.Fprint(w, "event: quit\ndata: {}\n\n")
			rc.Flush()
			return
		case <-changes:
			if send() != nil {
				return
			}
		case <-ping.C:
			if _, err := fmt.Fprint(w, "event: ping\ndata: {}\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		}
	}
}
