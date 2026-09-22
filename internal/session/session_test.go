package session

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func write(t *testing.T, p string, size int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewWithFilesAndFolder(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "a.jpg"), 10)
	write(t, filepath.Join(d, "album", "b.jpg"), 20)
	write(t, filepath.Join(d, "album", "sub", "c.mp4"), 30)
	write(t, filepath.Join(d, "album", ".DS_Store"), 1)
	write(t, filepath.Join(d, "album", "Thumbs.db"), 1)
	write(t, filepath.Join(d, "album", "desktop.ini"), 1)
	write(t, filepath.Join(d, "album", ".oculta", "x.txt"), 1)
	if runtime.GOOS != "windows" {
		os.Symlink(filepath.Join(d, "a.jpg"), filepath.Join(d, "album", "link.jpg"))
	}

	s, errs := New([]string{filepath.Join(d, "a.jpg"), filepath.Join(d, "album")})
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	var rels []string
	for i, f := range s.Files {
		if f.Index != i {
			t.Fatalf("Index %d en posición %d", f.Index, i)
		}
		rels = append(rels, f.Rel)
	}
	want := []string{"a.jpg", "album/b.jpg", "album/sub/c.mp4"}
	if len(rels) != len(want) {
		t.Fatalf("rels = %v, want %v", rels, want)
	}
	for i := range want {
		if rels[i] != want[i] {
			t.Fatalf("rels = %v, want %v", rels, want)
		}
	}
	if s.Files[2].Name != "c.mp4" || s.TotalSize() != 60 {
		t.Fatalf("Name=%q total=%d", s.Files[2].Name, s.TotalSize())
	}
	if s.Path() != "/s/"+s.Token || len(s.PIN) != 4 {
		t.Fatalf("Path=%q PIN=%q", s.Path(), s.PIN)
	}
}

func TestNewReportsUnreadable(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "ok.txt"), 1)
	s, errs := New([]string{filepath.Join(d, "ok.txt"), filepath.Join(d, "no-existe.mp4")})
	if len(s.Files) != 1 || len(errs) != 1 {
		t.Fatalf("files=%d errs=%v", len(s.Files), errs)
	}
}

func TestNewEmptyIsReceiveOnly(t *testing.T) {
	s, errs := New(nil)
	if s == nil || len(s.Files) != 0 || len(errs) != 0 {
		t.Fatalf("s=%v errs=%v", s, errs)
	}
}

func TestFileLookup(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "a"), 1)
	s, _ := New([]string{filepath.Join(d, "a")})
	if _, ok := s.File(0); !ok {
		t.Fatal("File(0) debería existir")
	}
	for _, i := range []int{-1, 1, 99} {
		if _, ok := s.File(i); ok {
			t.Fatalf("File(%d) no debería existir", i)
		}
	}
}
