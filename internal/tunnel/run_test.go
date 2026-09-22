package tunnel

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var fakeBin string

func TestMain(m *testing.M) {
	dir, _ := os.MkdirTemp("", "fakecf")
	fakeBin = filepath.Join(dir, "cloudflared")
	if runtime.GOOS == "windows" {
		fakeBin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", fakeBin, "../../test/fixtures/fake-cloudflared").CombinedOutput()
	if err != nil {
		panic(string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func okVerify(context.Context, string) error { return nil }

func TestParseURL(t *testing.T) {
	cases := map[string]string{
		"2026-09-21T12:00:00Z INF |  https://amber-cat-dream-yellow.trycloudflare.com   |":  "https://amber-cat-dream-yellow.trycloudflare.com",
		"INF Requesting new quick Tunnel on trycloudflare.com...":                           "",
		"ERR failed to request quick Tunnel: Post \"https://api.trycloudflare.com/tunnel\"": "",
		"https://evil.com/?x=https://a.trycloudflare.com.evil.com":                          "",
		"https://a-b-3.trycloudflare.com":                                                   "https://a-b-3.trycloudflare.com",
	}
	for line, want := range cases {
		got, ok := ParseURL(line)
		if got != want || ok != (want != "") {
			t.Errorf("ParseURL(%q) = %q,%v; want %q", line, got, ok, want)
		}
	}
}

func TestStartOKAndStop(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("FAKE_ARGS_FILE", argsFile)
	t.Setenv("FAKE_MODE", "ok")
	tun, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: 5 * time.Second, Verify: okVerify})
	if err != nil {
		t.Fatal(err)
	}
	if tun.URL != "https://amber-cat-dream-yellow.trycloudflare.com" {
		t.Fatalf("URL = %q", tun.URL)
	}
	args, _ := os.ReadFile(argsFile)
	want := "tunnel --url http://127.0.0.1:8080 --no-autoupdate --config " + nullConfig() + " --loglevel info"
	if string(args) != want {
		t.Fatalf("args = %q\nwant  %q", args, want)
	}
	tun.Stop()
	select {
	case <-tun.Done():
	case <-time.After(7 * time.Second):
		t.Fatal("Stop no terminó el proceso")
	}
}

func TestStartTimeoutWithoutURL(t *testing.T) {
	t.Setenv("FAKE_MODE", "nourl")
	start := time.Now()
	_, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: time.Second, Verify: okVerify})
	if err == nil || time.Since(start) > 4*time.Second {
		t.Fatalf("err=%v en %v", err, time.Since(start))
	}
	if !strings.Contains(err.Error(), "Requesting new quick Tunnel") {
		t.Fatalf("el error no trae el log para 'Ver detalle': %v", err)
	}
}

func TestStartVerifyFails(t *testing.T) {
	t.Setenv("FAKE_MODE", "ok")
	bad := func(context.Context, string) error { return errors.New("todavía no responde") }
	if _, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: time.Second, Verify: bad}); err == nil {
		t.Fatal("mostró una URL que no verificó")
	}
}

func TestChildDiesOnItsOwn(t *testing.T) {
	t.Setenv("FAKE_MODE", "die")
	tun, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: 5 * time.Second, Verify: okVerify})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-tun.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("no se enteró de que el proceso murió")
	}
}

func TestIsolatedHome(t *testing.T) {
	// Un ~/.cloudflared/config.yaml del usuario rompe los quick tunnels: el hijo corre con HOME vacío.
	env := childEnv("/tmp/pasame-cf-home")
	var home, profile bool
	for _, e := range env {
		home = home || e == "HOME=/tmp/pasame-cf-home"
		profile = profile || e == "USERPROFILE=/tmp/pasame-cf-home"
	}
	if !home || !profile {
		t.Fatalf("env sin HOME/USERPROFILE aislados")
	}
}
