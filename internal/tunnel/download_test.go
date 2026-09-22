package tunnel

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestPinnedAssets(t *testing.T) {
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	// Los 7 targets de Pasame tienen túnel.
	for _, k := range []string{"darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64", "linux/amd64", "linux/arm64", "linux/arm"} {
		a, ok := assets[k]
		if !ok || !hex64.MatchString(a.SHA256) {
			t.Errorf("%s: asset o SHA-256 faltante (%+v)", k, a)
		}
	}
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func tgz(t *testing.T, name string, content []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))})
	tw.Write(content)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

type fakeRelease struct {
	srv  *httptest.Server
	hits atomic.Int32
}

func serve(t *testing.T, files map[string][]byte) *fakeRelease {
	f := &fakeRelease{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		for name, b := range files {
			if r.URL.Path == "/"+Version+"/"+name {
				w.Header().Set("Content-Length", strconv.Itoa(len(b)))
				w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func TestEnsurePlainBinary(t *testing.T) {
	bin := []byte("#!/bin/sh\necho falso\n")
	rel := serve(t, map[string][]byte{"cloudflared-linux-amd64": bin})
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum(bin)}}
	dir := t.TempDir()
	var calls int
	p, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, func(d, tot int64) { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	fi, _ := os.Stat(p)
	executable := runtime.GOOS == "windows" || fi.Mode()&0o111 != 0 // Windows no tiene bit de ejecución
	if !bytes.Equal(got, bin) || !executable || calls == 0 {
		t.Fatalf("contenido/permisos/progreso mal: exec=%v calls=%d", fi.Mode(), calls)
	}
	// Segunda vez: ya está, no descarga.
	before := rel.hits.Load()
	if _, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, nil); err != nil || rel.hits.Load() != before {
		t.Fatal("volvió a descargar")
	}
}

func TestEnsureTgz(t *testing.T) {
	bin := []byte("binario mac falso")
	archive := tgz(t, "cloudflared", bin)
	rel := serve(t, map[string][]byte{"cloudflared-darwin-arm64.tgz": archive})
	tbl := map[string]Asset{"darwin/arm64": {Name: "cloudflared-darwin-arm64.tgz", SHA256: sum(archive), Tgz: true}}
	p, err := ensure(context.Background(), t.TempDir(), "darwin", "arm64", rel.srv.URL+"/", tbl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); !bytes.Equal(got, bin) {
		t.Fatal("no extrajo el binario del .tgz")
	}
}

func TestEnsureChecksumMismatch(t *testing.T) {
	rel := serve(t, map[string][]byte{"cloudflared-linux-amd64": []byte("adulterado")})
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum([]byte("original"))}}
	dir := t.TempDir()
	_, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(binPath(dir, "linux")); err == nil {
		t.Fatal("dejó el binario adulterado")
	}
	if left, _ := os.ReadDir(binDir(dir)); len(left) != 0 {
		t.Fatalf("dejó temporales: %v", left)
	}
}

func TestEnsureUnavailable(t *testing.T) {
	_, err := ensure(context.Background(), t.TempDir(), "freebsd", "amd64", "http://x/", assets, nil)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureHTTPError(t *testing.T) {
	rel := serve(t, nil)
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum(nil)}}
	if _, err := ensure(context.Background(), t.TempDir(), "linux", "amd64", rel.srv.URL+"/", tbl, nil); err == nil {
		t.Fatal("un 404 no dio error")
	}
}
