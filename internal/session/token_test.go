package session

import (
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		tok := NewToken()
		if len(tok) != 5 {
			t.Fatalf("len(%q) = %d", tok, len(tok))
		}
		for _, r := range tok {
			if !strings.ContainsRune(Alphabet, r) {
				t.Fatalf("%q tiene %q fuera del alfabeto", tok, r)
			}
		}
		seen[tok] = true
	}
	if len(seen) < 1990 {
		t.Fatalf("solo %d tokens distintos de 2000", len(seen))
	}
}

func TestNewPIN(t *testing.T) {
	for i := 0; i < 500; i++ {
		p := NewPIN()
		if len(p) != 4 || strings.Trim(p, "0123456789") != "" {
			t.Fatalf("PIN inválido %q", p)
		}
	}
}
