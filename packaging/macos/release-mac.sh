#!/usr/bin/env bash
# Arma Pasame.app y el .dmg desde el binario universal de GoReleaser; firma y notariza si hay certificado.
# Uso: release-mac.sh <versión>   (ej. 1.2.3). Corre en macOS.
set -euo pipefail
ver="${1#v}"
here="$(cd "$(dirname "$0")" && pwd)"
out=dist/mac
mkdir -p "$out"
"$here/build-app.sh" dist/macos_darwin_all/pasame "$ver" "$out" >/dev/null
app="$out/Pasame.app"
dmg=dist/Pasame-macOS.dmg
notarized=false

if [[ -n "${MACOS_CERT_P12:-}" ]]; then
  kc="$RUNNER_TEMP/pasame.keychain-db"; kcpass="$(uuidgen)"
  security create-keychain -p "$kcpass" "$kc"
  security set-keychain-settings -lut 3600 "$kc"
  security unlock-keychain -p "$kcpass" "$kc"
  echo "$MACOS_CERT_P12" | base64 --decode > "$RUNNER_TEMP/cert.p12"
  security import "$RUNNER_TEMP/cert.p12" -k "$kc" -P "$MACOS_CERT_PASSWORD" -T /usr/bin/codesign
  security set-key-partition-list -S apple-tool:,apple: -s -k "$kcpass" "$kc"
  security list-keychains -d user -s "$kc" login.keychain-db

  sign() { codesign --force --options runtime --timestamp --entitlements "$here/entitlements.plist" --sign "$MACOS_SIGN_IDENTITY" "$1"; }
  # De adentro hacia afuera: el binario Go, el ejecutable del droplet y por último el bundle.
  sign "$app/Contents/Resources/pasame"
  sign "$app/Contents/MacOS/droplet"
  sign "$app"
  codesign --verify --deep --strict --verbose=2 "$app"

  "$here/make-dmg.sh" "$app" "$dmg"
  sign "$dmg"
  echo "$MACOS_NOTARY_KEY" | base64 --decode > "$RUNNER_TEMP/notary.p8"
  xcrun notarytool submit "$dmg" --key "$RUNNER_TEMP/notary.p8" \
    --key-id "$MACOS_NOTARY_KEY_ID" --issuer "$MACOS_NOTARY_ISSUER_ID" --wait
  xcrun stapler staple "$dmg"
  spctl -a -t open --context context:primary-signature -vv "$dmg"   # Expected: accepted, source=Notarized Developer ID
  notarized=true
else
  echo "Sin certificado de Apple: el .dmg sale sin firmar (beta en Mac)."
  "$here/make-dmg.sh" "$app" "$dmg"
fi

printf '{"version":"%s","macNotarized":%s}\n' "$ver" "$notarized" > dist/latest.json
