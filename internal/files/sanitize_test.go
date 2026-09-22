package files

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want string
		err      bool
	}{
		{"foto.jpg", "foto.jpg", false},
		{"../../etc/passwd", "passwd", false},
		{`..\..\Windows\system.ini`, "system.ini", false},
		{"C:\\Users\\x\\a.txt", "a.txt", false},
		{"/abs/path/b.pdf", "b.pdf", false},
		{"foto.jpg\x00.exe", "foto.jpg.exe", false},
		{"a\x01b\x1fc.txt", "abc.txt", false},
		{`we*i?rd"<na>me|.txt`, "weirdname.txt", false},
		{"dos:puntos.txt", "dospuntos.txt", false},
		{"año ñandú 日本.jpg", "año ñandú 日本.jpg", false},
		{"  espacios  .txt  ", "espacios  .txt", false},
		{"trailing.dots...", "trailing.dots", false},
		{"CON", "", true},
		{"con.txt", "", true},
		{"NUL.jpg", "", true},
		{"COM1", "", true},
		{"lpt9.log", "", true},
		{"CONSOLE.txt", "CONSOLE.txt", false},
		{"", "", true},
		{".", "", true},
		{"..", "", true},
		{"...", "", true},
		{"/", "", true},
		{"\x00\x01", "", true},
		{".bashrc", ".bashrc", false},
	}
	for _, c := range cases {
		got, err := Sanitize(c.in)
		if c.err {
			if !errors.Is(err, ErrBadName) {
				t.Errorf("Sanitize(%q) err = %v, want ErrBadName", c.in, err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Sanitize(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestSanitizeTruncatesKeepingExtension(t *testing.T) {
	in := strings.Repeat("a", 300) + ".mp4"
	got, err := Sanitize(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > 200 || !strings.HasSuffix(got, ".mp4") {
		t.Fatalf("len=%d suffix ok=%v", len(got), strings.HasSuffix(got, ".mp4"))
	}
}

func TestSanitizeTruncatesOnRuneBoundary(t *testing.T) {
	in := strings.Repeat("ñ", 150) + ".txt" // 300 bytes + 4
	got, err := Sanitize(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ".txt") || len(got) > 200 || !utf8Valid(got) {
		t.Fatalf("got %q (len %d)", got, len(got))
	}
}

func utf8Valid(s string) bool { return strings.ToValidUTF8(s, "\uFFFD") == s }
