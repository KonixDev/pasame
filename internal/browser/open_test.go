package browser

import (
	"runtime"
	"testing"
)

func TestCommand(t *testing.T) {
	u := "http://127.0.0.1:5555/?t=abc"
	name, args := command(u)
	switch runtime.GOOS {
	case "darwin":
		if name != "open" || args[0] != u {
			t.Fatal(name, args)
		}
	case "windows":
		if name != "rundll32" || args[0] != "url.dll,FileProtocolHandler" || args[1] != u {
			t.Fatal(name, args)
		}
	default:
		if name != "xdg-open" || args[0] != u {
			t.Fatal(name, args)
		}
	}
}
