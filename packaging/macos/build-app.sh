#!/usr/bin/env bash
# Uso: build-app.sh <binario-universal> <versión> <dir-salida>
set -euo pipefail
bin="$1"; ver="${2#v}"; out="$3"
here="$(cd "$(dirname "$0")" && pwd)"
app="$out/Pasame.app"
rm -rf "$app"
osacompile -o "$app" "$here/droplet.applescript"
# osacompile genera su propio Info.plist y ejecutable "droplet": se reemplaza el plist por el nuestro.
sed "s/__VERSION__/$ver/g" "$here/Info.plist" > "$app/Contents/Info.plist"
cp "$here/icon.icns" "$app/Contents/Resources/icon.icns"
rm -f "$app/Contents/Resources/droplet.icns"
cp "$bin" "$app/Contents/Resources/pasame"
chmod 755 "$app/Contents/Resources/pasame"
# osacompile firmó el applet; al cambiar el plist y agregar archivos esa firma quedó rota. Re-firmar ad hoc
# (sin certificado): una firma rota en Apple Silicon es "la app está dañada". release-mac.sh la reemplaza
# por la de Developer ID cuando hay certificado.
codesign --force --deep -s - "$app" >/dev/null
echo "$app"
