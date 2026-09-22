package share

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func multipartBody(t *testing.T, files map[string][]byte) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sortStrings(names)
	for _, n := range names {
		fw, _ := mw.CreateFormFile("f", n)
		fw.Write(files[n])
	}
	mw.Close()
	return &b, mw.FormDataContentType()
}

func (f *fixture) upload(t *testing.T, files map[string][]byte, jsonResp bool) *httptest.ResponseRecorder {
	body, ct := multipartBody(t, files)
	r := httptest.NewRequest("POST", f.sess.Path()+"/up", body)
	r.Header.Set("Content-Type", ct)
	if jsonResp {
		r.Header.Set("Accept", "application/json")
	}
	return f.do(r)
}

func listDir(t *testing.T, d string) []string {
	es, _ := os.ReadDir(d)
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestUploadThreeFilesNoJS(t *testing.T) {
	f := newFixture(t, nil)
	w := f.upload(t, map[string][]byte{"a.jpg": []byte("aa"), "b.mp4": []byte("bbb"), "c.pdf": []byte("c")}, false)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != f.sess.Path()+"?subido=3" {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Location"))
	}
	got := listDir(t, f.quar)
	if strings.Join(got, ",") != "a.jpg,b.mp4,c.pdf" {
		t.Fatalf("cuarentena: %v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(f.quar, "b.mp4")); string(b) != "bbb" {
		t.Fatal("contenido distinto")
	}
	if len(f.stats.Snapshot().Received) != 3 {
		t.Fatal("stats no registró las subidas")
	}
}

func TestUploadJSONAndSanitizeAndCollision(t *testing.T) {
	f := newFixture(t, nil)
	os.WriteFile(filepath.Join(f.quar, "evil.txt"), []byte("ya estaba"), 0o644)
	w := f.upload(t, map[string][]byte{"../../evil.txt": []byte("x")}, true)
	var resp struct {
		OK    bool
		Files []string
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if w.Code != 200 || !resp.OK || len(resp.Files) != 1 || resp.Files[0] != "evil (2).txt" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(f.quar, "evil.txt")); string(b) != "ya estaba" {
		t.Fatal("pisó un archivo existente")
	}
}

func TestUploadEmptyFile(t *testing.T) {
	f := newFixture(t, nil)
	w := f.upload(t, map[string][]byte{"vacio.txt": nil}, false)
	if w.Header().Get("Location") != f.sess.Path()+"?error=err_empty" || len(listDir(t, f.quar)) != 0 {
		t.Fatalf("%q %v", w.Header().Get("Location"), listDir(t, f.quar))
	}
}

type fullDisk struct{ n int }

func (d *fullDisk) Write(p []byte) (int, error) {
	if d.n+len(p) > 10 {
		return 0, &os.PathError{Op: "write", Path: "x", Err: syscall.ENOSPC}
	}
	d.n += len(p)
	return len(p), nil
}
func (d *fullDisk) Close() error { return nil }

func TestUploadDiskFull(t *testing.T) {
	f := newFixture(t, nil)
	f.srv.createPart = func(p string) (io.WriteCloser, error) {
		os.WriteFile(p, nil, 0o644) // que exista, para verificar que se borra
		return &fullDisk{}, nil
	}
	w := f.upload(t, map[string][]byte{"grande.mp4": bytes.Repeat([]byte("x"), 100)}, true)
	if w.Code != http.StatusInsufficientStorage || !strings.Contains(w.Body.String(), "no tiene espacio") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if len(listDir(t, f.quar)) != 0 {
		t.Fatalf("quedó basura: %v", listDir(t, f.quar))
	}
}

func TestUploadCutMidwayLeavesNoPart(t *testing.T) {
	f := newFixture(t, nil)
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, _ := mw.CreateFormFile("f", "cortado.mp4")
		fw.Write(bytes.Repeat([]byte("x"), 1<<20))
		pw.CloseWithError(errors.New("se cortó el WiFi"))
	}()
	req, _ := http.NewRequest("POST", ts.URL+f.sess.Path()+"/up", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	http.DefaultClient.Do(req) // falla del lado cliente: es lo esperado
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(listDir(t, f.quar)) == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("quedó: %v", listDir(t, f.quar))
}

func TestUploadLargeStreamsWithoutBuffering(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	f := newFixture(t, nil)
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	const size = 300 << 20
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, _ := mw.CreateFormFile("f", "video.mp4")
		chunk := bytes.Repeat([]byte("v"), 1<<20)
		for i := 0; i < size>>20; i++ {
			fw.Write(chunk)
		}
		mw.Close()
		pw.Close()
	}()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	req, _ := http.NewRequest("POST", ts.URL+f.sess.Path()+"/up", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", err, resp)
	}
	runtime.ReadMemStats(&after)
	if fi, _ := os.Stat(filepath.Join(f.quar, "video.mp4")); fi == nil || fi.Size() != size {
		t.Fatal("tamaño incorrecto")
	}
	if int64(after.HeapInuse)-int64(before.HeapInuse) > 50<<20 {
		t.Fatal("la subida se bufferizó en memoria")
	}
}
