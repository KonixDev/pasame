package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	p, err := UniquePath(dir, "x.jpg")
	if err != nil || p != filepath.Join(dir, "x.jpg") {
		t.Fatalf("got %q %v", p, err)
	}
	os.WriteFile(p, nil, 0o644)
	p2, _ := UniquePath(dir, "x.jpg")
	if filepath.Base(p2) != "x (2).jpg" {
		t.Fatalf("got %q", p2)
	}
	os.WriteFile(p2, nil, 0o644)
	p3, _ := UniquePath(dir, "x.jpg")
	if filepath.Base(p3) != "x (3).jpg" {
		t.Fatalf("got %q", p3)
	}
	// Un .part en curso también cuenta como ocupado.
	os.WriteFile(filepath.Join(dir, "y.txt.part"), nil, 0o644)
	p4, _ := UniquePath(dir, "y.txt")
	if filepath.Base(p4) != "y (2).txt" {
		t.Fatalf("got %q", p4)
	}
}

func TestUniquePathNeverEscapes(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"../../evil", `..\..\evil`, "a/../../b"} {
		p, err := UniquePath(dir, n)
		if err != nil {
			continue
		}
		if filepath.Dir(p) != dir {
			t.Fatalf("%q escapó a %q", n, p)
		}
	}
	if _, err := UniquePath(dir, ".."); err == nil {
		t.Fatal("'..' debería fallar")
	}
}

func TestQuarantineDirIsCreated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DOWNLOAD_DIR", "")
	d, err := QuarantineDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(d) != "Pasame" {
		t.Fatalf("got %q", d)
	}
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		t.Fatalf("no se creó: %v", err)
	}
}

func TestMarkDownloadedDoesNotFail(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.exe")
	os.WriteFile(p, []byte("x"), 0o644)
	if err := MarkDownloaded(p); err != nil {
		t.Fatal(err)
	}
}
