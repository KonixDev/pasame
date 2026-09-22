package core

import (
	"net"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
)

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestIdleRuleQuitsAfter120s(t *testing.T) {
	e := newEnv(t)
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(119 * time.Second))
	if closed(e.c.Done()) {
		t.Fatal("salió antes de tiempo")
	}
	e.c.tick(t0.Add(121 * time.Second))
	if !closed(e.c.Done()) {
		t.Fatal("no salió a los 120 s")
	}
}

func TestIdleRuleWaitsForTab(t *testing.T) {
	e := newEnv(t)
	release := e.c.ClientConnected()
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(10 * time.Minute))
	if closed(e.c.Done()) {
		t.Fatal("salió con la pestaña abierta")
	}
	release()
	e.c.tick(t0.Add(11 * time.Minute))
	e.c.tick(t0.Add(13 * time.Minute))
	if !closed(e.c.Done()) {
		t.Fatal("no salió después de cerrar la pestaña")
	}
}

func TestIdleRuleWaitsForTransfers(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.Stats().DownloadStarted(0)
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(10 * time.Minute))
	if closed(e.c.Done()) {
		t.Fatal("salió con una descarga en curso")
	}
	e.c.Stats().DownloadDone(0, true)
	e.c.tick(t0.Add(11 * time.Minute))
	e.c.tick(t0.Add(13*time.Minute + time.Second))
	if !closed(e.c.Done()) {
		t.Fatal("no salió al terminar la descarga")
	}
}

func TestNetworkChangeIsFlagged(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	t0 := time.Now()
	e.c.tick(t0)
	e.ifs = []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("10.0.0.7")}}}
	ch, cancel := e.c.Subscribe()
	defer cancel()
	e.c.tick(t0.Add(5 * time.Second))
	if !closed(ch) || !e.c.State().NetChanged || e.c.State().Addresses[0].Display != "10.0.0.7:8080" {
		t.Fatalf("%+v", e.c.State())
	}
}

func TestRequestQuitIsIdempotent(t *testing.T) {
	e := newEnv(t)
	e.c.RequestQuit()
	e.c.RequestQuit()
	if !closed(e.c.Done()) {
		t.Fatal("Done no se cerró")
	}
}
