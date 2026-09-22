// Package share es el listener público. SOLO lee la sesión; nunca la controla.
package share

import (
	"html/template"
	"io/fs"
	"net"
	"net/http"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
	"github.com/KonixDev/pasame/web"
)

type Deps struct {
	Current    func() *session.Session
	Stats      func() *session.Stats
	Quarantine string
	Sender     func() string
	Strict     func() bool
}

type Server struct {
	d   Deps
	mux *http.ServeMux
	tpl *template.Template
}

// view es lo que reciben todas las plantillas.
type view struct {
	Lang   i18n.Lang
	Sender string
	Sess   *session.Session
	Strict bool
	Msg    string // aviso de resultado (subida OK, error)
	MsgErr bool
}

func New(d Deps) (*Server, error) {
	css, err := fs.ReadFile(web.Share, "share/page.css")
	if err != nil {
		return nil, err
	}
	tpl, err := template.New("").Funcs(template.FuncMap{
		"t":        i18n.T,
		"size":     i18n.Size,
		"css":      func() template.CSS { return template.CSS(css) },
		"viewable": Viewable,
		"js":       func() template.JS { return template.JS(mustRead("share/page.js")) },
	}).ParseFS(web.Share, "share/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{d: d, mux: http.NewServeMux(), tpl: tpl}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	s.mux.HandleFunc("GET /{$}", s.root)
	s.mux.HandleFunc("GET /s/{tok}", s.withSession(s.page))
	s.mux.HandleFunc("GET /s/{tok}/f/{i}", s.withSession(s.file))
	s.mux.HandleFunc("GET /s/{tok}/zip", s.withSession(s.zip))
	s.mux.HandleFunc("POST /s/{tok}/up", s.withSession(s.upload))
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) view(r *http.Request, sess *session.Session) view {
	return view{Lang: i18n.Pick(r.Header.Get("Accept-Language")), Sender: s.d.Sender(), Sess: sess, Strict: s.d.Strict()}
}

func (s *Server) render(w http.ResponseWriter, status int, name string, v view) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	s.tpl.ExecuteTemplate(w, name, v)
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	sess := s.d.Current()
	switch {
	case sess == nil:
		s.render(w, http.StatusGone, "gone.html", s.view(r, nil))
	case s.d.Strict():
		v := s.view(r, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		s.tpl.ExecuteTemplate(w, "gone.html", view{Lang: v.Lang, Msg: i18n.T(v.Lang, "root_strict")})
	default:
		http.Redirect(w, r, sess.Path(), http.StatusFound)
	}
}

type sessHandler func(http.ResponseWriter, *http.Request, *session.Session, *session.Stats)

// withSession resuelve el token; si no es el actual, 410. Registra al cliente.
func (s *Server) withSession(h sessHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := s.d.Current()
		if sess == nil || r.PathValue("tok") != sess.Token {
			s.render(w, http.StatusGone, "gone.html", s.view(r, nil))
			return
		}
		st := s.d.Stats()
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			st.ClientSeen(host)
		}
		h(w, r, sess, st)
	}
}

// Stubs: los completan las Tareas 7 a 10.
func (s *Server) zip(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
func (s *Server) upload(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
