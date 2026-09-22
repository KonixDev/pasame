package platform

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Name          string `json:"name,omitempty"`
	IfaceOverride string `json:"iface_override,omitempty"`
	PortOverride  int    `json:"port_override,omitempty"`
	Lang          string `json:"lang,omitempty"`
}

func LoadConfig(dir string) (Config, bool, error) {
	var c Config
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return c, true, nil
	}
	if err != nil {
		return c, false, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		log.Printf("config.json ilegible, uso valores por defecto: %v", err)
		return Config{}, false, nil
	}
	return c, false, nil
}

func SaveConfig(dir string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "config.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "config.json"))
}
