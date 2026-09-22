package share

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KonixDev/pasame/internal/files"
	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

const uploadIdle = 60 * time.Second

var errEmpty = errors.New("archivo vacío")

func defaultCreatePart(p string) (io.WriteCloser, error) {
	return os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
}

// upload lee el multipart parte por parte: nunca ParseMultipartForm (bufferiza).
func (s *Server) upload(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	rc := http.NewResponseController(w)
	rc.SetReadDeadline(time.Now().Add(uploadIdle)) // en httptest.Recorder devuelve error: se ignora
	mr, err := r.MultipartReader()
	if err != nil {
		s.uploadResult(w, r, sess, nil, "err_generic", http.StatusBadRequest)
		return
	}
	os.MkdirAll(s.d.Quarantine, 0o755)
	var saved []string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.uploadResult(w, r, sess, saved, "err_generic", http.StatusBadRequest)
			return
		}
		if part.FileName() == "" {
			continue // campos que no son archivos
		}
		name, err := s.savePart(part, rc, st)
		switch {
		case errors.Is(err, syscall.ENOSPC):
			s.uploadResult(w, r, sess, saved, "err_no_space", http.StatusInsufficientStorage)
			return
		case errors.Is(err, errEmpty):
			s.uploadResult(w, r, sess, saved, "err_empty", http.StatusBadRequest)
			return
		case err != nil:
			log.Printf("upload: %v", err)
			s.uploadResult(w, r, sess, saved, "err_generic", http.StatusBadRequest)
			return
		}
		saved = append(saved, name)
	}
	s.uploadResult(w, r, sess, saved, "", http.StatusOK)
}

// savePart escribe en "<nombre>.part" y renombra al terminar: nunca queda un archivo a medias con nombre final.
func (s *Server) savePart(part *multipart.Part, rc *http.ResponseController, st *session.Stats) (string, error) {
	dest, err := files.UniquePath(s.d.Quarantine, part.FileName())
	if err != nil {
		return "", err
	}
	name := filepath.Base(dest)
	tmp := dest + ".part"
	out, err := s.createPart(tmp)
	if err != nil {
		return "", err
	}
	st.UploadStarted(name)
	bp := bufPool.Get().(*[]byte)
	n, err := io.CopyBuffer(out, &idleReader{r: part, rc: rc}, *bp)
	bufPool.Put(bp)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n == 0 {
		err = errEmpty
	}
	if err == nil {
		err = os.Rename(tmp, dest)
	}
	if err != nil {
		os.Remove(tmp)
		st.UploadDone(name, false)
		return "", err
	}
	files.MarkDownloaded(dest)
	st.UploadDone(name, true)
	return name, nil
}

// idleReader corre el deadline de lectura en cada Read: corta si pasan 60 s sin bytes.
type idleReader struct {
	r  io.Reader
	rc *http.ResponseController
}

func (i *idleReader) Read(b []byte) (int, error) {
	i.rc.SetReadDeadline(time.Now().Add(uploadIdle))
	return i.r.Read(b)
}

func (s *Server) uploadResult(w http.ResponseWriter, r *http.Request, sess *session.Session, saved []string, errKey string, status int) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := map[string]any{"ok": errKey == "", "files": saved}
		if errKey != "" {
			v := s.view(r, sess)
			msg := i18n.T(v.Lang, errKey)
			if strings.Contains(msg, "%s") {
				msg = i18n.T(v.Lang, errKey, v.Sender)
			}
			resp["error"] = msg
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	loc := sess.Path() + "?subido=" + strconv.Itoa(len(saved))
	if errKey != "" {
		loc = sess.Path() + "?error=" + errKey
	}
	http.Redirect(w, r, loc, http.StatusSeeOther)
}
