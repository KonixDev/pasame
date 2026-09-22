// Package share es el listener público. SOLO lee la sesión; nunca la controla.
package share

import (
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"

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
	Lang       func() i18n.Lang // idioma de quien comparte: es el de la página salvo que el receptor lo cambie
}

type Server struct {
	d          Deps
	mux        *http.ServeMux
	tpl        *template.Template
	createPart func(path string) (io.WriteCloser, error) // inyectable en tests (disco lleno)
}

// view es lo que reciben todas las plantillas.
type view struct {
	Lang   i18n.Lang
	Sender string
	Sess   *session.Session
	Strict bool
	Msg    string // aviso de resultado (subida OK, error)
	MsgErr bool

	Alt        i18n.Lang // el otro idioma, para el link de cambio
	AltName    string    // "English" | "Español"
	SuggestAlt bool      // el navegador pide el otro idioma: el link se muestra también arriba
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
		"ext":      typeLabel,
		"js":       func() template.JS { return template.JS(mustRead("share/page.js")) },
	}).ParseFS(web.Share, "share/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{d: d, mux: http.NewServeMux(), tpl: tpl, createPart: defaultCreatePart}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	s.mux.HandleFunc("GET /{$}", s.root)
	s.mux.HandleFunc("GET /s/{tok}", s.withSession(s.page))
	s.mux.HandleFunc("GET /s/{tok}/f/{i}", s.withSession(s.file))
	s.mux.HandleFunc("GET /s/{tok}/zip", s.withSession(s.zip))
	s.mux.HandleFunc("POST /s/{tok}/up", s.withSession(s.upload))
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

var langNames = map[i18n.Lang]string{i18n.ES: "Español", i18n.EN: "English"}

// pickLang: ?lang= explícito, después la cookie del receptor, después el idioma de quien comparte.
// El Accept-Language no decide: en PCs con Windows o Chrome en inglés mostraría inglés a un receptor
// que habla español. Solo sirve para ofrecer el cambio más visible.
func (s *Server) pickLang(r *http.Request) (l i18n.Lang, explicit bool) {
	if q := i18n.Lang(r.URL.Query().Get("lang")); langNames[q] != "" {
		return q, true
	}
	if c, err := r.Cookie("pasame_lang"); err == nil && langNames[i18n.Lang(c.Value)] != "" {
		return i18n.Lang(c.Value), true
	}
	return s.d.Lang(), false
}

func (s *Server) view(r *http.Request, sess *session.Session) view {
	l, explicit := s.pickLang(r)
	v := view{Lang: l, Sender: s.d.Sender(), Sess: sess, Strict: s.d.Strict(), Alt: i18n.EN}
	if l == i18n.EN {
		v.Alt = i18n.ES
	}
	v.AltName = langNames[v.Alt]
	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	v.SuggestAlt = !explicit && i18n.Pick(accept) == v.Alt && strings.Contains(accept, string(v.Alt))
	return v
}

// rememberLang guarda la elección del receptor para las próximas páginas (y envíos) de este emisor.
func rememberLang(w http.ResponseWriter, r *http.Request) {
	if q := i18n.Lang(r.URL.Query().Get("lang")); langNames[q] != "" {
		http.SetCookie(w, &http.Cookie{Name: "pasame_lang", Value: string(q), Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
	}
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
		rememberLang(w, r)
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
