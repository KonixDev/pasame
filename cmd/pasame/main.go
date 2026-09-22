// Pasame: pasá archivos a cualquier celular o computadora que esté cerca.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/browser"
	"github.com/KonixDev/pasame/internal/control"
	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/internal/files"
	"github.com/KonixDev/pasame/internal/lifecycle"
	"github.com/KonixDev/pasame/internal/platform"
	"github.com/KonixDev/pasame/internal/share"
)

var version = "dev"

func cleanArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func main() {
	if err := run(); err != nil {
		log.Printf("error fatal: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfgDir, err := platform.ConfigDir()
	if err != nil {
		return err
	}
	setupLogging(cfgDir)
	log.Printf("versión %s", version)
	paths := cleanArgs(os.Args[1:])

	lock, err := lifecycle.Acquire(cfgDir)
	if errors.Is(err, lifecycle.ErrHeld) {
		return delegate(cfgDir, paths)
	}
	if err != nil {
		return err
	}
	defer lock.Release()

	cfg, firstRun, err := platform.LoadConfig(cfgDir)
	if err != nil {
		return err
	}
	quarantine, err := files.QuarantinePath() // no se crea acá: ver QuarantinePath
	if err != nil {
		return err
	}

	port := 8080
	if cfg.PortOverride > 0 {
		port = cfg.PortOverride
	}
	shareL, err := share.Listen("0.0.0.0", port)
	if err != nil {
		return err
	}
	sharePort := shareL.Addr().(*net.TCPAddr).Port

	lan := addr.NewLAN(sharePort)
	book := addr.NewBook()
	book.Register(lan)
	if m, err := addr.NewMDNS(lan, sharePort); err == nil { // solo en builds con -tags mdns
		book.Register(m)
		defer m.Close()
	}
	c := core.New(core.Options{
		SharePort: sharePort, Quarantine: quarantine, ConfigDir: cfgDir,
		Config: cfg, FirstRun: firstRun, LAN: lan, Book: book, Version: version,
	})

	shareH, err := share.New(share.Deps{
		Current: c.Current, Stats: c.Stats, Quarantine: quarantine, Sender: c.Sender, Strict: c.Strict, Lang: c.Lang,
		PINKey: c.PINKey,
	})
	if err != nil {
		return err
	}
	shareSrv := share.HTTPServer(shareH)

	ctlL, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctlPort := ctlL.Addr().(*net.TCPAddr).Port
	token := control.NewToken()
	ctlSrv := &http.Server{Handler: control.New(c, token, ctlPort), ReadHeaderTimeout: 10 * time.Second}

	go serve(shareSrv, shareL)
	go serve(ctlSrv, ctlL)
	if err := lock.Publish(lifecycle.Info{Port: ctlPort, Token: token, PID: os.Getpid()}); err != nil {
		return err
	}
	log.Printf("compartir en :%d, control en 127.0.0.1:%d", sharePort, ctlPort)

	if len(paths) > 0 {
		c.Share(paths)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go c.Run(ctx)
	browser.Open(fmt.Sprintf("http://127.0.0.1:%d/?t=%s", ctlPort, token))

	select {
	case <-ctx.Done():
	case <-c.Done():
	}
	log.Print("cerrando")
	c.RequestQuit() // avisa a las pestañas abiertas (evento SSE "quit")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c.DisableTunnel() // no dejar cloudflared andando
	ctlSrv.Shutdown(sctx)
	shareSrv.Shutdown(sctx)
	if shareH.Received() {
		cleanParts(quarantine) // solo si se usó: no pedir permiso de Descargas al cerrar
	}
	return nil
}

func serve(s *http.Server, l net.Listener) {
	if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("servidor: %v", err)
	}
}

// delegate: ya hay una instancia; se le pasan los archivos o se reabre su UI.
func delegate(cfgDir string, paths []string) error {
	info, err := lifecycle.ReadInfo(cfgDir)
	if err != nil {
		return fmt.Errorf("otra instancia corre pero no publicó sus datos: %w", err)
	}
	url, err := lifecycle.Handoff(info, paths)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		browser.Open(url)
	}
	return nil
}

func cleanParts(dir string) {
	ms, _ := filepath.Glob(filepath.Join(dir, "*.part"))
	for _, m := range ms {
		os.Remove(m)
	}
}
