#!/usr/bin/env bash
# Arma los archivos de una release en dist/. Uso: scripts/release.sh 0.1.0
# Corre en macOS (necesita lipo y ditto para el .app universal).
set -euo pipefail
ver="${1:?versión, ej. 0.1.0}"
root="$(cd "$(dirname "$0")/.." && pwd)"
out="$root/dist"
rm -rf "$out" && mkdir -p "$out/tmp"
build() { # os arch salida [ldflags extra]
  local extra="${4:-}"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" GOARM=7 go build -trimpath -ldflags="-s -w -X main.version=$ver $extra" -o "$3" "$root/cmd/pasame"
}
cd "$root"
go test ./...

# Windows: sin consola (-H=windowsgui).
build windows amd64 "$out/Pasame-Windows.exe" "-H=windowsgui"
build windows arm64 "$out/Pasame-Windows-arm64.exe" "-H=windowsgui"

# macOS: binario universal dentro de Pasame.app.
build darwin amd64 "$out/tmp/pasame-amd64"
build darwin arm64 "$out/tmp/pasame-arm64"
lipo -create -output "$out/tmp/pasame-universal" "$out/tmp/pasame-amd64" "$out/tmp/pasame-arm64"
packaging/macos/build-app.sh "$out/tmp/pasame-universal" "$ver" "$out/tmp" >/dev/null
(cd "$out/tmp" && ditto -c -k --keepParent Pasame.app "$out/Pasame-macOS.zip")

# Linux: x64, arm64 y armv7 (Raspberry Pi de 32 bits).
for a in amd64 arm64 arm; do
  name="pasame-linux-$a"; [ "$a" = arm ] && name="pasame-linux-armv7"
  mkdir -p "$out/tmp/$name"
  build linux "$a" "$out/tmp/$name/pasame"
  cp README.md LICENSE "$out/tmp/$name/"
  tar -C "$out/tmp" -czf "$out/$name.tar.gz" "$name"
done

rm -rf "$out/tmp"
(cd "$out" && shasum -a 256 * > checksums.txt)
ls -la "$out"
