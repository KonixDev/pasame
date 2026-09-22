package tunnel

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	ErrUnavailable = errors.New("tunnel: no hay cloudflared para esta computadora")
	ErrChecksum    = errors.New("tunnel: el archivo descargado no coincide")
)

const releaseBase = "https://github.com/cloudflare/cloudflared/releases/download/"

type Progress func(done, total int64)

func Available() bool {
	_, ok := assets[runtime.GOOS+"/"+runtime.GOARCH]
	return ok
}

func Ensure(ctx context.Context, configDir string, p Progress) (string, error) {
	return ensure(ctx, configDir, runtime.GOOS, runtime.GOARCH, releaseBase, assets, p)
}

func binDir(configDir string) string { return filepath.Join(configDir, "cloudflared", Version) }

func binPath(configDir, goos string) string {
	name := "cloudflared"
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir(configDir), name)
}

func ensure(ctx context.Context, configDir, goos, goarch, base string, table map[string]Asset, p Progress) (string, error) {
	a, ok := table[goos+"/"+goarch]
	if !ok {
		return "", ErrUnavailable
	}
	dst := binPath(configDir, goos)
	if fi, err := os.Stat(dst); err == nil && fi.Size() > 0 {
		return dst, nil
	}
	if err := os.MkdirAll(binDir(configDir), 0o755); err != nil {
		return "", err
	}
	tmp, err := download(ctx, base+Version+"/"+a.Name, binDir(configDir), a.SHA256, p)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	if a.Tgz {
		err = extract(tmp, dst+".tmp")
	} else {
		err = os.Rename(tmp, dst+".tmp")
	}
	if err == nil {
		err = os.Chmod(dst+".tmp", 0o755)
	}
	if err == nil {
		err = os.Rename(dst+".tmp", dst)
	}
	if err != nil {
		os.Remove(dst + ".tmp")
		return "", err
	}
	if goos == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", dst).Run() // normalmente no lo tiene; se intenta igual
	}
	return dst, nil
}

// download baja url a un temporal en dir, calculando el SHA-256 mientras copia.
func download(ctx context.Context, url, dir, wantSHA string, p Progress) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tunnel: descarga respondió %s", resp.Status)
	}
	f, err := os.CreateTemp(dir, "dl-*")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	pr := &progressReader{r: resp.Body, total: resp.ContentLength, fn: p}
	_, err = io.Copy(io.MultiWriter(f, h), pr)
	f.Close()
	if err == nil && hex.EncodeToString(h.Sum(nil)) != wantSHA {
		err = ErrChecksum
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func extract(tgzPath, dst string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err != nil {
			return fmt.Errorf("tunnel: el .tgz no trae cloudflared: %w", err)
		}
		if filepath.Base(h.Name) != "cloudflared" || h.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, io.LimitReader(tr, 200<<20))
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		return err
	}
}

type progressReader struct {
	r     io.Reader
	done  int64
	total int64
	fn    Progress
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.fn != nil && n > 0 {
		p.fn(p.done, p.total)
	}
	return n, err
}
