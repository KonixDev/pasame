#!/usr/bin/env bash
# Uso: make-dmg.sh <Pasame.app> <salida.dmg>
set -euo pipefail
app="$1"; out="$2"
stage="$(mktemp -d)"
cp -R "$app" "$stage/"
ln -s /Applications "$stage/Aplicaciones"
rm -f "$out"
hdiutil create -volname Pasame -srcfolder "$stage" -ov -format UDZO "$out"
rm -rf "$stage"
