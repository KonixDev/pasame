package lifecycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handoff le pasa las rutas a la instancia viva y devuelve la URL de su UI.
func Handoff(i Info, paths []string) (string, error) {
	base := fmt.Sprintf("http://127.0.0.1:%d", i.Port)
	url := base + "/?t=" + i.Token
	if len(paths) == 0 {
		return url, nil
	}
	body, _ := json.Marshal(map[string][]string{"paths": paths})
	req, _ := http.NewRequest("POST", base+"/api/add", bytes.NewReader(body))
	req.Header.Set("X-Pasame-Token", i.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return url, err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return url, fmt.Errorf("la instancia abierta respondió %d", resp.StatusCode)
	}
	return url, nil
}
