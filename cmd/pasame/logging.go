package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const maxLog = 5 << 20

// setupLogging escribe a stderr y a <config>/pasame.log (se trunca al superar 5 MB).
func setupLogging(dir string) {
	p := filepath.Join(dir, "pasame.log")
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, err := os.Stat(p); err == nil && fi.Size() > maxLog {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(p, flags, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("pasame: ")
}
