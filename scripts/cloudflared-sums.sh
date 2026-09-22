#!/usr/bin/env bash
# Descarga los assets de cloudflared de la versión fijada y escribe el bloque Go con sus SHA-256.
# Uso: scripts/cloudflared-sums.sh 2026.9.1 > /tmp/sums.go.txt
set -euo pipefail
v="${1:?versión}"
base="https://github.com/cloudflare/cloudflared/releases/download/$v"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
emit() { # clave-go asset tgz
  curl -fsSL -o "$tmp/$2" "$base/$2"
  sum="$(shasum -a 256 "$tmp/$2" | cut -d' ' -f1)"
  printf '\t"%s": {Name: "%s", SHA256: "%s", Tgz: %s},\n' "$1" "$2" "$sum" "$3"
}
emit darwin/amd64  cloudflared-darwin-amd64.tgz  true
emit darwin/arm64  cloudflared-darwin-arm64.tgz  true
emit windows/amd64 cloudflared-windows-amd64.exe false
emit windows/arm64 cloudflared-windows-386.exe   false
emit linux/amd64   cloudflared-linux-amd64       false
emit linux/arm64   cloudflared-linux-arm64       false
emit linux/arm     cloudflared-linux-armhf       false
