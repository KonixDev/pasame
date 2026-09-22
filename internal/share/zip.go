package share

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KonixDev/pasame/internal/session"
)

const copyBuf = 256 << 10

var bufPool = sync.Pool{New: func() any { b := make([]byte, copyBuf); return &b }}

// zip escribe todos los archivos directo al ResponseWriter. Store: fotos y videos ya vienen comprimidos.
// Sin Content-Length (tamaño desconocido) y sin Range: un ZIP al vuelo no es reanudable.
func (s *Server) zip(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	if len(sess.Files) == 0 {
		http.NotFound(w, r)
		return
	}
	name := "Pasame-" + time.Now().Format("2006-01-02") + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", contentDisposition("attachment", name))
	w.Header().Set("Cache-Control", "no-store")

	// El ZIP cuenta como una descarga "en curso" de cada archivo (lo ve la Actividad del emisor).
	for _, f := range sess.Files {
		st.DownloadStarted(f.Index)
	}
	ok := map[int]bool{}
	defer func() {
		for _, f := range sess.Files {
			st.DownloadDone(f.Index, ok[f.Index])
		}
	}()

	zw := zip.NewWriter(w)
	bp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bp)
	for _, f := range sess.Files {
		fh, err := os.Open(f.Abs)
		if err != nil {
			log.Printf("zip: omito %s: %v", f.Rel, err)
			continue
		}
		hdr := &zip.FileHeader{Name: f.Rel, Method: zip.Store, Modified: f.ModTime}
		hdr.SetMode(0o644)
		dst, err := zw.CreateHeader(hdr)
		if err == nil {
			_, err = io.CopyBuffer(dst, fh, *bp)
		}
		fh.Close()
		if err != nil {
			log.Printf("zip: cortado en %s: %v", f.Rel, err)
			return // el cliente se fue; no hay forma de "arreglar" un stream a medias
		}
		ok[f.Index] = true
	}
	zw.Close()
}
