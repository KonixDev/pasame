package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, first, err := LoadConfig(dir)
	if err != nil || !first || c != (Config{}) {
		t.Fatalf("%+v %v %v", c, first, err)
	}
	want := Config{Name: "Martín", IfaceOverride: "192.168.1.42", PortOverride: 9000}
	if err := SaveConfig(dir, want); err != nil {
		t.Fatal(err)
	}
	got, first, err := LoadConfig(dir)
	if err != nil || first || got != want {
		t.Fatalf("%+v %v %v", got, first, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json.tmp")); err == nil {
		t.Fatal("quedó el temporal")
	}
}

func TestLoadConfigCorruptIsNotFatal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{no es json"), 0o644)
	c, first, err := LoadConfig(dir)
	if err != nil || first || c != (Config{}) {
		t.Fatalf("un config roto no debe impedir arrancar: %+v %v %v", c, first, err)
	}
}

func TestConfigDirEndsInPasame(t *testing.T) {
	d, err := ConfigDir()
	if err != nil {
		t.Skip("sin UserConfigDir en este entorno:", err)
	}
	if filepath.Base(d) != "Pasame" {
		t.Fatalf("%q", d)
	}
}

func TestOpenCommand(t *testing.T) {
	name, args := openCommand("/tmp/x")
	want := map[string]string{"darwin": "open", "windows": "explorer", "linux": "xdg-open"}[runtime.GOOS]
	if name != want || len(args) != 1 || args[0] != "/tmp/x" {
		t.Fatalf("%s %v", name, args)
	}
}
