package lifecycle

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

// TestMain permite que el binario de test haga de "segunda instancia".
func TestMain(m *testing.M) {
	switch os.Getenv("PASAME_LOCK_CHILD") {
	case "try":
		_, err := Acquire(os.Getenv("PASAME_LOCK_DIR"))
		if errors.Is(err, ErrHeld) {
			os.Exit(3)
		}
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "hold-and-die":
		if _, err := Acquire(os.Getenv("PASAME_LOCK_DIR")); err != nil {
			os.Exit(1)
		}
		os.Exit(0) // sin Release: simula un cierre abrupto
	}
	os.Exit(m.Run())
}

func child(t *testing.T, mode, dir string) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "PASAME_LOCK_CHILD="+mode, "PASAME_LOCK_DIR="+dir)
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0
}

func TestSecondInstanceIsDetected(t *testing.T) {
	dir := t.TempDir()
	l, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if code := child(t, "try", dir); code != 3 {
		t.Fatalf("la segunda instancia no vio el lock (exit %d)", code)
	}
	l.Release()
	if code := child(t, "try", dir); code != 0 {
		t.Fatalf("después de Release no pudo tomarlo (exit %d)", code)
	}
}

func TestOrphanLockIsRecovered(t *testing.T) {
	dir := t.TempDir()
	if code := child(t, "hold-and-die", dir); code != 0 {
		t.Fatalf("exit %d", code)
	}
	l, err := Acquire(dir)
	if err != nil {
		t.Fatalf("lock de un proceso muerto no se recuperó: %v", err)
	}
	l.Release()
}

func TestPublishAndRead(t *testing.T) {
	dir := t.TempDir()
	l, _ := Acquire(dir)
	defer l.Release()
	want := Info{Port: 53211, Token: "abc", PID: os.Getpid()}
	if err := l.Publish(want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadInfo(dir)
	if err != nil || got != want {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestHandoff(t *testing.T) {
	var gotPaths []string
	var gotToken, gotHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken, gotHost = r.Header.Get("X-Pasame-Token"), r.Host
		var b struct{ Paths []string }
		json.NewDecoder(r.Body).Decode(&b)
		gotPaths = b.Paths
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	info := Info{Port: port, Token: "tok"}

	url, err := Handoff(info, []string{"/a.jpg", "/b.mp4"})
	if err != nil || len(gotPaths) != 2 || gotToken != "tok" || gotHost != srv.Listener.Addr().String() {
		t.Fatalf("%v %v %q %q", err, gotPaths, gotToken, gotHost)
	}
	if url != srv.URL+"/?t=tok" {
		t.Fatalf("url = %q", url)
	}
	gotPaths = nil
	if _, err := Handoff(info, nil); err != nil || gotPaths != nil {
		t.Fatal("sin rutas no debería llamar a la API")
	}
}
