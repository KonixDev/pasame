package core

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTunnel struct {
	err     error
	done    chan struct{}
	stopped atomic.Bool
}

func (f *fakeTunnel) start(_ context.Context, progress func(int)) (string, <-chan struct{}, func(), error) {
	progress(50)
	if f.err != nil {
		return "", nil, nil, f.err
	}
	f.done = make(chan struct{})
	return "https://amber-cat.trycloudflare.com", f.done, func() {
		if f.stopped.CompareAndSwap(false, true) {
			close(f.done)
		}
	}, nil
}

func withTunnel(t *testing.T, ft *fakeTunnel, available bool) *env {
	e := newEnv(t)
	e.c.o.Tunnel = ft.start
	e.c.o.TunnelAvailable = func() bool { return available }
	return e
}

func TestEnableTunnel(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.Share([]string{e.file(t, "a", 1)})
	keyBefore := append([]byte(nil), e.c.PINKey()...)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	s := e.c.State()
	pin := e.c.Current().PIN
	if !s.Strict || !e.c.Strict() || s.Tunnel.Host != "amber-cat.trycloudflare.com" {
		t.Fatalf("%+v", s.Tunnel)
	}
	if s.Tunnel.PIN != strings.Join(strings.Split(pin, ""), " ") {
		t.Fatalf("PIN mostrado %q", s.Tunnel.PIN)
	}
	if s.Addresses[0].Kind != "tunnel" || !strings.HasSuffix(s.Addresses[0].URL, "?pin="+pin) {
		t.Fatalf("primaria %+v", s.Addresses[0])
	}
	if s.Addresses[0].Display != "amber-cat.trycloudflare.com"+e.c.Current().Path() {
		t.Fatalf("lo que se tipea tiene que llevar la ruta (sin PIN): %q", s.Addresses[0].Display)
	}
	if !strings.HasSuffix(s.Addresses[1].URL, "?pin="+pin) {
		t.Fatal("la dirección LAN también necesita el PIN en modo estricto")
	}
	if bytes.Equal(keyBefore, e.c.PINKey()) {
		t.Fatal("la clave no rotó al prender")
	}
}

func TestDisableTunnel(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	key := append([]byte(nil), e.c.PINKey()...)
	e.c.DisableTunnel()
	s := e.c.State()
	if s.Tunnel.Status != "off" || s.Strict || !ft.stopped.Load() || s.Addresses[0].Kind != "lan" {
		t.Fatalf("%+v", s)
	}
	if strings.Contains(s.Addresses[0].URL, "?pin=") {
		t.Fatal("quedó el PIN en la URL LAN")
	}
	if bytes.Equal(key, e.c.PINKey()) {
		t.Fatal("la clave no rotó al apagar")
	}
}

func TestTunnelDropsOnItsOwn(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	close(ft.done) // Cloudflare cortó
	ft.stopped.Store(true)
	waitFor(t, func() bool { s := e.c.State(); return s.Tunnel.Status == "off" && s.Tunnel.Dropped && !s.Strict })
}

func TestTunnelError(t *testing.T) {
	e := withTunnel(t, &fakeTunnel{err: errors.New("línea 1\nlínea 2")}, true)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "error" })
	if s := e.c.State(); s.Strict || !strings.Contains(s.Tunnel.Detail, "línea 2") {
		t.Fatalf("%+v", s.Tunnel)
	}
}

func TestTunnelUnavailable(t *testing.T) {
	e := withTunnel(t, &fakeTunnel{}, false)
	if s := e.c.State(); s.Tunnel.Status != "unavailable" {
		t.Fatalf("%+v", s.Tunnel)
	}
	e.c.EnableTunnel()
	if s := e.c.State(); s.Tunnel.Status != "unavailable" || s.Strict {
		t.Fatalf("%+v", s.Tunnel)
	}
}

// Terminar o cerrar mientras se conecta: no puede quedar cloudflared andando ni el estado en "on".
func TestDisableWhileConnecting(t *testing.T) {
	e := newEnv(t)
	started := make(chan struct{})
	e.c.o.TunnelAvailable = func() bool { return true }
	e.c.o.Tunnel = func(ctx context.Context, _ func(int)) (string, <-chan struct{}, func(), error) {
		close(started)
		<-ctx.Done()
		return "", nil, nil, ctx.Err()
	}
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.EnableTunnel()
	<-started
	e.c.DisableTunnel()
	time.Sleep(50 * time.Millisecond)
	if s := e.c.State(); s.Tunnel.Status != "off" || s.Strict {
		t.Fatalf("%+v", s.Tunnel)
	}
}

// Terminar de compartir también corta internet: la próxima vez se arranca solo por WiFi.
func TestStopTurnsTunnelOff(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	e.c.Stop()
	if s := e.c.State(); s.Tunnel.Status != "off" || s.Strict || !ft.stopped.Load() {
		t.Fatalf("%+v", s.Tunnel)
	}
}
