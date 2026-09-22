# Checklist del túnel real (cloudflared 2026.9.1)

Se completa con los archivos de la release, no con `make build`. ✓ = funcionó · ✗ = falló (anotar qué pasó) · vacío = falta probar.

Primera pasada: 2026-09-22 · Emisor: Mac (Apple Silicon, macOS 26), build de la rama `plan-02` · Receptor: `curl` desde la misma Mac, pasando por Cloudflare.

| # | Chequeo | Resultado |
|---|---|---|
| 1 | Con `~/.cloudflared/config.yaml` presente (crear uno de prueba), "Compartir por internet" igual funciona (HOME aislado + `--config`). | |
| 2 | Primera vez: se ve "Preparando…" con barra y termina en menos de 2 min con conexión normal. | ✓ (descarga + conexión en unos 30 s) |
| 3 | Segunda vez: no vuelve a descargar (pasa directo a "Conectando…"). | ✓ |
| 4 | La URL aparece en < 30 s y el QR cambia solo cuando `/healthz` responde. | ✓ 11–15 s en 4 intentos |
| 5 | Celular en datos móviles escanea el QR y entra sin tipear la clave. | parcial: el link del QR (`?pin=`) entra y deja la cookie (probado con `curl`); falta un celular real |
| 6 | Tipear la dirección en otra PC pide la clave; 5 errores → "Demasiados intentos"; al minuto se puede de nuevo. | ✓ pide la clave y bloquea; falta cronometrar el minuto |
| 7 | Descarga de un video de 2 GB por el túnel completa (anotar velocidad). | parcial: 3 MB idénticos byte a byte; falta el de 2 GB |
| 8 | Subida de 3 fotos desde el celular por el túnel llega a `Descargas/Pasame`. | |
| 9 | "Volver a compartir solo por WiFi": el link de internet deja de responder; en la LAN ya no pide clave. | ✓ (Cloudflare responde 530; LAN 200) |
| 10 | `pkill cloudflared` con el túnel activo → aviso "Se cortó la conexión por internet". | ✓ |
| 11 | Cerrar Pasame con el túnel activo no deja un `cloudflared` vivo (`ps aux | grep cloudflared`). | ✓ |
| 12 | Windows ARM: el túnel funciona con `cloudflared-windows-386.exe` por emulación. | |
| 13 | Raspberry Pi 32 bits: el túnel funciona con `cloudflared-linux-armhf`. | |
| 14 | Windows: al compartir por internet **no** se abre ninguna ventana negra de consola. | |
| 15 | Desde la LAN, tipear `192.168.x.x:8080` con el túnel activo → "Para entrar necesitás el código QR o el link de quien comparte."; la dirección que muestra el emisor (con `/s/…`) sí entra y pide la clave. | ✓ |

## Pendientes que no puede hacer un agente

- **Capturas para `site/como-abrir.html`** (y `site/en/how-to-open.html`), en máquinas reales con la próxima release: "Windows protegió tu PC" + "Más información", "Ejecutar de todas formas", la ventana del Firewall de Windows Defender, el aviso de macOS "no se puede abrir" y el botón "Abrir de todos modos". Recortar a la ventana, 1200 px de ancho máximo, guardar en `site/img/` y agregar los `<img>` en cada paso. `TestComoAbrir` verifica que existan los archivos que se nombran.
- **`pasame.local`** (solo builds con `-tags mdns`): en la Mac de la primera pasada el firewall de macOS estaba apagado, así que la regla de corte (ningún diálogo de firewall extra) todavía no se evaluó. Probar con el firewall prendido y en Windows 11.
- **Dominio `pasame.com.ar`**: registrarlo en NIC.ar; en Cloudflare DNS, cuatro `A` a `185.199.108.153`, `185.199.109.153`, `185.199.110.153` y `185.199.111.153` más `CNAME www → konixdev.github.io`, con el proxy apagado (nube gris); después crear `site/CNAME` con `pasame.com.ar` y poner el dominio en GitHub → Settings → Pages.
