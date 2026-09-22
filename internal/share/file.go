package share

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

func (s *Server) file(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	i, err := strconv.Atoi(r.PathValue("i"))
	f, ok := sess.File(i)
	if err != nil || !ok {
		http.NotFound(w, r)
		return
	}
	fh, err := os.Open(f.Abs)
	var info os.FileInfo
	if err == nil {
		info, err = fh.Stat()
	}
	if err != nil || info.Size() != f.Size {
		if fh != nil {
			fh.Close()
		}
		v := s.view(r, sess)
		http.Error(w, i18n.T(v.Lang, "file_gone"), http.StatusNotFound)
		return
	}
	defer fh.Close()

	kind := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		kind = "inline"
	}
	w.Header().Set("Content-Disposition", contentDisposition(kind, f.Name))
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(f.Name))); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	st.DownloadStarted(i)
	cw := &countingWriter{ResponseWriter: w}
	http.ServeContent(cw, r, f.Name, f.ModTime, fh)
	// "Completa" = se mandó el archivo entero en esta respuesta (no cuenta un Range parcial).
	st.DownloadDone(i, cw.status == http.StatusOK && cw.n == f.Size)
}

// contentDisposition arma el header con nombre UTF-8 (RFC 6266) y un fallback ASCII.
func contentDisposition(kind, name string) string {
	var ascii strings.Builder
	for _, r := range name {
		if r >= 0x20 && r < 0x7f && r != '"' && r != '\\' {
			ascii.WriteRune(r)
		}
	}
	return kind + `; filename="` + ascii.String() + `"; filename*=UTF-8''` + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
}

type countingWriter struct {
	http.ResponseWriter
	status int
	n      int64
}

func (c *countingWriter) WriteHeader(code int) { c.status = code; c.ResponseWriter.WriteHeader(code) }

func (c *countingWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	n, err := c.ResponseWriter.Write(b)
	c.n += int64(n)
	return n, err
}

// Unwrap deja que http.ResponseController llegue al writer real (flush, deadlines).
func (c *countingWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
