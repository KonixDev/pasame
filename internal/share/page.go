package share

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
	"github.com/KonixDev/pasame/web"
)

var viewExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".heic": true,
	".mp4": true, ".webm": true,
	".mp3": true, ".m4a": true, ".aac": true, ".wav": true, ".ogg": true,
	".pdf": true,
}

// typeLabel es la etiqueta de tipo que se ve al lado de cada archivo ("MP4", "JPG"). Sin extensión, "—".
func typeLabel(name string) string {
	e := strings.ToUpper(strings.TrimPrefix(filepath.Ext(name), "."))
	if e == "" || len(e) > 4 {
		return "—"
	}
	return e
}

// Viewable indica si el archivo merece botón "Ver" (abrir inline).
func Viewable(name string) bool { return viewExt[strings.ToLower(filepath.Ext(name))] }

func (s *Server) page(w http.ResponseWriter, r *http.Request, sess *session.Session, _ *session.Stats) {
	v := s.view(r, sess)
	if n, err := strconv.Atoi(r.URL.Query().Get("subido")); err == nil && n > 0 {
		v.Msg = i18n.T(v.Lang, "sent_n", n)
		if n == 1 {
			v.Msg = i18n.T(v.Lang, "sent_1")
		}
	}
	if k := r.URL.Query().Get("error"); strings.HasPrefix(k, "err_") {
		// Solo err_no_space lleva %s; pasarle el nombre a las demás agregaría "%!(EXTRA…)".
		msg := i18n.T(v.Lang, k)
		if strings.Contains(msg, "%s") {
			msg = i18n.T(v.Lang, k, v.Sender)
		}
		v.Msg, v.MsgErr = msg, true
	}
	s.render(w, http.StatusOK, "page.html", v)
}

func mustRead(name string) string {
	b, err := fs.ReadFile(web.Share, name)
	if err != nil {
		panic(fmt.Sprintf("falta %s embebido: %v", name, err))
	}
	return string(b)
}
