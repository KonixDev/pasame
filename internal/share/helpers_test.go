package share

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/KonixDev/pasame/internal/session"
)

type fixture struct {
	srv    *Server
	sess   *session.Session
	stats  *session.Stats
	dir    string // archivos compartidos
	quar   string // cuarentena
	strict bool
}

// newFixture crea archivos con los tamaños dados y una sesión que los comparte.
func newFixture(t *testing.T, sizes map[string]int) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), quar: t.TempDir(), stats: session.NewStats(nil)}
	var paths []string
	for name, n := range sizes {
		p := filepath.Join(f.dir, name)
		if err := os.WriteFile(p, pattern(n), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	sortStrings(paths)
	f.sess, _ = session.New(paths)
	srv, err := New(Deps{
		Current:    func() *session.Session { return f.sess },
		Stats:      func() *session.Stats { return f.stats },
		Quarantine: f.quar,
		Sender:     func() string { return "Martín" },
		Strict:     func() bool { return f.strict },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.srv = srv
	return f
}

func (f *fixture) do(r *http.Request) *httptest.ResponseRecorder {
	if r.RemoteAddr == "" || r.RemoteAddr == "192.0.2.1:1234" {
		r.RemoteAddr = "192.168.1.50:5555"
	}
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	return w
}

// pattern genera bytes deterministas para comparar descargas.
func pattern(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
