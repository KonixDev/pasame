package core

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/dialog"
	"github.com/KonixDev/pasame/internal/platform"
)

type env struct {
	c    *Core
	dir  string
	cfg  string
	ifs  []addr.Iface
	pick func(string) ([]string, error)
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{dir: t.TempDir(), cfg: t.TempDir()}
	e.ifs = []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}
	lan := addr.NewLAN(8080)
	lan.List = func() ([]addr.Iface, error) { return e.ifs, nil }
	lan.Route = func() net.IP { return nil }
	book := addr.NewBook()
	book.Register(lan)
	e.c = New(Options{
		SharePort: 8080, Quarantine: t.TempDir(), ConfigDir: e.cfg,
		Config: platform.Config{Name: "Martín"}, LAN: lan, Book: book,
		Pick: func(k string) ([]string, error) { return e.pick(k) },
	})
	return e
}

func (e *env) file(t *testing.T, name string, n int) string {
	p := filepath.Join(e.dir, name)
	os.WriteFile(p, make([]byte, n), 0o644)
	return p
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout esperando condición")
}

func TestIdleState(t *testing.T) {
	e := newEnv(t)
	s := e.c.State()
	if s.Phase != "idle" || s.Name != "Martín" || e.c.Current() != nil || e.c.Strict() {
		t.Fatalf("%+v", s)
	}
}

func TestShare(t *testing.T) {
	e := newEnv(t)
	ch, cancel := e.c.Subscribe()
	defer cancel()
	e.c.Share([]string{e.file(t, "a.jpg", 1300), filepath.Join(e.dir, "no-existe.mp4")})
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no notificó")
	}
	s := e.c.State()
	if s.Phase != "sharing" || s.Count != 1 || s.Total != "1,3 KB" || s.Files[0].Name != "a.jpg" {
		t.Fatalf("%+v", s)
	}
	if len(s.Unreadable) != 1 || s.Unreadable[0] != "no-existe.mp4" {
		t.Fatalf("Unreadable = %v", s.Unreadable)
	}
	tok := e.c.Current().Token
	if len(s.Addresses) != 1 || s.Addresses[0].URL != "http://192.168.1.42:8080/s/"+tok || !strings.HasPrefix(s.QR, "<svg") {
		t.Fatalf("%+v", s.Addresses)
	}
	if s.SharedAt == 0 {
		t.Fatal("SharedAt vacío")
	}
}

func TestShareAgainIsNewSession(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	old := e.c.Current().Token
	e.c.Share([]string{e.file(t, "b", 1)})
	if e.c.Current().Token == old {
		t.Fatal("reutilizó el token")
	}
}

func TestReceiveOnlyAndStop(t *testing.T) {
	e := newEnv(t)
	e.c.ReceiveOnly()
	if s := e.c.State(); s.Phase != "sharing" || !s.ReceiveOnly || s.Count != 0 {
		t.Fatalf("%+v", s)
	}
	e.c.Stop()
	if s := e.c.State(); s.Phase != "idle" || e.c.Current() != nil || s.QR != "" {
		t.Fatalf("%+v", s)
	}
}

func TestPickFlows(t *testing.T) {
	e := newEnv(t)
	p := e.file(t, "x.pdf", 5)
	e.pick = func(k string) ([]string, error) {
		if k != "files" {
			t.Errorf("kind = %q", k)
		}
		return []string{p}, nil
	}
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "sharing" })

	e.c.Stop()
	e.pick = func(string) ([]string, error) { return nil, dialog.ErrCanceled }
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "idle" })

	e.pick = func(string) ([]string, error) { return nil, dialog.ErrUnsupported }
	e.c.Pick("folder")
	waitFor(t, func() bool { return e.c.State().PickUnsupported })
}

func TestPickErrorKeepsCurrentSession(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.pick = func(string) ([]string, error) { return nil, errors.New("boom") }
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "sharing" })
}

func TestSetNamePersists(t *testing.T) {
	e := newEnv(t)
	if err := e.c.SetName("  Abuela Rosa  "); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := platform.LoadConfig(e.cfg)
	if cfg.Name != "Abuela Rosa" || e.c.Sender() != "Abuela Rosa" {
		t.Fatalf("%+v %q", cfg, e.c.Sender())
	}
	if err := e.c.SetName("   "); err == nil {
		t.Fatal("nombre vacío aceptado")
	}
}

func TestSetIface(t *testing.T) {
	e := newEnv(t)
	e.ifs = append(e.ifs, addr.Iface{Name: "tailscale0", Up: true, IPs: []net.IP{net.ParseIP("100.64.0.9")}})
	e.c.Share([]string{e.file(t, "a", 1)})
	if err := e.c.SetIface("100.64.0.9"); err != nil {
		t.Fatal(err)
	}
	if s := e.c.State(); s.Addresses[0].Display != "100.64.0.9:8080" {
		t.Fatalf("%+v", s.Addresses)
	}
	if err := e.c.SetIface("1.2.3.4"); err == nil {
		t.Fatal("aceptó una IP que no es de esta compu")
	}
	cfg, _, _ := platform.LoadConfig(e.cfg)
	if cfg.IfaceOverride != "100.64.0.9" {
		t.Fatalf("%+v", cfg)
	}
}
