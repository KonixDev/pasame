package share

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

const (
	maxFails    = 5
	lockoutTime = 60 * time.Second
)

// limiter es global a propósito: por el túnel todas las IP llegan como 127.0.0.1.
type limiter struct {
	mu          sync.Mutex
	fails       int
	lockedUntil time.Time
}

func (l *limiter) locked(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.lockedUntil.IsZero() && !now.Before(l.lockedUntil) {
		l.lockedUntil, l.fails = time.Time{}, 0
	}
	return now.Before(l.lockedUntil)
}

func (l *limiter) fail(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fails++; l.fails >= maxFails {
		l.lockedUntil = now.Add(lockoutTime)
	}
}

func (l *limiter) reset() { l.mu.Lock(); l.fails = 0; l.mu.Unlock() }

func pinCookieValue(key []byte, token string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Server) hasPINCookie(r *http.Request, sess *session.Session) bool {
	c, err := r.Cookie("pasame_pin")
	if err != nil {
		return false
	}
	want := pinCookieValue(s.d.PINKey(), sess.Token)
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1
}

// checkPIN valida un PIN recibido; si es correcto deja la cookie. Si falla, devuelve el código HTTP y el texto.
func (s *Server) checkPIN(w http.ResponseWriter, sess *session.Session, pin string) (ok bool, status int, msgKey string) {
	now := time.Now()
	if s.lim.locked(now) {
		return false, http.StatusTooManyRequests, "pin_locked"
	}
	if subtle.ConstantTimeCompare([]byte(pin), []byte(sess.PIN)) != 1 {
		s.lim.fail(now) // el 5.º fallo todavía responde "La clave no es esa"; el bloqueo se ve en el 6.º intento
		return false, http.StatusUnauthorized, "pin_wrong"
	}
	s.lim.reset()
	http.SetCookie(w, &http.Cookie{
		Name: "pasame_pin", Value: pinCookieValue(s.d.PINKey(), sess.Token),
		Path: sess.Path(), HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return true, 0, ""
}

// gated agrega el control de PIN a una ruta de sesión. En modo LAN no hace nada.
func (s *Server) gated(h sessHandler) sessHandler {
	return func(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
		if !s.d.Strict() || s.hasPINCookie(r, sess) {
			h(w, r, sess, st)
			return
		}
		if pin := r.URL.Query().Get("pin"); pin != "" && r.Method == http.MethodGet {
			if ok, status, key := s.checkPIN(w, sess, pin); !ok {
				s.pinPage(w, r, sess, status, key)
				return
			}
			q := r.URL.Query()
			q.Del("pin")
			target := r.URL.Path
			if len(q) > 0 {
				target += "?" + q.Encode()
			}
			http.Redirect(w, r, target, http.StatusSeeOther)
			return
		}
		s.pinPage(w, r, sess, http.StatusUnauthorized, "")
	}
}

func (s *Server) pinPost(w http.ResponseWriter, r *http.Request, sess *session.Session, _ *session.Stats) {
	if ok, status, key := s.checkPIN(w, sess, strings.TrimSpace(r.PostFormValue("pin"))); !ok {
		s.pinPage(w, r, sess, status, key)
		return
	}
	http.Redirect(w, r, sess.Path(), http.StatusSeeOther)
}

func (s *Server) pinPage(w http.ResponseWriter, r *http.Request, sess *session.Session, status int, msgKey string) {
	v := s.view(r, sess)
	if msgKey == "pin_wrong" {
		v.Msg = i18n.T(v.Lang, msgKey, v.Sender)
	} else if msgKey != "" {
		v.Msg = i18n.T(v.Lang, msgKey)
	}
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		msg := v.Msg
		if msg == "" {
			msg = i18n.T(v.Lang, "pin_prompt", v.Sender)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
		return
	}
	s.render(w, status, "pin.html", v)
}
