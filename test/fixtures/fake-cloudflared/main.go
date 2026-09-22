// cloudflared falso para tests: imprime logs parecidos a los reales y una URL.
// FAKE_MODE: "ok" (espera hasta que lo maten), "nourl" (nunca da URL), "die" (da URL y muere al segundo).
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	if os.Getenv("FAKE_ARGS_FILE") != "" {
		os.WriteFile(os.Getenv("FAKE_ARGS_FILE"), []byte(strings.Join(os.Args[1:], " ")), 0o644)
	}
	log := func(s string) { fmt.Fprintf(os.Stderr, "2026-09-21T12:00:00Z INF %s\n", s) }
	log("Thank you for trying Cloudflare Tunnel.")
	log("Requesting new quick Tunnel on trycloudflare.com...")
	fmt.Fprintln(os.Stderr, "2026-09-21T12:00:00Z WRN Cannot determine default configuration path. No file [config.yml config.yaml]")
	mode := os.Getenv("FAKE_MODE")
	if mode != "nourl" {
		log("+--------------------------------------------------------------------------------------------+")
		log("|  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |")
		log("|  https://amber-cat-dream-yellow.trycloudflare.com                                           |")
		log("+--------------------------------------------------------------------------------------------+")
	}
	if mode == "die" {
		time.Sleep(time.Second)
		os.Exit(1)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
