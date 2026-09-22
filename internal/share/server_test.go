package share

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Fatalf("%d %q", w.Code, w.Body)
	}
}

func TestRootRedirectsToSession(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusFound || w.Header().Get("Location") != f.sess.Path() {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Location"))
	}
}

func TestRootStrictDoesNotRevealToken(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	w := f.do(httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), f.sess.Token) {
		t.Fatalf("%d, body contiene token=%v", w.Code, strings.Contains(w.Body.String(), f.sess.Token))
	}
	if !strings.Contains(w.Body.String(), "código QR") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestOldTokenIsGone(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	for _, p := range []string{"/s/zzzzz", "/s/zzzzz/f/0", "/s/zzzzz/zip"} {
		w := f.do(httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusGone || !strings.Contains(w.Body.String(), "Este envío ya terminó") {
			t.Fatalf("%s → %d", p, w.Code)
		}
	}
}

func TestNoSessionIsGone(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.sess = nil
	if w := f.do(httptest.NewRequest("GET", "/", nil)); w.Code != http.StatusGone {
		t.Fatalf("/ sin sesión → %d", w.Code)
	}
}

func TestGoneInEnglish(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	r := httptest.NewRequest("GET", "/s/zzzzz", nil)
	r.AddCookie(&http.Cookie{Name: "pasame_lang", Value: "en"})
	if w := f.do(r); !strings.Contains(w.Body.String(), "This share has ended") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestClientSeenOnSessionRequest(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.do(httptest.NewRequest("GET", f.sess.Path(), nil))
	if f.stats.Snapshot().Clients != 1 {
		t.Fatal("no registró al cliente")
	}
}

func TestListenFallback(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := busy.Addr().(*net.TCPAddr).Port
	l, err := Listen("127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	got := l.Addr().(*net.TCPAddr).Port
	if got == port {
		t.Fatal("usó el puerto ocupado")
	}
}

func TestHTTPServerTimeouts(t *testing.T) {
	s := HTTPServer(http.NotFoundHandler())
	if s.ReadTimeout != 0 || s.WriteTimeout != 0 || s.ReadHeaderTimeout == 0 || s.IdleTimeout == 0 {
		t.Fatalf("%+v", s)
	}
}
