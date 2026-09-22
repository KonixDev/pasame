// Package tunnel maneja cloudflared: bajarlo verificado, correrlo y leer su URL.
package tunnel

// Version se actualiza a mano junto con los SHA-256 (scripts/cloudflared-sums.sh).
// Nunca "latest": el comportamiento del túnel tiene que ser reproducible y testeable.
const Version = "2026.9.1"

type Asset struct {
	Name   string
	SHA256 string
	Tgz    bool
}

// SHA-256 verificados el 2026-09-21 contra el campo "digest" de la API de GitHub.
// windows/arm64 usa el binario x86 (Windows en ARM lo emula); linux/arm usa armhf (armv7).
var assets = map[string]Asset{
	"darwin/amd64":  {Name: "cloudflared-darwin-amd64.tgz", SHA256: "ff0d3b51d5ff70eceef89d6b32145fee985018a2174596a5dbe405e2766e2ac4", Tgz: true},
	"darwin/arm64":  {Name: "cloudflared-darwin-arm64.tgz", SHA256: "c27ab8fd0aa489449e3d201eb02f957ef460a13b613662928b1b23394bf1bcfe", Tgz: true},
	"windows/amd64": {Name: "cloudflared-windows-amd64.exe", SHA256: "2837888cc0f5d58f15b6dc478376de90b4d3ba5241c7947455d1e0a0df429712"},
	"windows/arm64": {Name: "cloudflared-windows-386.exe", SHA256: "11b6e4b2d306950bd87e7caa4deee8e80a32d71ffee555a96237a76651eeae4c"},
	"linux/amd64":   {Name: "cloudflared-linux-amd64", SHA256: "03f1f25d1cc93b9ad6c60569d44060bc4f17ed97075760ed8cfca4b12dcd68cc"},
	"linux/arm64":   {Name: "cloudflared-linux-arm64", SHA256: "3d97437c71848bd8df68041e12436b484a661d95073ea1937f01a845ce88faa3"},
	"linux/arm":     {Name: "cloudflared-linux-armhf", SHA256: "95420507a720fb543122a5d69372fbde8f5c919790e95ddd4374a449e0a6f4dd"},
}
