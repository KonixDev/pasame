package control

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/internal/platform"
)

const tok = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newCore(t *testing.T) *core.Core {
	lan := addr.NewLAN(8080)
	lan.List = func() ([]addr.Iface, error) {
		return []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}, nil
	}
	lan.Route = func() net.IP { return nil }
	book := addr.NewBook()
	book.Register(lan)
	return core.New(core.Options{SharePort: 8080, Quarantine: t.TempDir(), ConfigDir: t.TempDir(),
		Config: platform.Config{Name: "Martín"}, LAN: lan, Book: book})
}

func req(method, path, body, host, token string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = host
	if token != "" {
		r.Header.Set("X-Pasame-Token", token)
	}
	return r
}

func TestRejectsWithoutToken(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, r := range []*http.Request{
		req("POST", "/api/stop", "", "127.0.0.1:5555", ""),
		req("POST", "/api/stop", "", "127.0.0.1:5555", "malo"),
		req("GET", "/events", "", "127.0.0.1:5555", ""),
		req("GET", "/events?t=malo", "", "127.0.0.1:5555", ""),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s → %d", r.Method, r.URL, w.Code)
		}
	}
}

func TestRejectsForeignHost(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, host := range []string{"evil.com", "192.168.1.42:5555", "127.0.0.1:6666", "localhost.evil.com:5555"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("POST", "/api/stop", "", host, tok))
		if w.Code != http.StatusForbidden {
			t.Errorf("Host %q → %d", host, w.Code)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req("GET", "/", "", host, ""))
		if w.Code != http.StatusForbidden {
			t.Errorf("GET / con Host %q → %d", host, w.Code)
		}
	}
}

func TestIndexServed(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, host := range []string{"127.0.0.1:5555", "localhost:5555"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("GET", "/?t="+tok, "", host, ""))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Pasame") {
			t.Fatalf("%s → %d", host, w.Code)
		}
	}
}

func TestActions(t *testing.T) {
	c := newCore(t)
	h := New(c, tok, 5555)
	do := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("POST", path, body, "127.0.0.1:5555", tok))
		return w
	}
	if w := do("/api/receive-only", ""); w.Code != http.StatusNoContent || c.State().Phase != "sharing" {
		t.Fatalf("%d %+v", w.Code, c.State().Phase)
	}
	if w := do("/api/stop", ""); w.Code != http.StatusNoContent || c.Current() != nil {
		t.Fatal("stop")
	}
	if w := do("/api/name", `{"name":"Rosa"}`); w.Code != http.StatusNoContent || c.Sender() != "Rosa" {
		t.Fatal("name")
	}
	if w := do("/api/name", `{"name":"  "}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "error") {
		t.Fatalf("name vacío → %d", w.Code)
	}
	if w := do("/api/name", `no-json`); w.Code != http.StatusBadRequest {
		t.Fatal("json roto")
	}
	if w := do("/api/iface", `{"ip":"8.8.8.8"}`); w.Code != http.StatusBadRequest {
		t.Fatal("iface ajena aceptada")
	}
	if w := do("/api/pick", `{"kind":"otra"}`); w.Code != http.StatusBadRequest {
		t.Fatal("kind inválido aceptado")
	}
	if w := do("/api/quit", ""); w.Code != http.StatusNoContent {
		t.Fatal("quit")
	}
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("quit no cerró Done")
	}
}

func TestEventsStream(t *testing.T) {
	c := newCore(t)
	srv := httptest.NewUnstartedServer(nil)
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	srv.Config.Handler = New(c, tok, port)
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/events?t=" + tok)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("CT %q", resp.Header.Get("Content-Type"))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	next := func() core.State {
		t.Helper()
		var ev string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "event: ") {
				ev = strings.TrimPrefix(line, "event: ")
			}
			if strings.HasPrefix(line, "data: ") && ev == "state" {
				var s core.State
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &s); err != nil {
					t.Fatal(err)
				}
				return s
			}
		}
		t.Fatal("se cortó el stream")
		return core.State{}
	}
	if s := next(); s.Phase != "idle" {
		t.Fatalf("primer estado %+v", s.Phase)
	}
	c.ReceiveOnly()
	if s := next(); s.Phase != "sharing" || !strings.HasPrefix(s.QR, "<svg") {
		t.Fatalf("estado tras compartir: %+v", s.Phase)
	}
}
