package share

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func timeZero() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }

func withCookie(r *http.Request, c *http.Cookie) *http.Request {
	if c != nil {
		r.AddCookie(c)
	}
	return r
}

func cookieFrom(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == "pasame_pin" {
			return c
		}
	}
	return nil
}

func TestLANModeNeedsNoPIN(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	if w := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)); w.Code != 200 {
		t.Fatalf("%d", w.Code)
	}
}

func TestStrictBlocksEveryRoute(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	for _, r := range []*http.Request{
		httptest.NewRequest("GET", f.sess.Path(), nil),
		httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil),
		httptest.NewRequest("GET", f.sess.Path()+"/zip", nil),
		httptest.NewRequest("POST", f.sess.Path()+"/up", strings.NewReader("")),
	} {
		w := f.do(r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s → %d", r.Method, r.URL.Path, w.Code)
		}
	}
	w := f.do(httptest.NewRequest("GET", f.sess.Path(), nil))
	if !strings.Contains(w.Body.String(), "Escribí la clave de 4 números que aparece en la pantalla de Martín.") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestPINFromQRSetsCookieAndCleansURL(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil))
	c := cookieFrom(w)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != f.sess.Path() || c == nil {
		t.Fatalf("%d %q %v", w.Code, w.Header().Get("Location"), c)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != f.sess.Path() || c.Secure {
		t.Fatalf("cookie %+v", c)
	}
	for _, p := range []string{"", "/f/0", "/zip"} {
		if w := f.do(withCookie(httptest.NewRequest("GET", f.sess.Path()+p, nil), c)); w.Code != 200 {
			t.Errorf("%s con cookie → %d", p, w.Code)
		}
	}
}

func TestPINTyped(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	form := url.Values{"pin": {f.sess.PIN}}.Encode()
	r := httptest.NewRequest("POST", f.sess.Path()+"/pin", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := f.do(r)
	if w.Code != http.StatusSeeOther || cookieFrom(w) == nil {
		t.Fatalf("%d", w.Code)
	}
}

func TestWrongPINAndLockout(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	wrong := "0000"
	if f.sess.PIN == wrong {
		wrong = "1111"
	}
	for i := 0; i < 5; i++ {
		w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+wrong, nil))
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "La clave no es esa") {
			t.Fatalf("intento %d → %d", i, w.Code)
		}
	}
	// Bloqueado: ni siquiera el PIN correcto entra durante 60 s.
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil))
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "Demasiados intentos") {
		t.Fatalf("%d", w.Code)
	}
}

func TestKeyRotationInvalidatesCookies(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	c := cookieFrom(f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil)))
	f.key = []byte("otra-clave-distinta-32-bytes-000")
	if w := f.do(withCookie(httptest.NewRequest("GET", f.sess.Path(), nil), c)); w.Code != http.StatusUnauthorized {
		t.Fatalf("cookie vieja aceptada: %d", w.Code)
	}
}

func TestLimiter(t *testing.T) {
	var l limiter
	now := timeZero()
	for i := 0; i < 4; i++ {
		l.fail(now)
	}
	if l.locked(now) {
		t.Fatal("bloqueó con 4")
	}
	l.fail(now)
	if !l.locked(now) || !l.locked(now.Add(59e9)) || l.locked(now.Add(61e9)) {
		t.Fatal("ventana de 60 s incorrecta")
	}
}
