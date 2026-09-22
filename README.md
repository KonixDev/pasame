<p align="center">
  <img src="docs/marca/logo.svg" alt="Pasame" width="280">
</p>

<p align="center"><b>Pasá archivos a cualquier celular o computadora que esté cerca.</b><br>Sin cable, sin cuentas, sin instalar nada del lado de quien recibe.</p>

<p align="center">
  <a href="https://github.com/KonixDev/pasame/releases"><img alt="Descargar" src="https://img.shields.io/github/v/release/KonixDev/pasame?include_prereleases&label=descargar&color=2F5D2E"></a>
  <a href="https://github.com/KonixDev/pasame/stargazers"><img alt="Estrellas" src="https://img.shields.io/github/stars/KonixDev/pasame?style=flat&color=A7C957"></a>
  <a href="LICENSE"><img alt="Licencia MIT" src="https://img.shields.io/badge/licencia-MIT-1B2A1A"></a>
</p>

---

En 2026, pasar un video del celular a la compu de al lado sigue siendo un lío: cables, apps que comprimen, cuentas, mails a uno mismo. **Pasame** lo resuelve con lo que ya tiene cualquier dispositivo, que es un navegador.

1. Abrís Pasame en tu computadora y elegís los archivos.
2. La otra persona apunta la cámara del celular al código que aparece (o escribe la dirección en su navegador).
3. Listo: descarga lo que compartiste y, si quiere, te manda archivos de vuelta.

> **Una persona tiene Pasame en una computadora. La otra, solo un navegador.**
> Pueden entrar varias a la vez: una comparte, muchas reciben, como en una ronda de mate.

<table>
<tr>
<td width="68%"><img src="docs/img/emisor.png" alt="Pantalla de Pasame en la computadora: código QR, la dirección 192.168.1.40:8080 y la actividad"></td>
<td width="32%"><img src="docs/img/receptor.png" alt="Página que ve quien recibe en el celular: Descargar todo, la lista de archivos y el botón para mandar archivos de vuelta"></td>
</tr>
<tr>
<td align="center"><sub>En la computadora que comparte</sub></td>
<td align="center"><sub>En el celular que recibe</sub></td>
</tr>
</table>

## Descargar

Bajá la versión para tu computadora desde **[Releases](https://github.com/KonixDev/pasame/releases/latest)**:

| Tu computadora | Archivo |
|---|---|
| Windows 10 u 11 | `Pasame-Windows.exe` (o `Pasame-Windows-arm64.exe` en equipos ARM) |
| Mac (macOS 12 o más nuevo, Intel o Apple) | `Pasame-macOS.zip` |
| Linux (incluida Raspberry Pi) | `pasame-linux-amd64.tar.gz`, `-arm64` o `-armv7` |

No hace falta instalar nada: es un solo archivo que se abre con doble clic. Del lado de quien recibe **no se instala nada**: sirve cualquier navegador de 2017 en adelante, incluidos iPhone, Android, Smart TVs y e-readers.

### La primera vez que lo abrís

Pasame todavía no está firmado por Microsoft ni por Apple, así que la primera vez tu computadora avisa. Se hace una sola vez.

- **Windows:** si aparece "Windows protegió tu PC", tocá **Más información** → **Ejecutar de todas formas**. Si aparece una ventana del "Firewall de Windows Defender", tocá **Permitir acceso**: es para que el celular pueda ver tu computadora.
- **Mac:** descomprimí el `.zip`, mové **Pasame** a Aplicaciones y abrilo. Si dice que no se puede abrir, andá a **Ajustes del Sistema → Privacidad y seguridad**, bajá hasta el final y tocá **Abrir de todos modos**.
- **Linux:** descomprimí y ejecutá `./pasame`. Para elegir archivos con una ventana hace falta `zenity` o `kdialog` (vienen en GNOME y KDE).

## Lo que conviene saber

- **Los dos dispositivos tienen que estar en la misma red WiFi** (o uno por cable al mismo router).
- Si están en el WiFi de un bar, un hotel o un aeropuerto, esas redes suelen bloquear esto. Compartir por internet llega en la próxima versión.
- Si cerrás la pestaña de Pasame, la app se cierra sola a los 2 minutos (si no hay nada bajando).
- Lo que te mandan queda en **Descargas → Pasame**.
- Las descargas se retoman solas si se corta el WiFi, y "Descargar todo" arma un ZIP al vuelo, sin esperar ni ocupar espacio extra.

## Qué no promete

En tu WiFi, cualquiera conectado a esa misma red que tenga el link puede ver lo que compartís mientras la app está abierta. Los archivos no salen de tu red, pero no viajan cifrados dentro de ella. No uses Pasame en una red en la que no confíes.

## Estado

**v0.1 · red local.** Funciona: compartir archivos y carpetas, código QR y dirección corta, varios receptores a la vez, ZIP al vuelo, descargas reanudables, mandar archivos de vuelta (también arrastrándolos a la página), idioma español e inglés, instancia única.

Próximo (ver [plan](docs/superpowers/plans/2026-09-21-pasame-02-tunel-y-distribucion.md)): compartir por internet con clave de 4 números, app de Mac firmada por Apple, instaladores y el sitio [pasame.com.ar](https://pasame.com.ar).

## Para desarrolladores

Go 1.27, un solo binario sin dependencias del sistema (`CGO_ENABLED=0`), dos dependencias directas ([zenity](https://github.com/ncruces/zenity) para los diálogos nativos y [go-qrcode](https://github.com/skip2/go-qrcode)). La interfaz es HTML, CSS y JavaScript sin frameworks, embebida en el binario; la página de quien recibe funciona incluso sin JavaScript.

```bash
make test     # tests unitarios y de integración (go test -race)
make build    # ./pasame
make cross    # compila los 7 targets (Windows, macOS, Linux; x64 y ARM)
./pasame foto.jpg video.mp4   # abre compartiendo esos archivos
```

| Carpeta | Qué hay |
|---|---|
| `cmd/pasame` | El binario: arma todo, abre el navegador, maneja el cierre. |
| `internal/share` | Lo que ve quien recibe (puerto 8080): página, descargas, ZIP, subidas. |
| `internal/control` | La interfaz de quien comparte, solo en `127.0.0.1` y protegida con un token. |
| `internal/core` | Orquesta la sesión, las direcciones y la regla de cierre automático. |
| `internal/addr` | Elige la red correcta (ignora VPN, Docker y similares). |
| `web/` | HTML, CSS y JS de las dos pantallas. |
| `docs/marca` | [Manual de marca](docs/marca/manual-de-marca.html), logos y colores. |
| `docs/superpowers` | Diseño técnico y planes de implementación. |

Las decisiones de diseño y sus porqués están en el [documento de diseño](docs/superpowers/specs/2026-09-21-share-now-design.md). Los dos criterios que mandan en todo el proyecto: **que funcione en la mayor cantidad de dispositivos posible** y **que lo pueda usar cualquier persona, sin ayuda**.

## Si te sirvió

Dale una ⭐ al repo: ayuda a que más gente lo encuentre. Y si algo no anduvo, [abrí un issue](https://github.com/KonixDev/pasame/issues) contando qué computadora y qué celular usaste.

---

<p align="center">Creado por <a href="https://martincoll.dev">martincoll.dev</a> · Licencia <a href="LICENSE">MIT</a></p>
