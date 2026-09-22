package share

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KonixDev/pasame/internal/session"
)

func readZip(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("ZIP inválido: %v", err)
	}
	out := map[string][]byte{}
	for _, zf := range zr.File {
		if zf.Method != zip.Store {
			t.Errorf("%s comprimido con método %d", zf.Name, zf.Method)
		}
		rc, _ := zf.Open()
		out[zf.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	return out
}

func TestZipAll(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 100, "b.bin": 200_000})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="Pasame-`) || !strings.HasSuffix(cd, ".zip") {
		t.Fatalf("CD = %q", cd)
	}
	files := readZip(t, w.Body.Bytes())
	if !bytes.Equal(files["a.txt"], pattern(100)) || !bytes.Equal(files["b.bin"], pattern(200_000)) {
		t.Fatal("contenido distinto")
	}
}

func TestZipKeepsFolderStructure(t *testing.T) {
	d := t.TempDir()
	os.MkdirAll(filepath.Join(d, "album", "sub"), 0o755)
	os.WriteFile(filepath.Join(d, "album", "sub", "c.jpg"), []byte("c"), 0o644)
	f := newFixture(t, nil)
	f.sess, _ = session.New([]string{filepath.Join(d, "album")})
	files := readZip(t, f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)).Body.Bytes())
	if string(files["album/sub/c.jpg"]) != "c" {
		t.Fatalf("entradas: %v", files)
	}
}

func TestZipSkipsVanishedFile(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10, "b.txt": 10, "c.txt": 10})
	os.Remove(f.sess.Files[1].Abs)
	files := readZip(t, f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)).Body.Bytes())
	if len(files) != 2 || files["a.txt"] == nil || files["c.txt"] == nil {
		t.Fatalf("entradas: %v", files)
	}
}

func TestZipEmptySessionIs404(t *testing.T) {
	f := newFixture(t, nil)
	if w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)); w.Code != http.StatusNotFound {
		t.Fatalf("%d", w.Code)
	}
}

// ZIP64 real: un archivo sparse de 5 GB no ocupa disco pero obliga a ZIP64.
func TestZip64Sparse(t *testing.T) {
	if testing.Short() {
		t.Skip("lee 5 GB de ceros; ~10 s")
	}
	d := t.TempDir()
	big := filepath.Join(d, "big.bin")
	fh, _ := os.Create(big)
	if err := fh.Truncate(5 << 30); err != nil {
		t.Skip("el FS no soporta sparse:", err)
	}
	fh.Close()
	f := newFixture(t, nil)
	f.sess, _ = session.New([]string{big})
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	resp, err := http.Get(ts.URL + f.sess.Path() + "/zip")
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := os.CreateTemp(d, "*.zip")
	defer tmp.Close() // en Windows un archivo abierto no se puede borrar al limpiar TempDir
	n, _ := io.Copy(tmp, resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(tmp, n)
	if err != nil || len(zr.File) != 1 || zr.File[0].UncompressedSize64 != 5<<30 {
		t.Fatalf("zip64 inválido: %v", err)
	}
}
