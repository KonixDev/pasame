//go:build mdns

package addr

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestMDNSAddressAndReannounce(t *testing.T) {
	ip := "192.168.1.42"
	lan := NewLAN(8080)
	lan.List = func() ([]Iface, error) { return []Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP(ip)}}}, nil }
	lan.Route = func() net.IP { return nil }
	var announced []string
	m := &MDNS{lan: lan, port: 8080, announce: func(ip string) (func(), error) {
		announced = append(announced, ip)
		return func() {}, nil
	}}
	as, _ := m.Addresses(context.Background(), "/s/abcde")
	if len(as) != 1 || as[0].Display != "pasame.local:8080" || as[0].Kind != "mdns" || as[0].URL != "http://pasame.local:8080/s/abcde" {
		t.Fatalf("%+v", as)
	}
	m.Addresses(context.Background(), "/s/abcde")
	ip = "10.0.0.7"
	m.Addresses(context.Background(), "/s/abcde")
	if len(announced) != 2 || announced[1] != "10.0.0.7" {
		t.Fatalf("anuncios: %v", announced)
	}
}

func TestMDNSFailureIsSilent(t *testing.T) {
	lan := NewLAN(8080)
	lan.List = func() ([]Iface, error) {
		return []Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}, nil
	}
	lan.Route = func() net.IP { return nil }
	m := &MDNS{lan: lan, port: 8080, announce: func(string) (func(), error) { return nil, errors.New("5353 ocupado") }}
	if as, err := m.Addresses(context.Background(), "/s/x"); as != nil || err != nil {
		t.Fatalf("%v %v", as, err)
	}
}
