package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var urlRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com\b`)

// ParseURL busca la URL del quick tunnel en una línea de log. "api." es el endpoint de Cloudflare, no el túnel.
func ParseURL(line string) (string, bool) {
	for _, u := range urlRe.FindAllString(line, -1) {
		if end := strings.Index(line, u) + len(u); end < len(line) && line[end] == '.' {
			continue // "…trycloudflare.com.evil.com"
		}
		if u != "https://api.trycloudflare.com" {
			return u, true
		}
	}
	return "", false
}

type Options struct {
	Bin     string
	Port    int
	Timeout time.Duration
	Verify  func(ctx context.Context, url string) error
}

type Tunnel struct {
	URL  string
	cmd  *exec.Cmd
	done chan struct{}
	home string

	mu  sync.Mutex
	log []string
}

func childEnv(home string) []string {
	return append(os.Environ(), "HOME="+home, "USERPROFILE="+home)
}

func Start(ctx context.Context, o Options) (*Tunnel, error) {
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.Verify == nil {
		o.Verify = verifyHealthz
	}
	home, err := os.MkdirTemp("", "pasame-cf-")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(o.Bin, "tunnel", "--url", "http://127.0.0.1:"+strconv.Itoa(o.Port),
		"--no-autoupdate", "--config", nullConfig(), "--loglevel", "info")
	cmd.Env = childEnv(home)
	hideConsole(cmd)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(home)
		return nil, err
	}
	t := &Tunnel{cmd: cmd, done: make(chan struct{}), home: home}
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			t.addLog(line)
			if u, ok := ParseURL(line); ok {
				select {
				case found <- u:
				default:
				}
			}
		}
	}()
	go func() {
		cmd.Wait()
		os.RemoveAll(home)
		close(t.done)
	}()

	deadline, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	select {
	case t.URL = <-found:
	case <-t.done:
		return nil, fmt.Errorf("cloudflared terminó sin dar una dirección:\n%s", t.LastLog())
	case <-deadline.Done():
		t.Stop()
		return nil, fmt.Errorf("cloudflared no dio una dirección a tiempo:\n%s", t.LastLog())
	}
	// Los quick tunnels tardan unos segundos en propagarse: reintentar hasta el plazo.
	for {
		err := o.Verify(deadline, t.URL)
		if err == nil {
			return t, nil
		}
		select {
		case <-deadline.Done():
			t.Stop()
			return nil, fmt.Errorf("la dirección %s no respondió: %v", t.URL, err)
		case <-t.done:
			return nil, fmt.Errorf("cloudflared se cerró:\n%s", t.LastLog())
		case <-time.After(time.Second):
		}
	}
}

// verifyHealthz pide <url>/healthz resolviendo el nombre con DNS públicos y no con el de la red: la
// dirección recién creada tarda unos segundos en existir, y el "no existe" de la primera consulta queda
// guardado en el DNS del router, lo que haría fallar también a los celulares de esa misma red.
func verifyHealthz(ctx context.Context, url string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/healthz", nil)
	resp, err := verifyClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("respondió %s", resp.Status)
	}
	return nil
}

var publicDNS = []string{"1.1.1.1:53", "8.8.8.8:53"}

var verifyClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{Timeout: 10 * time.Second, Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, system string) (net.Conn, error) {
				d := net.Dialer{Timeout: 2 * time.Second}
				for _, addr := range publicDNS {
					if c, err := d.DialContext(ctx, network, addr); err == nil {
						return c, nil
					}
				}
				return d.DialContext(ctx, network, system) // red que bloquea DNS externos: el de la red
			},
		}}).DialContext,
	},
}

func (t *Tunnel) Done() <-chan struct{} { return t.done }

func (t *Tunnel) Stop() {
	select {
	case <-t.done:
		return
	default:
	}
	interrupt(t.cmd)
	select {
	case <-t.done:
	case <-time.After(stopGrace):
		kill(t.cmd)
		<-t.done
	}
}

func (t *Tunnel) addLog(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.log = append(t.log, line)
	if len(t.log) > 30 {
		t.log = t.log[len(t.log)-30:]
	}
}

func (t *Tunnel) LastLog() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(t.log, "\n")
}
