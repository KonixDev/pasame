package addr

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Iface struct {
	Name     string
	IPs      []net.IP
	Up       bool
	Loopback bool
}

type Candidate struct {
	Name    string `json:"name"`
	Human   string `json:"human"`
	IP      string `json:"ip"`
	Virtual bool   `json:"virtual"`
	Score   int    `json:"-"`
}

type LAN struct {
	Port  int
	List  func() ([]Iface, error)
	Route func() net.IP

	mu       sync.Mutex
	override string
}

func NewLAN(port int) *LAN { return &LAN{Port: port, List: systemIfaces, Route: routeIP} }

func (l *LAN) Kind() string { return "lan" }

func (l *LAN) SetOverride(ip string) {
	l.mu.Lock()
	l.override = ip
	l.mu.Unlock()
}

// Prefijos de interfaces virtuales: pierden preferencia pero siguen en "Cambiar red".
var virtualPrefixes = []string{"docker", "br-", "veth", "virbr", "vbox", "vmnet", "utun", "tun", "tap",
	"tailscale", "zt", "wg", "ppp", "hyper-v", "vethernet", "wsl"}

// Prefijos que cuentan como VPN para el aviso (subconjunto de los virtuales).
var vpnPrefixes = []string{"utun", "tun", "tailscale", "wg", "ppp"}

func hasPrefix(name string, ps []string) bool {
	n := strings.ToLower(name)
	for _, p := range ps {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

var (
	_, net192, _ = net.ParseCIDR("192.168.0.0/16")
	_, net10, _  = net.ParseCIDR("10.0.0.0/8")
	_, net172, _ = net.ParseCIDR("172.16.0.0/12")
)

func (l *LAN) Candidates() []Candidate {
	ifs, err := l.List()
	if err != nil {
		return nil
	}
	route := l.Route()
	var out []Candidate
	for _, in := range ifs {
		if !in.Up || in.Loopback {
			continue
		}
		for _, ip := range in.IPs {
			v4 := ip.To4()
			if v4 == nil || v4.IsLinkLocalUnicast() || v4.IsLoopback() {
				continue
			}
			c := Candidate{Name: in.Name, Human: human(in.Name), IP: v4.String(), Virtual: hasPrefix(in.Name, virtualPrefixes)}
			switch {
			case net192.Contains(v4):
				c.Score = 3
			case net10.Contains(v4), net172.Contains(v4):
				c.Score = 2
			default:
				c.Score = 1
			}
			if route != nil && route.Equal(v4) {
				c.Score++
			}
			if n := strings.ToLower(in.Name); strings.HasPrefix(n, "wl") || strings.Contains(n, "wi-fi") ||
				strings.Contains(n, "wlan") || n == "en0" || strings.Contains(n, "ethernet") && !c.Virtual {
				c.Score++
			}
			if c.Virtual {
				c.Score -= 10
			}
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		ri, rj := route != nil && route.String() == out[i].IP, route != nil && route.String() == out[j].IP
		if ri != rj {
			return ri
		}
		return out[i].Name < out[j].Name
	})
	l.mu.Lock()
	ov := l.override
	l.mu.Unlock()
	for i, c := range out {
		if c.IP == ov && i > 0 {
			out = append([]Candidate{c}, append(out[:i:i], out[i+1:]...)...)
			break
		}
	}
	return out
}

func human(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "tailscale"):
		return "Tailscale"
	case strings.HasPrefix(n, "wl"), strings.Contains(n, "wi-fi"), strings.Contains(n, "wlan"), n == "en0":
		return "Wi-Fi"
	case strings.HasPrefix(n, "eth"), strings.HasPrefix(n, "en"), n == "ethernet":
		return "Ethernet"
	}
	return name
}

func (l *LAN) VPN() bool {
	ifs, err := l.List()
	if err != nil {
		return false
	}
	for _, in := range ifs {
		if !in.Up || !hasPrefix(in.Name, vpnPrefixes) {
			continue
		}
		for _, ip := range in.IPs {
			if ip.To4() != nil {
				return true
			}
		}
	}
	return false
}

func (l *LAN) Addresses(_ context.Context, sessionPath string) ([]Address, error) {
	cs := l.Candidates()
	if len(cs) == 0 {
		return nil, nil
	}
	hp := net.JoinHostPort(cs[0].IP, strconv.Itoa(l.Port))
	return []Address{{Kind: "lan", URL: "http://" + hp + sessionPath, Display: hp, Label: "En esta red WiFi"}}, nil
}

func systemIfaces() ([]Iface, error) {
	sys, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []Iface
	for _, s := range sys {
		in := Iface{Name: s.Name, Up: s.Flags&net.FlagUp != 0, Loopback: s.Flags&net.FlagLoopback != 0}
		addrs, _ := s.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				in.IPs = append(in.IPs, ipn.IP)
			}
		}
		out = append(out, in)
	}
	return out, nil
}

// routeIP pregunta al OS qué IP usaría para salir a internet. UDP "connect" no manda paquetes.
func routeIP() net.IP {
	c, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		return nil
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP
}
