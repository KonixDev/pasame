package session

import (
	"net"
	"sync"
	"time"
)

type Snapshot struct {
	Clients   int         `json:"clients"`
	Active    map[int]int `json:"active"`
	Completed map[int]int `json:"completed"`
	Uploading []string    `json:"uploading"`
	Received  []string    `json:"received"`
	FirstSeen time.Time   `json:"firstSeen"`
}

type Stats struct {
	mu        sync.Mutex
	onChange  func()
	clients   map[string]bool
	active    map[int]int
	completed map[int]int
	uploading map[string]int
	received  []string
	firstSeen time.Time
}

func NewStats(onChange func()) *Stats {
	return &Stats{
		onChange: onChange, clients: map[string]bool{},
		active: map[int]int{}, completed: map[int]int{}, uploading: map[string]int{},
	}
}

func (s *Stats) change(f func()) {
	s.mu.Lock()
	f()
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Stats) ClientSeen(ip string) {
	if p := net.ParseIP(ip); p == nil || p.IsLoopback() {
		return
	}
	s.mu.Lock()
	known := s.clients[ip]
	s.mu.Unlock()
	if known {
		return
	}
	s.change(func() {
		s.clients[ip] = true
		if s.firstSeen.IsZero() {
			s.firstSeen = time.Now()
		}
	})
}

func (s *Stats) DownloadStarted(i int) { s.change(func() { s.active[i]++ }) }

func (s *Stats) DownloadDone(i int, ok bool) {
	s.change(func() {
		if s.active[i]--; s.active[i] <= 0 {
			delete(s.active, i)
		}
		if ok {
			s.completed[i]++
		}
	})
}

func (s *Stats) UploadStarted(name string) { s.change(func() { s.uploading[name]++ }) }

func (s *Stats) UploadDone(name string, ok bool) {
	s.change(func() {
		if s.uploading[name]--; s.uploading[name] <= 0 {
			delete(s.uploading, name)
		}
		if ok {
			s.received = append(s.received, name)
		}
	})
}

func (s *Stats) InFlight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, v := range s.active {
		n += v
	}
	for _, v := range s.uploading {
		n += v
	}
	return n
}

func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := Snapshot{
		Clients: len(s.clients), Active: map[int]int{}, Completed: map[int]int{},
		Received: append([]string(nil), s.received...), FirstSeen: s.firstSeen,
	}
	for k, v := range s.active {
		snap.Active[k] = v
	}
	for k, v := range s.completed {
		snap.Completed[k] = v
	}
	for k := range s.uploading {
		snap.Uploading = append(snap.Uploading, k)
	}
	return snap
}
