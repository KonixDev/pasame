// Package lifecycle garantiza una sola instancia y le pasa el trabajo a la que ya corre.
package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ErrHeld = errors.New("lifecycle: otra instancia está corriendo")

type Info struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

type Lock struct {
	f   *os.File
	dir string
}

func Acquire(dir string) (*Lock, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "pasame.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := tryLock(f); err != nil {
		f.Close()
		return nil, ErrHeld
	}
	return &Lock{f: f, dir: dir}, nil
}

func (l *Lock) Publish(i Info) error {
	b, _ := json.Marshal(i)
	tmp := filepath.Join(l.dir, "pasame.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil { // 0600: el token no lo lee otro usuario
		return err
	}
	return os.Rename(tmp, filepath.Join(l.dir, "pasame.json"))
}

func (l *Lock) Release() error {
	os.Remove(filepath.Join(l.dir, "pasame.json"))
	unlock(l.f)
	return l.f.Close()
}

func ReadInfo(dir string) (Info, error) {
	var i Info
	b, err := os.ReadFile(filepath.Join(dir, "pasame.json"))
	if err != nil {
		return i, err
	}
	return i, json.Unmarshal(b, &i)
}
