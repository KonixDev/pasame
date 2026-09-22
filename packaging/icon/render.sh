#!/usr/bin/env bash
# Genera .icns (macOS), .ico (Windows) y .png (Linux) desde el SVG. Requiere: brew install librsvg imagemagick
set -euo pipefail
cd "$(dirname "$0")"
set_dir=pasame.iconset
rm -rf "$set_dir" && mkdir "$set_dir"
for s in 16 32 64 128 256 512 1024; do
  src=pasame.svg; [ $s -lt 24 ] && src=../../docs/marca/isotipo-reducido.svg
  rsvg-convert -w $s -h $s "$src" -o "png-$s.png"
done
for s in 16 32 128 256 512; do
  cp "png-$s.png" "$set_dir/icon_${s}x${s}.png"
  cp "png-$((s*2)).png" "$set_dir/icon_${s}x${s}@2x.png"
done
iconutil -c icns "$set_dir" -o ../macos/icon.icns
magick png-16.png png-32.png png-64.png png-128.png png-256.png ../windows/icon.ico
mkdir -p ../linux && cp png-256.png ../linux/icon.png
rm -rf "$set_dir" png-*.png
