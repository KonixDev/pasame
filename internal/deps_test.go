package internal

import (
	"os/exec"
	"strings"
	"testing"
)

const mod = "github.com/KonixDev/pasame/"

func deps(t *testing.T, pkg string) map[string]bool {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", mod+pkg).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", pkg, err)
	}
	m := map[string]bool{}
	for _, l := range strings.Fields(string(out)) {
		m[l] = true
	}
	return m
}

func imports(t *testing.T, pkg string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", `{{join .Imports " "}}`, mod+pkg).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(out))
}

func TestPurePackages(t *testing.T) {
	for _, p := range []string{"internal/session", "internal/files", "internal/qr", "internal/i18n"} {
		d := deps(t, p)
		for _, banned := range []string{"net/http", "os/exec"} {
			if d[banned] {
				t.Errorf("%s depende de %s", p, banned)
			}
		}
	}
}

func TestLayering(t *testing.T) {
	rules := map[string][]string{
		"internal/share": {"internal/control", "internal/core", "internal/addr", "internal/tunnel"},
		"internal/addr":  {"internal/control", "internal/share"},
	}
	for p, banned := range rules {
		d := deps(t, p)
		for _, b := range banned {
			if d[mod+b] {
				t.Errorf("%s no puede depender de %s", p, b)
			}
		}
	}
	for _, imp := range imports(t, "internal/control") {
		if strings.HasPrefix(imp, mod+"internal/") && imp != mod+"internal/core" {
			t.Errorf("control solo puede importar core; importa %s", imp)
		}
	}
}
