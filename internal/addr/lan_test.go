package addr

import (
	"context"
	"net"
	"testing"
)

func ifc(name, ip string) Iface {
	return Iface{Name: name, Up: true, IPs: []net.IP{net.ParseIP(ip)}}
}

func lanWith(route string, ifs ...Iface) *LAN {
	l := NewLAN(8080)
	l.List = func() ([]Iface, error) { return ifs, nil }
	l.Route = func() net.IP { return net.ParseIP(route) }
	return l
}

func primary(t *testing.T, l *LAN) string {
	t.Helper()
	as, _ := l.Addresses(context.Background(), "/s/abcde")
	if len(as) == 0 {
		return ""
	}
	return as[0].Display
}

func TestLANScenarios(t *testing.T) {
	lo := Iface{Name: "lo0", Up: true, Loopback: true, IPs: []net.IP{net.ParseIP("127.0.0.1")}}
	cases := []struct {
		name string
		l    *LAN
		want string
		vpn  bool
	}{
		{"solo WiFi", lanWith("192.168.1.42", lo, ifc("en0", "192.168.1.42")), "192.168.1.42:8080", false},
		{"WiFi + Tailscale", lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("utun3", "100.101.1.2")), "192.168.1.42:8080", true},
		{"WiFi + VPN full-tunnel", lanWith("10.8.0.2", ifc("en0", "192.168.1.42"), ifc("utun4", "10.8.0.2")), "192.168.1.42:8080", true},
		{"Ethernet + WiFi misma red", lanWith("192.168.1.43", ifc("en0", "192.168.1.42"), ifc("en7", "192.168.1.43")), "192.168.1.43:8080", false},
		{"Docker + WiFi", lanWith("192.168.0.10", ifc("docker0", "172.17.0.1"), ifc("br-3f2a", "172.18.0.1"), ifc("veth12", "172.17.0.5"), ifc("wlan0", "192.168.0.10")), "192.168.0.10:8080", false},
		{"solo VPN", lanWith("10.8.0.2", lo, ifc("wg0", "10.8.0.2")), "10.8.0.2:8080", true},
		{"sin red", lanWith("", lo), "", false},
		{"Windows Hyper-V + Wi-Fi", lanWith("192.168.1.9", ifc("vEthernet (WSL)", "172.25.0.1"), ifc("Wi-Fi", "192.168.1.9")), "192.168.1.9:8080", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := primary(t, c.l); got != c.want {
				t.Fatalf("primaria = %q, want %q (candidatas %+v)", got, c.want, c.l.Candidates())
			}
			if c.l.VPN() != c.vpn {
				t.Fatalf("VPN() = %v", c.l.VPN())
			}
		})
	}
}

func TestLANOverride(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("tailscale0", "100.101.1.2"))
	l.SetOverride("100.101.1.2")
	if got := primary(t, l); got != "100.101.1.2:8080" {
		t.Fatalf("override ignorado: %q", got)
	}
	l.SetOverride("10.99.99.99") // ya no existe → automático
	if got := primary(t, l); got != "192.168.1.42:8080" {
		t.Fatalf("override inexistente no cayó a automático: %q", got)
	}
}

func TestLANIgnoresUnusable(t *testing.T) {
	l := lanWith("",
		Iface{Name: "en1", Up: false, IPs: []net.IP{net.ParseIP("192.168.5.5")}},
		ifc("en2", "169.254.10.10"),
		ifc("en3", "fe80::1"),
	)
	if c := l.Candidates(); len(c) != 0 {
		t.Fatalf("%+v", c)
	}
}

func TestLANVirtualStillListed(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("tailscale0", "100.101.1.2"))
	c := l.Candidates()
	if len(c) != 2 || c[1].Human != "Tailscale" || !c[1].Virtual || c[0].Human != "Wi-Fi" {
		t.Fatalf("%+v", c)
	}
}

func TestLANAddressShape(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"))
	as, _ := l.Addresses(context.Background(), "/s/k3x9m")
	if len(as) != 1 || as[0].URL != "http://192.168.1.42:8080/s/k3x9m" || as[0].Kind != "lan" || as[0].Label != "En esta red WiFi" {
		t.Fatalf("%+v", as)
	}
}
