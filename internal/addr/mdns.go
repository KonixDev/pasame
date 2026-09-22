//go:build mdns

package addr

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/hashicorp/mdns"
)

var ErrMDNSDisabled = errors.New("addr: compilado sin mdns")

const mdnsHost = "pasame.local"

type MDNS struct {
	lan      *LAN
	port     int
	announce func(ip string) (func(), error) // devuelve cómo dejar de anunciar

	mu   sync.Mutex
	ip   string
	stop func()
	ok   bool
}

func NewMDNS(lan *LAN, port int) (*MDNS, error) {
	return &MDNS{lan: lan, port: port, announce: realAnnounce(port)}, nil
}

func realAnnounce(port int) func(string) (func(), error) {
	return func(ip string) (func(), error) {
		svc, err := mdns.NewMDNSService("Pasame", "_http._tcp", "", mdnsHost+".", port, []net.IP{net.ParseIP(ip)}, nil)
		if err != nil {
			return nil, err
		}
		srv, err := mdns.NewServer(&mdns.Config{Zone: svc, Logger: log.New(io.Discard, "", 0)})
		if err != nil {
			return nil, err
		}
		return func() { srv.Shutdown() }, nil
	}
}

func (m *MDNS) Kind() string { return "mdns" }

func (m *MDNS) Addresses(_ context.Context, sessionPath string) ([]Address, error) {
	cs := m.lan.Candidates()
	if len(cs) == 0 {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cs[0].IP != m.ip {
		if m.stop != nil {
			m.stop()
		}
		m.ip, m.stop, m.ok = cs[0].IP, nil, false
		if stop, err := m.announce(cs[0].IP); err == nil {
			m.stop, m.ok = stop, true
		} else {
			log.Printf("mdns: %v (no se muestra pasame.local)", err)
		}
	}
	if !m.ok {
		return nil, nil
	}
	hp := mdnsHost + ":" + strconv.Itoa(m.port)
	return []Address{{Kind: "mdns", URL: "http://" + hp + sessionPath, Display: hp, Label: "Nombre fácil (puede no funcionar)"}}, nil
}

func (m *MDNS) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop != nil {
		m.stop()
	}
}
