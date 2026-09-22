# Matriz manual por release

Se completa con los archivos de la release, no con `make build`. Sin la matriz mínima completa no se publica una versión estable (spec §9.3). ✓ = funcionó · ✗ = falló (anotar qué pasó) · n/a = no aplica.

## Compartir en la red local

Emisor × receptor: Windows 11 + macOS actual + Ubuntu LTS como emisores; iPhone actual, iOS−2, Android actual, Android 9 y una PC como receptores.

| Receptor ↓ / Emisor → | Windows 11 | macOS actual | Ubuntu LTS |
|---|---|---|---|
| iPhone (iOS actual) | | | |
| iPhone (iOS−2) | | | |
| Android actual | | | |
| Android 9 | | | |
| PC con Windows (navegador) | n/a | ✓ (v0.1: Mac por WiFi → PC por cable) | |

## Instalación y primer arranque (con los assets de la release, no con `make build`)

| Chequeo | Windows 11 x64 | Windows 11 ARM | macOS 15 | macOS 12 | Ubuntu LTS (.deb) | Raspberry Pi OS |
|---|---|---|---|---|---|---|
| El botón del sitio baja el archivo correcto | | | | | | |
| Los pasos de como-abrir.html coinciden con lo que se ve | | | | | | |
| Sin notarizar: "Abrir de todos modos" aparece y funciona | n/a | n/a | | | n/a | n/a |
| Notarizado: abre sin ningún aviso | n/a | n/a | | | n/a | n/a |
| Soltar archivos sobre el ícono los comparte | n/a | n/a | | | n/a | n/a |
| El icono se ve bien (Dock/Finder, Explorador, lanzador) | | | | | | |
| Antivirus (Defender) no lo marca | | | n/a | n/a | n/a | n/a |
| `pasame.local` (solo builds con `-tags mdns`) | | | | | | |
