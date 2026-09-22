package share

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestDownloadFull(t *testing.T) {
	f := newFixture(t, map[string]int{"año 2026.bin": 3_000_000})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), pattern(3_000_000)) {
		t.Fatalf("code %d, len %d", w.Code, w.Body.Len())
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(cd, "attachment;") ||
		!strings.Contains(cd, `filename*=UTF-8''a%C3%B1o%202026.bin`) ||
		!strings.Contains(cd, `filename="ao 2026.bin"`) {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if w.Header().Get("Content-Length") != "3000000" || w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("headers %v", w.Header())
	}
	if f.stats.Snapshot().Completed[0] != 1 {
		t.Fatal("no contó la descarga completa")
	}
}

func TestDownloadRange(t *testing.T) {
	f := newFixture(t, map[string]int{"v.mp4": 1_000_000})
	r := httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil)
	r.Header.Set("Range", "bytes=500000-")
	w := f.do(r)
	if w.Code != http.StatusPartialContent || w.Header().Get("Content-Range") != "bytes 500000-999999/1000000" {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Content-Range"))
	}
	if !bytes.Equal(w.Body.Bytes(), pattern(1_000_000)[500000:]) {
		t.Fatal("bytes distintos")
	}
}

func TestDownloadInline(t *testing.T) {
	f := newFixture(t, map[string]int{"foto.jpg": 10})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0?inline=1", nil))
	if !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline;") ||
		w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("%v", w.Header())
	}
}

func TestDownloadBadIndex(t *testing.T) {
	f := newFixture(t, map[string]int{"a": 1})
	// No hay caso "../": las URLs usan índices, así que un nombre de ruta nunca llega a os.Open.
	for _, p := range []string{"/f/1", "/f/-1", "/f/x", "/f/0x1", "/f/%2e%2e"} {
		if w := f.do(httptest.NewRequest("GET", f.sess.Path()+p, nil)); w.Code != http.StatusNotFound {
			t.Errorf("%s → %d", p, w.Code)
		}
	}
}

func TestDownloadFileVanished(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10})
	os.Remove(f.sess.Files[0].Abs)
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "ya no está disponible") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

func TestDownloadFileChangedSize(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10})
	os.WriteFile(f.sess.Files[0].Abs, []byte("otro contenido más largo"), 0o644)
	if w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil)); w.Code != http.StatusNotFound {
		t.Fatalf("%d", w.Code)
	}
}

func TestDownloadConcurrentMemory(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	f := newFixture(t, map[string]int{"big.bin": 20_000_000})
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + f.sess.Path() + "/f/0")
			if err != nil {
				t.Error(err)
				return
			}
			n, _ := io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if n != 20_000_000 {
				t.Errorf("n = %d", n)
			}
		}()
	}
	wg.Wait()
	runtime.ReadMemStats(&after)
	if grown := int64(after.HeapInuse) - int64(before.HeapInuse); grown > 50<<20 {
		t.Fatalf("el heap creció %d MB", grown>>20)
	}
}
