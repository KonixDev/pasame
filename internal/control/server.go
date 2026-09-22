// Package control es la UI local del emisor. Solo escucha en loopback y solo conoce a core.
package control

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/web"
)

func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type server struct {
	c     *core.Core
	token string
	hosts map[string]bool
	mux   *http.ServeMux
}

func New(c *core.Core, token string, port int) http.Handler {
	p := strconv.Itoa(port)
	s := &server{c: c, token: token, mux: http.NewServeMux(),
		hosts: map[string]bool{"127.0.0.1:" + p: true, "localhost:" + p: true}}
	static, _ := fs.Sub(web.Control, "control")
	s.mux.Handle("GET /", http.FileServerFS(static))
	s.mux.HandleFunc("GET /events", s.auth(s.events))
	s.mux.HandleFunc("POST /api/pick", s.auth(s.pick))
	s.mux.HandleFunc("POST /api/add", s.auth(s.add))
	s.mux.HandleFunc("POST /api/receive-only", s.auth(s.simple(c.ReceiveOnly)))
	s.mux.HandleFunc("POST /api/stop", s.auth(s.simple(c.Stop)))
	s.mux.HandleFunc("POST /api/quit", s.auth(s.simple(c.RequestQuit)))
	s.mux.HandleFunc("POST /api/name", s.auth(s.name))
	s.mux.HandleFunc("POST /api/iface", s.auth(s.iface))
	s.mux.HandleFunc("POST /api/lang", s.auth(s.lang))
	s.mux.HandleFunc("POST /api/tunnel", s.auth(s.tunnel))
	s.mux.HandleFunc("POST /api/open-folder", s.auth(s.errOnly(c.OpenFolder)))
	s.mux.HandleFunc("POST /api/firewall", s.auth(s.errOnly(c.OpenFirewall)))
	return s
}

// ServeHTTP rechaza cualquier Host que no sea el nuestro (mitiga DNS rebinding).
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.hosts[r.Host] {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	s.mux.ServeHTTP(w, r)
}

func (s *server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Pasame-Token")
		if r.URL.Path == "/events" {
			got = r.URL.Query().Get("t")
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
