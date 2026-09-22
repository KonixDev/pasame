# Diseño técnico — "Pasame" (nombre provisorio de carpeta: `share-now`)

**Fecha:** 2026-09-21
**Estado:** diseño aprobado para pasar a plan de implementación
**Autor:** arquitectura (sesión de brainstorming + este documento)

---

## 0. Resumen ejecutivo

**Qué es:** una herramienta para pasar archivos entre dispositivos que están en el mismo lugar, sin cable, sin cuentas, sin instalar nada del lado de quien recibe. La persona que comparte abre un programa, elige archivos y aparece un QR. La que recibe escanea el QR con la cámara del celular (o tipea una dirección corta en cualquier navegador) y descarga. La misma página le permite mandar archivos de vuelta.

**Los dos objetivos que mandan** (todo lo demás es secundario):

1. **Máximo alcance de dispositivos.**
2. **Máxima simpleza de UI/UX** (una persona de 60 años, sin ayuda).

**Decisiones clave de este documento** (cada una está justificada en su sección):

| # | Decisión | Por qué, en una línea |
|---|---|---|
| 1 | **Go**, binario único estático, sin CGO | Un solo archivo que se hace doble clic; cross-compile a 6 targets desde una máquina; stdlib HTTP con streaming y Range gratis. |
| 2 | **La UI del emisor es una página web servida en `127.0.0.1`** que el binario abre en el navegador por defecto | Cero capa de GUI nativa, una sola tecnología de UI para emisor y receptor, cross-platform gratis; el navegador ya está en todas las máquinas. |
| 3 | **Dos listeners separados**: control (`127.0.0.1`, puerto aleatorio) y compartir (`0.0.0.0:8080`) | Con el túnel activo, `cloudflared` corre local y sus requests llegan **desde loopback**: un único listener con chequeo "¿es local?" abriría la UI de control a internet. |
| 4 | **La página del receptor funciona sin JavaScript** (HTML + links + `<form>`); JS solo agrega barras de progreso | Cubre Smart TVs, e-readers, navegadores viejos y modos restringidos sin escribir código aparte. |
| 5 | **Selección de archivos con diálogo nativo del OS** (`ncruces/zenity`, sin CGO) + argumentos de línea de comando | El navegador no puede darle rutas al servidor; el diálogo nativo sí. Arrastrar sobre el ícono cae gratis por argv en Windows/Linux. |
| 6 | **Elegir archivos = empezar a compartir.** No hay botón "Iniciar" | Un click menos. 3 clicks del emisor, 2 toques del receptor, hasta que el archivo baja. |
| 7 | **Cuando el túnel está activo, toda la sesión exige PIN** (no solo la subida) | Regla de una sola línea, explicable a cualquiera; el QR lleva el PIN adentro así nadie lo tipea. |
| 8 | **Celular como emisor: solo vía dirección inversa** (celular → PC que tiene la app). Sin app móvil en v1 | iOS no permite servidores en segundo plano; una app nativa duplica el proyecto. La dirección inversa cubre el 90 % del caso. |
| 9 | **macOS necesita notarización (US$ 99/año) para ser usable por no técnicos.** Windows sale sin firma con una página de "cómo abrir" | Verificado: macOS 15+ obliga a ir a Ajustes → Privacidad y seguridad → "Abrir de todos modos". Sin firma, el objetivo 2 se rompe en Mac. |
| 10 | **Nombre recomendado: "Pasame"** | Es literalmente lo que se dice ("pasame ese archivo"), se pronuncia en inglés, `github.com/<usuario>/pasame` está libre. |

**Stack:** Go 1.27 (stdlib `net/http`, `archive/zip`, `embed`) + 2 dependencias directas (`ncruces/zenity` para diálogos nativos, `skip2/go-qrcode` para QR) + `cloudflared` descargado bajo demanda. HTML/CSS/JS vanilla embebido, sin framework ni build step de frontend.

**Matriz de OS (resumen):** emisor = Windows 10/11 x64/arm64, macOS 12+ (binario universal), Linux x64/arm64 (incl. Raspberry Pi). Receptor = cualquier navegador de 2017 en adelante en cualquier OS, incluidos iOS 12+, Android 5+, Smart TVs y e-readers con navegador.

**Riesgos principales:** (1) Gatekeeper/SmartScreen frenan al usuario no técnico antes de que la app arranque; (2) AP isolation en WiFi público no se puede detectar de forma fiable, solo se puede sugerir el túnel a tiempo; (3) elección de interfaz de red equivocada con VPN; (4) dependencia de un servicio gratuito de terceros (TryCloudflare) sin SLA; (5) el receptor cierra la pestaña del emisor creyendo que cerró la app.

---

## 1. Cómo se usan los dos objetivos como criterio de decisión

Cada decisión de este documento se evaluó con estas dos preguntas, en este orden:

1. **¿Suma o resta dispositivos?** Cualquier cosa que requiera instalar algo en el receptor, un navegador específico, o una versión de OS reciente, resta. Cualquier cosa que dependa de HTTP/1.1 y HTML básico, suma.
2. **¿Suma o resta pasos, decisiones o palabras técnicas para el usuario?** Cada pantalla se escribió pensando en alguien que no sabe qué es una IP, un puerto, un firewall ni un QR (sabe "apuntar la cámara").

Cuando hubo conflicto entre esto y la elegancia técnica, ganó esto. Ejemplos concretos de trade-offs aceptados:

- HTTP plano en LAN (sin candado) en lugar de HTTPS autofirmado: la pantalla roja del navegador mata el objetivo 2.
- La página del receptor es "fea" a propósito: HTML del 2010, botones enormes, sin animaciones. Funciona en un Kindle.
- IP cruda en pantalla (`192.168.1.42:8080`) en lugar de un dominio lindo: funciona en todos lados, no depende de nada.

---

## 2. Decisiones heredadas del brainstorming (se construye sobre ellas)

Se listan para que el documento sea autocontenido. No se relitigan.

- **Modelo "servidor efímero 1:N"**: el emisor levanta un servidor HTTP local; N receptores lo visitan con un navegador. El QR es el descubrimiento (sin mDNS/multicast para descubrir pares).
- **Alcance v1**: LAN directa + dirección inversa (subida desde el receptor) + túnel Cloudflare opt-in (quick tunnel, sin cuenta, `cloudflared` descargado la primera vez).
- **5 componentes**: UI, Sesión, Servidor HTTP, Direcciones (costura clave, con proveedores enchufables), QR.
- **Seguridad en función del transporte**: túnel exige PIN de 4 dígitos, LAN no. PIN embebido en el QR. Subidas a carpeta de cuarentena fija con nombres saneados.
- **UX**: HTTP plano en LAN; IP cruda gigante para PC + hostname `.local` como bonus opcional; QR para celular; sin acortadores públicos.
- **Técnica**: streaming por chunks con HTTP Range; ZIP generado al vuelo.
- **Modos de falla a contemplar**: VPN elige IP equivocada, AP isolation, firewall del OS, VLAN de invitados, Android apaga WiFi.

---

## 3. Decisiones nuevas

### 3.1 Lenguaje y runtime → **Go**

La pregunta real no es "qué lenguaje es más lindo" sino **"¿qué tiene que hacer una persona no técnica para que esto corra?"**. Eso descarta cualquier cosa que requiera un runtime aparte.

| Opción | Instalación para el usuario | Cross-compile | Tamaño | Firma/notarización | Riesgo técnico | Veredicto |
|---|---|---|---|---|---|---|
| **Go** (stdlib + 2 libs puras) | Doble clic en un archivo. Nada más. | `GOOS`/`GOARCH` desde cualquier máquina, sin toolchains nativos. Binario universal de macOS con `lipo`. | ~8–10 MB estimado (estimación: binario de `net/http` + templates embebidas + QR; verificar al primer build) | Mismo que cualquier binario nativo: firma opcional, funciona sin ella. | Muy bajo. `net/http` maneja streaming, Range (`http.ServeContent`), multipart, HTTP/1.1 keep-alive. `archive/zip` escribe a un `io.Writer` (streaming). | **Elegido** |
| Rust + Tauri | Doble clic. En Windows necesita WebView2 (viene en Win 11 y Win 10 actualizados; si falta, el instalador lo baja). En Linux necesita WebKitGTK instalado. | Requiere toolchain por target; en la práctica se compila en CI por OS. | 5–10 MB | Igual. | Medio. Más superficie (IPC, webview por plataforma), tiempo de desarrollo mayor por el borrow checker en código de red. | Segunda opción válida. Pierde por costo de desarrollo y por la dependencia de webview del sistema en Linux. |
| Node + Electron | Doble clic, pero descarga de 100–200 MB. | Solo empaquetando por OS con `electron-builder`. | 100–200 MB | Igual. | Bajo. | Descartado: 20× el tamaño para hacer lo mismo. |
| Python | "Instalá Python primero" o empaquetar con PyInstaller (30–60 MB, antivirus falsos positivos frecuentes). | PyInstaller solo compila en el OS destino. | 30–60 MB | Igual. | Medio (streaming y concurrencia con `asyncio` es más frágil que en Go). | Descartado: la instalación rompe el objetivo 2. |
| Go + GUI nativa (Fyne/Wails) | Doble clic. | Requiere CGO (OpenGL o webview) → cross-compile con Docker o CI por OS. | 20–30 MB | Igual. | Medio: CGO complica todo. | Descartado: ver 3.2, no hace falta GUI nativa. |

**Lo que Go da gratis y que este proyecto necesita:**

- `http.ServeContent(w, r, name, modtime, file)` implementa Range, `If-Range`, `Last-Modified` y `Content-Type` correctos. Un `*os.File` se sirve sin cargarlo en memoria. Esto resuelve el requisito no negociable de streaming y descargas reanudables sin código propio.
- `archive/zip` escribe directamente al `http.ResponseWriter`: ZIP al vuelo con un `for` de 10 líneas, y ZIP64 automático para más de 4 GB.
- `embed` mete el HTML/CSS/JS en el binario. Un solo archivo para distribuir.
- Un goroutine por request: 20 personas bajando un video a la vez no requiere ningún diseño de concurrencia adicional.
- `os/exec` para `cloudflared`, `osascript`, `xdg-open`.

**Restricciones que impone Go** (y que se aceptan):

- Go 1.27 requiere **Windows 10+** y **macOS 12+** (Go dejó de soportar Windows 7/8 en 1.21 y macOS 10.15 en 1.23). Se acepta: son sistemas sin soporte del fabricante.
- **Regla: CGO deshabilitado (`CGO_ENABLED=0`)** en todo el proyecto. Cualquier dependencia que lo requiera queda descartada. Es lo que hace el cross-compile trivial.

**Versión:** Go 1.27 (verificado: 1.27.1 es la estable al 2026-09-21). Se fija en `go.mod` y en `.github/workflows`.

**Dependencias directas, exactamente dos:**

| Módulo | Para qué | Estado verificado |
|---|---|---|
| `github.com/ncruces/zenity` | Diálogo nativo "elegir archivos / carpeta". Sin CGO: Win32 API por `syscall` en Windows, `osascript` en macOS, `zenity`/`qarma`/`matedialog` en Linux. | Activo (último push 2026-08), 900+ estrellas. |
| `github.com/skip2/go-qrcode` | Render del QR a PNG/SVG. Pura Go. | Sin cambios desde 2024-03 (3000+ estrellas). Es una librería "terminada": el estándar QR no cambia. Aceptable. Si se prefiere, se reemplaza con 1 archivo; la interfaz `qr.SVG(url string) string` la aísla. |

Opcional (ver 4.5.3): `github.com/hashicorp/mdns` para anunciar `pasame.local`. Solo si se implementa el bonus; se puede cortar sin tocar nada más.

`cloudflared` **no** es una dependencia de compilación: se descarga en tiempo de ejecución desde GitHub Releases (ver 4.6).

### 3.2 Forma de la UI del emisor → **binario que abre el navegador local con la UI**

| Forma | Objetivo 1 (alcance) | Objetivo 2 (simpleza) | Costo | Veredicto |
|---|---|---|---|---|
| CLI | Alta (corre en todo) | **Nula.** Una terminal es el fin de la conversación con un no técnico. | Bajo | Descartado como interfaz principal. Queda como *modo avanzado* gratis (`pasame archivo.pdf` en terminal funciona porque argv es la misma ruta que arrastrar sobre el ícono). |
| GUI nativa (Fyne, Wails, Tauri) | Alta pero con CGO/webview del sistema. | Alta: "se ve como una app". | Alto: dos códigos de UI (nativa para el emisor, HTML para el receptor), CGO, cross-compile en CI por OS. | Descartado por costo. |
| App de menubar/tray | Igual que la anterior + necesita CGO en macOS/Linux para el tray. | Media: los íconos de tray son invisibles para mucha gente ("¿dónde está la app?"). | Alto | Descartado. |
| **Binario que abre el navegador con la UI en `127.0.0.1`** | **Máxima**: solo depende de que haya un navegador, que hay en toda máquina que pueda correr el binario. | **Alta**, con dos problemas a resolver (abajo). | **Mínimo**: una sola tecnología de UI (HTML) para emisor y receptor, sin CGO, cross-compile trivial. | **Elegido** |

**Cómo funciona:** al hacer doble clic, el binario (a) levanta el listener de control en `127.0.0.1:<puerto aleatorio>`, (b) genera un token de sesión de control, (c) abre el navegador por defecto en `http://127.0.0.1:<puerto>/?t=<token>` (`open` en macOS, `rundll32 url.dll,FileProtocolHandler` en Windows, `xdg-open` en Linux). La pestaña que se abre *es* la app.

**Los dos problemas de esta forma, y su solución:**

1. **"Cerré la pestaña y pensé que cerré la app."** Solución: la página de control manda un *heartbeat* (SSE abierto; cuando se corta, el servidor lo nota en ≤ 5 s). Regla de ciclo de vida:
   - Si no hay pestaña de control conectada **y** no hay transferencias en curso durante **2 minutos** → el proceso termina solo.
   - Si hay transferencias en curso → espera a que terminen y después aplica la regla.
   - Si el usuario vuelve a hacer doble clic mientras el proceso sigue vivo → el segundo proceso detecta el *lock file* (ver 4.10), abre el navegador en la UI del primero y termina. El usuario ve "su" app de nuevo. Nunca ve dos.
2. **Ventana de consola parásita.** En Windows se compila con `-ldflags="-H=windowsgui"`: sin consola. En macOS el binario se distribuye dentro de un `.app` con `LSUIElement=true` en `Info.plist`: doble clic no abre Terminal ni muestra ícono en el Dock (la "app" es la pestaña). En Linux se distribuye con un `.desktop` que lo lanza con `Terminal=false`.

**Arrastrar un archivo encima del ícono:**

- **Windows y Linux:** gratis. Arrastrar sobre el `.exe`/`.desktop` ejecuta el binario con las rutas como argumentos. `pasame.exe foto.jpg video.mp4` → abre la UI ya compartiendo esos archivos, sin diálogo. También habilita "Abrir con → Pasame" y "Enviar a → Pasame" (el usuario lo agrega con un acceso directo en `shell:sendto`; no se automatiza en v1).
- **macOS:** un binario común dentro de un `.app` **no** recibe archivos arrastrados (macOS los manda como Apple Event `odoc`, no por argv). Solución: el `.app` es un *droplet* de AppleScript compilado con `osacompile` en CI (runner macOS gratuito para repos públicos). Su handler `on open` ejecuta el binario en `Contents/Resources/pasame` con las rutas POSIX como argv, desacoplado (`do shell script "... &> /dev/null &"`). Su handler `on run` (doble clic sin archivos) lo ejecuta sin argumentos. Son ~15 líneas de AppleScript. Es la única pieza no-Go del proyecto y se marca como **recortable**: si da problemas, v1 en macOS sale sin arrastrar-sobre-ícono y con el diálogo nativo como único camino.

### 3.3 Matriz de OS soportados

**Emisor** (el que corre el binario):

| OS | Versión mínima | Arquitecturas | Formato de entrega | Notas |
|---|---|---|---|---|
| Windows | 10 (build 1809+) y 11 | x64, arm64 | `Pasame.exe` portable (un archivo, sin instalador) | Diálogo de firewall al primer uso (ver 6.3). SmartScreen sin firma (ver 10). |
| macOS | 12 Monterey+ | Universal (Intel + Apple Silicon en un solo binario) | `Pasame.app` dentro de `.dmg` | Sin notarizar, macOS 15+ exige Ajustes → Privacidad → "Abrir de todos modos" (verificado). Ver 10. |
| Linux | Cualquier distro con kernel 3.2+ y `glibc` irrelevante (binario estático) | x64, arm64, armv7 | `tar.gz` con binario + `.desktop`, y `.deb` | Necesita un navegador, `xdg-open`, y `zenity` o `kdialog` para el diálogo de archivos (GNOME y KDE los traen). Si no hay diálogo, la UI muestra un campo para pegar la ruta. |
| Raspberry Pi OS | 64-bit y 32-bit | arm64, armv7 | Mismo `tar.gz` de Linux | Cae gratis del cross-compile. |

**Explícitamente fuera como emisor:** Windows 7/8/8.1, macOS 10.x/11, 32-bit x86 en Windows y macOS, ChromeOS (salvo dentro de su contenedor Linux, que funciona pero no se documenta ni testea), iOS, iPadOS, Android (ver 3.4).

**Receptor** (el que solo abre un navegador):

| Plataforma | Navegador mínimo | Qué funciona | Limitaciones conocidas |
|---|---|---|---|
| iPhone / iPad | Safari en iOS 12+ | Escaneo de QR con la cámara nativa (iOS 11+), descarga a Files (iOS 13+ tiene gestor de descargas), subida de fotos/videos desde la galería con `<input type=file multiple>`, descompresión de ZIP nativa en Files (iOS 13+). | En iOS 12 no hay gestor de descargas: los archivos se abren inline y se guardan con "Compartir". Las fotos se guardan en Fotos si se abren inline (por eso existe el botón "Ver", ver 6.4). |
| Android | Chrome 60+ / Samsung Internet / Firefox | Escaneo de QR con la cámara (Android 9+ nativo; antes, Google Lens o cualquier app de QR). Descarga a la carpeta Descargas. Subida desde galería o archivos. | En Android 5–8 la cámara no lee QR nativamente: se tipea la dirección. |
| Windows / macOS / Linux / ChromeOS | Cualquier navegador de 2017+ (Chrome 60, Firefox 60, Safari 11, Edge Chromium) | Todo. | Ninguna. |
| Smart TV (Samsung Tizen, LG webOS, Android TV) | El navegador que traiga | Ver la página, reproducir video/audio inline con `<video>`/`<audio>` (streaming con Range). | La mayoría **no puede descargar archivos**. Se documenta como "ver, no guardar". Tipear la dirección con el control remoto es tedioso pero posible. |
| E-readers (Kindle, Kobo) | Navegador "experimental" | Ver la página, descargar `.epub`/`.pdf`/`.txt` en Kobo. Kindle descarga `.txt`/`.azw`. | Kindle: sin JS moderno. Por eso la página base no necesita JS. |
| Consolas (PS5, Xbox, Switch) | Navegador oculto o limitado | Ver la página. | Sin descarga real. Anecdótico; no se testea. |

**Compromiso de compatibilidad de la página del receptor:** HTML5 + CSS que degrada (sin `grid` obligatorio, sin variables CSS obligatorias) + JavaScript ES5 opcional. Sin frameworks, sin `fetch` obligatorio (`XMLHttpRequest` para la barra de progreso de subida), sin módulos ES. Tamaño de la página < 30 KB total.

### 3.4 ¿Puede el celular ser emisor? → **Solo vía dirección inversa en v1**

Los casos reales de "celular como emisor":

| Caso | ¿Cubierto en v1? | Cómo |
|---|---|---|
| Celular → PC con la app | **Sí** | Dirección inversa: la PC comparte (aunque sea sin archivos: botón "Solo recibir"), el celular escanea y sube fotos/videos desde la página. Cero instalación en el celular. |
| Celular → PC de otra persona que **no** tiene la app | No | Requiere que el celular corra el servidor. |
| Celular → celular | No | Ídem. |
| Celular → Smart TV | No | Ídem. |

**Opciones evaluadas para cubrir los "No":**

- **App nativa Android/iOS**: duplica el proyecto en otro lenguaje (Kotlin/Swift) o mete un framework multiplataforma (Flutter, React Native) que no comparte código con Go. En iOS, además, el sistema suspende la app a los pocos segundos de ir a segundo plano: un servidor HTTP que muere cuando bloqueás la pantalla es peor que no tenerlo. Publicar en App Store requiere cuenta de desarrollador (US$ 99/año) y revisión.
- **PWA**: un navegador no puede escuchar conexiones entrantes. No existe la primitiva. Descartado de plano.
- **Binario Go en Android** (gomobile, Termux): técnicamente posible, no distribuible a no técnicos.

**Decisión:** v1 no tiene emisor móvil. Se agrega el modo **"Solo recibir"** en la PC (un click, sin elegir archivos) para que el caso "mandá las fotos del celu a mi compu" sea de 1 click + 1 escaneo. Se documenta explícitamente en el README: *"Una de las dos personas tiene que tener la app en una computadora. La otra, solo un navegador."* Una app Android nativa que reutilice el servidor Go vía `gomobile` es el candidato natural para v2 si hay demanda; el diseño del servidor (sin estado global, sin dependencias de OS de escritorio en `internal/share`) lo deja posible.

### 3.5 Nombre del proyecto → **Pasame**

Criterios: memorable, pronunciable en español e inglés, describe la acción, repo y dominio razonablemente libres, sin colisión con un producto conocido.

| Opción | A favor | En contra | Verificado (2026-09-21) |
|---|---|---|---|
| **Pasame** | Es literalmente la frase del caso de uso ("pasame ese archivo"). Corta. En inglés se lee "pah-SAH-meh" sin problemas. Sin tilde en el nombre de marca (la forma correcta es "pasame" con voseo, que no lleva tilde). | Hay repos con el nombre (apuntes de exámenes, un repo vacío en `github.com/pasame/pasame` sin actividad desde 2023). `pasame.app` está registrado (DNS en Vercel). | `github.com/<usuario>/pasame`: libre. `pasame.dev`: sin DNS (probablemente libre). `pasame.com.ar`: sin DNS. |
| Acercame | Todo libre (`github.com/acercame` sin usuario, `acercame.app` sin DNS). | Más largo, "ce" se pronuncia distinto en inglés, y "acercar" no describe tan bien la acción. | Todo libre. |
| Ahí va (`ahiva`) | Es lo que se dice al entregar algo. Simpático. | La grafía `ahiva` es ambigua (¿"ahiva"? ¿"ahí va"?); difícil de buscar. `ahiva.app` registrado. | `ahiva.app`: registrado. |

**Decidido: "Pasame".** Repo `github.com/<usuario>/pasame`, binario `pasame`, app `Pasame.app`/`Pasame.exe`, **dominio `pasame.com.ar`** (decisión del 2026-09-21; se registra en NIC.ar y se apunta a GitHub Pages con un `CNAME` en `site/`, HTTPS con el certificado automático de Pages). Carpeta de cuarentena: `~/Descargas/Pasame/`. El resto del documento usa "Pasame" como nombre; la carpeta `share-now` se renombra al empezar la implementación.

**Nota sobre el dominio:** `pasame.com.ar` no participa del producto en ejecución. La app nunca lo consulta: las direcciones que se muestran son siempre IP local o el subdominio efímero de `trycloudflare.com`. El dominio es solo la puerta de entrada para descargar la app y para la página de "cómo abrirla" (ver 11). Si el dominio se cae o expira, la app sigue funcionando igual.

---

## 4. Arquitectura

### 4.1 Vista de componentes

```
┌──────────────────────────── proceso `pasame` (Go) ─────────────────────────────┐
│                                                                                │
│  ┌──────────────┐   eventos (SSE)   ┌──────────────────────────────────────┐   │
│  │  control     │◄──────────────────│              core                    │   │
│  │  127.0.0.1:N │   API JSON        │  ┌─────────┐ ┌────────┐ ┌────────┐   │   │
│  │  (UI emisor) │──────────────────►│  │ session │ │ addr   │ │ tunnel │   │   │
│  └──────┬───────┘                   │  │ token   │ │ (LAN,  │ │ cloud- │   │   │
│         │ abre                      │  │ PIN     │ │ túnel, │ │ flared │   │   │
│         ▼                           │  │ files   │ │ mdns)  │ │ )      │   │   │
│   navegador local                   │  │ stats   │ └───┬────┘ └───┬────┘   │   │
│                                     │  └────┬────┘     │          │        │   │
│                                     │       │          └──────────┘        │   │
│                                     │       │              ▲               │   │
│  ┌──────────────┐                   │       │              │ URL activa    │   │
│  │  share       │◄──────────────────┼───────┘         ┌────┴────┐          │   │
│  │  0.0.0.0:8080│  lee sesión       │                 │   qr    │          │   │
│  │  (receptores)│                   │                 └─────────┘          │   │
│  └──────┬───────┘                   └──────────────────────────────────────┘   │
│         │                                                                      │
└─────────┼──────────────────────────────────────────────────────────────────────┘
          │ HTTP plano                     ▲
          ├───────────────► celulares, PCs, TVs en la misma red
          │                                │
          └──── 127.0.0.1:8080 ◄── cloudflared (proceso hijo) ◄── https://xxx.trycloudflare.com ◄── internet
```

Cinco componentes acordados (UI, Sesión, Servidor, Direcciones, QR) más tres auxiliares que aparecen al bajar a implementación: `tunnel` (gestión del binario `cloudflared`, es un *proveedor* de `addr`), `dialog` (diálogo nativo) y `lifecycle` (single instance, apagado).

### 4.2 Dos listeners, no uno — la trampa del túnel

Es la decisión estructural más importante que no estaba en el brainstorming.

**Problema:** la UI de control (elegir archivos, parar, abrir túnel, ver carpeta) solo debe ser accesible desde la propia máquina. La forma "obvia" es un único servidor en `0.0.0.0:8080` con un middleware que chequea `r.RemoteAddr` es loopback para las rutas `/app/*`. **Esto es un agujero con el túnel activo:** `cloudflared` corre en la misma máquina y reenvía cada request de internet a `http://localhost:8080`. Para el servidor, *todo* el tráfico del túnel llega desde `127.0.0.1`. El chequeo pasa. Cualquiera en internet con la URL del túnel controla la app.

**Solución:** dos `net.Listener` distintos, con dos `http.ServeMux` distintos, sin rutas compartidas:

| Listener | Bind | Puerto | Qué sirve | Quién llega |
|---|---|---|---|---|
| `control` | `127.0.0.1` (solo IPv4 loopback; también `[::1]` si está disponible) | Aleatorio (`:0`), impreso en la URL que se abre | `/` (UI del emisor), `/api/*`, `/events` (SSE) | Solo el navegador local. Requiere el token `t` (ver 4.8). |
| `share` | `0.0.0.0` y `[::]` | **8080** fijo, con fallback 8081…8089 y luego aleatorio | `/`, `/s/<token>`, `/s/<token>/f/<i>`, `/s/<token>/zip`, `/s/<token>/up`, `/s/<token>/pin` | La red local y, si está activo, `cloudflared` (que apunta **solo** a este listener). |

El túnel se levanta con `--url http://127.0.0.1:8080` (el puerto del listener `share`). Nunca conoce el puerto del listener `control`. Aunque alguien lo adivinara, el token `t` lo frena.

**Por qué 8080:** memorable ("ochenta ochenta"), lo suficientemente alto para no necesitar privilegios en ningún OS, y raramente ocupado en la máquina de un no técnico. Si está ocupado (típico en la máquina de un desarrollador), se prueba 8081–8089 y después uno aleatorio; la dirección que se muestra en pantalla siempre incluye el puerto real, así que el usuario no lo nota.

### 4.3 Sesión

Una sola sesión activa por proceso. Simplifica todo: la UI no tiene "lista de sesiones", el receptor no elige nada, y `/` en el listener `share` redirige a la única sesión.

```go
type Session struct {
    Token     string      // 5 caracteres del alfabeto "abcdefghjkmnpqrstuvwxyz23456789" (sin 0/o/1/l/i), ej. "k3x9m"
    PIN       string      // 4 dígitos, generado siempre; exigido solo en modo estricto
    Files     []File      // snapshot inmutable al momento de elegir
    CreatedAt time.Time
    Strict    bool        // true mientras el túnel está activo → PIN obligatorio en toda ruta
    stats     Stats       // contadores por archivo (descargas iniciadas/completadas), subidas recibidas, receptores únicos (por IP)
}

type File struct {
    Index   int    // posición estable, usada en la URL /f/<i>
    Name    string // nombre a mostrar y a usar en Content-Disposition (saneado)
    Rel     string // ruta relativa dentro del ZIP (para carpetas: "carpeta/sub/archivo.jpg")
    Abs     string // ruta absoluta en disco (nunca sale del proceso)
    Size    int64
    ModTime time.Time
}
```

Reglas:

- **Token de 5 caracteres** con alfabeto de 31 símbolos ≈ 28,6 millones de combinaciones. Suficiente para que un vecino de LAN no adivine y para que un link viejo no muestre la sesión nueva. No es una defensa contra internet: para eso están el PIN y la URL aleatoria de Cloudflare.
- **El PIN se genera siempre** (aunque no se use en LAN), así activar el túnel no cambia el QR que ya se mostró… salvo que la URL cambia de todas formas al activar el túnel. Se genera siempre igual porque cuesta nada y simplifica el código.
- **Cambiar los archivos = nueva sesión** (nuevo token). Los links viejos responden "Este envío ya terminó". Evita que alguien con un link de ayer vea lo que compartís hoy.
- **Snapshot de carpetas:** al elegir una carpeta se recorre una sola vez (`filepath.WalkDir`), se omiten archivos ocultos (`.` inicial, `Thumbs.db`, `desktop.ini`, `.DS_Store`) y symlinks (no se siguen), y la lista queda fija. Si el usuario agrega archivos a la carpeta después, no aparecen. Es predecible y evita recorrer disco en cada request.
- **Sin TTL absoluto en v1.** La sesión vive mientras el proceso viva; el proceso muere por la regla de ciclo de vida (4.10). Un TTL configurable es innecesario para el caso presencial.

### 4.4 Servidor de compartir (listener `share`)

| Método y ruta | Qué hace | Detalles de implementación |
|---|---|---|
| `GET /` | Modo normal: `302` a `/s/<token>`. Modo estricto: página "Pedile el QR a quien comparte" (sin revelar el token). | Permite tipear solo `192.168.1.42:8080`. |
| `GET /s/<token>` | Página del receptor (lista de archivos, botón "Descargar todo", zona de subida). | Template Go embebido. Detecta `Accept-Language`: `es` por defecto, `en` si el navegador lo prefiere. `Cache-Control: no-store`. |
| `GET /s/<token>/f/<i>` | Descarga del archivo `i`. | `http.ServeContent` sobre `*os.File` → Range, reanudación, `Content-Length`. `Content-Disposition: attachment; filename*=UTF-8''<nombre>` más `filename="<ascii fallback>"`. Con `?inline=1`: `Content-Disposition: inline` (botón "Ver": fotos en Safari iOS para guardar en Fotos, video en Smart TV). |
| `GET /s/<token>/zip` | Todos los archivos en un ZIP generado al vuelo. | `zip.NewWriter(w)`; método `Store` (sin compresión: fotos/videos ya están comprimidos y `Deflate` bajaría a 30 MB/s); `Transfer-Encoding: chunked` (tamaño total desconocido, se acepta); ZIP64 automático. Nombre: `Pasame-<fecha>.zip`. `Content-Disposition: attachment`. Sin Range (un ZIP al vuelo no es reanudable; se documenta). |
| `POST /s/<token>/up` | Dirección inversa: subida de 1..N archivos. | `multipart/form-data`; se lee con `r.MultipartReader()` parte por parte, **nunca** `ParseMultipartForm` (bufferiza en memoria/disco temporal). Cada parte se escribe en `<cuarentena>/<nombre>.part` y se renombra al terminar (atómico). Respuesta: sin JS, `303` a `/s/<token>?subido=3`; con JS (`Accept: application/json`), `{"ok":true,"files":[...]}`. Sin límite artificial de tamaño; si el disco se llena, `507` y mensaje claro. |
| `GET/POST /s/<token>/pin` | Solo en modo estricto: formulario de PIN. | `GET ?pin=1234` (viene del QR) o `POST pin=1234` (tipeado). Si es correcto: cookie `pasame_pin=<HMAC(token,pin)>`, `HttpOnly`, `SameSite=Lax`, y `302` a `/s/<token>`. 5 intentos fallidos → 60 s de bloqueo global (ver 5). |
| `GET /s/<otro-token>` | Sesión vieja o inventada. | `410 Gone` con la página "Este envío ya terminó. Pedile un QR nuevo a quien te lo compartió." |
| `GET /healthz` | `200 ok`. | Lo usa la UI de control para el autodiagnóstico (ver 6.3) y los tests. |

**Página del receptor (HTML base, sin JS):**

```
┌───────────────────────────────────────────┐
│  Pasame                                   │
│                                           │
│  Martín te comparte 3 archivos            │
│                                           │
│  ┌─────────────────────────────────────┐  │
│  │ ⬇  Descargar todo (1,2 GB)          │  │   ← <a href=".../zip"> gigante
│  └─────────────────────────────────────┘  │
│                                           │
│  ─────────────────────────────────────    │
│  ▸ vacaciones.mp4         980 MB   [Ver]  │   ← <a href=".../f/0"> ; [Ver] solo para imagen/video/audio/pdf
│  ▸ contrato.pdf           1,3 MB   [Ver]  │
│  ▸ foto-abuela.jpg        4,2 MB   [Ver]  │
│  ─────────────────────────────────────    │
│                                           │
│  ¿Querés mandarle algo a Martín?          │
│  ┌─────────────────────────────────────┐  │
│  │ ⬆  Elegir archivos para enviar      │  │   ← <form method=post enctype=multipart> + <input type=file multiple>
│  └─────────────────────────────────────┘  │
│                                           │
│  Los archivos viajan solo por esta red.   │   ← en modo túnel: "Los archivos viajan cifrados por internet."
└───────────────────────────────────────────┘
```

**JavaScript (progresivo, ~4 KB, ES5):** si está disponible, (a) al elegir archivos en el input, sube automáticamente con `XMLHttpRequest` y muestra barra de progreso "Enviando 2 de 5 · 43 %"; sin JS, aparece un botón "Enviar" y el navegador muestra su propio progreso; (b) muestra "✓ Enviado" al terminar. Nada más. Las descargas son links comunes: el gestor de descargas del navegador ya muestra progreso, pausa y reanuda.

**"Quién comparte":** el nombre ("Martín te comparte…") sale del nombre de usuario del OS (`os/user`), capitalizado. Es un detalle que hace que la página se sienta dirigida a la persona. Editable en la UI del emisor (campo "Tu nombre", recordado en el archivo de configuración).

**Concurrencia:** sin límite artificial. Cada request es un goroutine. Se setea `http.Server{ReadHeaderTimeout: 10s, IdleTimeout: 120s}` y **no** `WriteTimeout` (mataría descargas largas). Para uploads, `ReadTimeout` tampoco (un video de 2 GB por WiFi tarda); se protege con un `http.MaxBytesReader` inexistente (sin límite) pero con un *watchdog de progreso*: si una parte no recibe bytes en 60 s, se aborta y se borra el `.part`.

### 4.5 Direcciones — la costura clave

```go
// internal/addr
type Address struct {
    Kind     string // "lan" | "tunnel" | "mdns"
    URL      string // "http://192.168.1.42:8080/s/k3x9m" | "https://xxx.trycloudflare.com/s/k3x9m?pin=1234"
    Display  string // lo que se muestra en letra grande: "192.168.1.42:8080" | "xxx.trycloudflare.com"
    Label    string // "En esta red WiFi" | "Por internet" | "Nombre fácil (puede no funcionar)"
    Primary  bool   // la que va al QR por defecto
}

type Provider interface {
    Kind() string
    Addresses(ctx context.Context, sessionPath string) ([]Address, error) // sessionPath = "/s/k3x9m" (+ "?pin=" si estricto)
}

type Book struct {
    providers []Provider
    changed   chan struct{} // notifica a la UI (SSE) cuando cambia el conjunto
}

func (b *Book) Active(ctx) []Address // concatena, en orden: tunnel (si está), lan, mdns
```

- La UI, el servidor y el QR solo conocen `Book.Active()`. No importan `tunnel` ni `lan`.
- **LAN siempre está registrado.** `tunnel` se registra en el `Book` cuando el usuario lo activa y se quita cuando lo apaga. `mdns` se registra si el bonus está compilado y el anuncio tuvo éxito.
- Agregar ngrok, Tailscale Funnel o un VPS propio mañana = un archivo nuevo en `internal/addr/` que implementa `Provider`. Nada más cambia.
- **Primaria para el QR:** si hay túnel, el túnel (porque si el usuario lo prendió, es porque la LAN no le funcionó); si no, LAN. La UI muestra las otras como secundarias.

#### 4.5.1 Proveedor LAN: elegir la interfaz correcta (el bug clásico de las VPN)

Algoritmo, en orden:

1. Enumerar `net.Interfaces()`, descartar: `down`, loopback, sin IPv4 (IPv6 se ignora en v1: los QR con `[fe80::…]` son ilegibles y los link-local necesitan zone id), link-local `169.254/16`.
2. Descartar por nombre (prefijos, case-insensitive): `docker`, `br-`, `veth`, `virbr`, `vbox`, `vmnet`, `utun`, `tun`, `tap`, `tailscale`, `zt`, `wg`, `ppp`, `Hyper-V`, `vEthernet`, `WSL`. Se descartan de la **preferencia**, no de la lista: siguen apareciendo en "Cambiar red" por si el usuario justamente quiere compartir por Tailscale.
3. Puntuar las restantes: `192.168.0.0/16` = 3, `10.0.0.0/8` = 2, `172.16.0.0/12` = 2, cualquier otra (IP pública en la interfaz, típico en universidades) = 1. Sumar +1 si coincide con la interfaz que el OS usaría para salir a internet (`net.Dial("udp", "1.1.1.1:53")` sin enviar nada, leer `LocalAddr`): con una VPN de túnel completo esto apunta a la VPN, pero como las `utun`/`tun` ya perdieron preferencia en el paso 2, el empate lo decide el rango privado. Sumar +1 si el nombre contiene `wl`, `wi-fi`, `wlan`, `en0` (macOS WiFi habitual), `Ethernet`.
4. La de mayor puntaje es la primaria. Todas se listan en "Cambiar red" con su nombre humano (`Wi-Fi`, `Ethernet`, `Tailscale`) e IP. La elección manual se recuerda en configuración para esa interfaz.
5. **Aviso de VPN:** si existe alguna interfaz `up` con IP cuyo nombre matchea el paso 2 (`tun`, `utun`, `tailscale`, `wg`, `ppp`), la UI muestra: *"Parece que tenés una VPN activa. Si el celular no puede entrar, apagala un momento o tocá 'Cambiar red'."*

Se re-evalúa cada 5 s (es barato) y, si la primaria cambia (el usuario apagó la VPN, cambió de WiFi), la UI actualiza el QR con un aviso "Cambió la red: el QR se actualizó".

#### 4.5.2 Proveedor túnel

Envuelve a `internal/tunnel` (4.6). `Addresses()` devuelve la URL pública **con** `?pin=<PIN>` en la query, porque en modo estricto la página exige PIN y el QR lo lleva embebido. `Display` es solo el host (`xxx.trycloudflare.com`), para leerlo en voz alta si hace falta.

#### 4.5.3 Proveedor mDNS (bonus, recortable)

Anuncia `pasame.local` → IP primaria, con un responder mDNS propio (`hashicorp/mdns`), para que la dirección tipeable sea `pasame.local:8080` en vez de `192.168.1.42:8080`. Verificado: resuelve nativamente en macOS/iOS, Windows 10 1903+, Android 12+; en Linux necesita `avahi` (viene en Ubuntu/Fedora de escritorio). Reglas:

- Se muestra como **tercera línea, en letra chica**, con la leyenda *"o probá: pasame.local:8080"*. La IP sigue siendo la principal. Nunca va al QR.
- Si el anuncio falla (puerto 5353 ocupado, multicast bloqueado, otra instancia ya anunció `pasame.local`) no se muestra nada. Silencioso.
- Es lo último que se implementa y está detrás de un flag de compilación (`-tags mdns`). Si trae un solo bug de multicast o un diálogo extra de firewall que confunda, se saca del build de v1 sin tocar otra cosa.

### 4.6 Túnel: `cloudflared` bajo demanda

**Verificado (docs oficiales, 2026-09-21):** los quick tunnels (`cloudflared tunnel --url`) no necesitan cuenta ni token; límite de **200 requests en vuelo** (devuelve `429` por encima); **no soportan Server-Sent Events**; **no funcionan si existe `~/.cloudflared/config.yaml`**; "solo para testing y desarrollo", sin SLA ni garantía de uptime; Cloudflare se reserva probar features nuevas en ellos. Versión actual de `cloudflared`: 2026.9.1.

Consecuencias de diseño:

- **200 requests en vuelo alcanza de sobra** para "compartir con 10 personas". No se limita nada.
- **SSE no se usa en el listener `share`** (solo en `control`, que es local). Por eso la página del receptor no tiene "actualización en vivo" y la barra de subida usa XHR. Diseño alineado con el límite.
- **`config.yaml` existente**: se evita pasando `--config /dev/null` (`NUL` en Windows) en la línea de comando. Hay que verificar en el primer test de integración que esta bandera anula la detección automática; si no, se ejecuta con `HOME`/`USERPROFILE` apuntando a un directorio temporal vacío. Se documenta como el primer test manual del túnel.
- **"Solo para testing"**: el uso es opt-in, esporádico y efímero (minutos). Se muestra en la UI: *"Usa un servicio gratuito de Cloudflare. Puede no estar disponible."* Si un día deja de andar, el botón muestra el error y el resto de la app sigue igual; agregar otro proveedor es un archivo (4.5).

**Ciclo del binario:**

1. **Ubicación:** `<config dir>/cloudflared/<version>/cloudflared[.exe]` (`~/Library/Application Support/Pasame/` en macOS, `%APPDATA%\Pasame\` en Windows, `~/.config/pasame/` en Linux).
2. **Descarga (primera vez):** desde `https://github.com/cloudflare/cloudflared/releases/download/<version>/<asset>` con `<version>` **fijada en el código** (no "latest": reproducible y testeable). Assets verificados: `cloudflared-darwin-amd64.tgz`, `cloudflared-darwin-arm64.tgz`, `cloudflared-windows-amd64.exe`, `cloudflared-linux-amd64`, `cloudflared-linux-arm64`. No hay asset para Windows arm64 ni Linux armv7: en esos targets el botón de túnel muestra *"No disponible en esta computadora"*. Se verifica el **SHA-256 fijado en el código** por asset (Cloudflare publica los checksums en la release) antes de marcar como ejecutable. Tamaño ≈ 35–40 MB (estimación; se muestra progreso de descarga en la UI). En macOS el `.tgz` se descomprime; se le quita el atributo de cuarentena con `xattr -d com.apple.quarantine` (un binario bajado por nuestra app, no por el navegador, normalmente no lo tiene, pero se hace igual).
3. **Ejecución:** `cloudflared tunnel --url http://127.0.0.1:<puerto share> --no-autoupdate --config <null> --loglevel info`. Se lee `stderr` línea por línea buscando la regex `https://[a-z0-9-]+\.trycloudflare\.com`. Timeout de 30 s para obtenerla; si no aparece, se mata el proceso y se muestra el error con el log.
4. **Verificación:** antes de mostrar la URL se hace `GET <url>/healthz` desde el propio proceso (sale a internet y vuelve por el túnel). Si responde `200`, se registra el proveedor y la sesión pasa a modo estricto. Evita mostrar un QR que todavía no funciona (los quick tunnels tardan unos segundos en propagarse).
5. **Apagado:** al apagar el túnel o cerrar la app se manda `SIGTERM` (`taskkill /T` en Windows) y se espera 5 s. La sesión vuelve a modo normal. Se limpia la cookie de PIN del lado del servidor invalidando el HMAC (rota la clave).
6. **Si el proceso hijo muere solo** (Cloudflare lo cortó): el proveedor se desregistra, la sesión vuelve a modo normal, la UI muestra *"Se cortó la conexión por internet. Podés volver a activarla."*

### 4.7 QR

- Contenido: la URL completa de la dirección primaria, tal cual (`http://192.168.1.42:8080/s/k3x9m` ≈ 37 caracteres; con túnel y PIN ≈ 70). Ambas entran cómodas en un QR versión 4–6 con corrección `M`.
- Render: SVG generado por el servidor (`skip2/go-qrcode` → matriz → `<svg>` con `<rect>`), insertado inline en la UI de control. Sin PNG, sin canvas: escala perfecto en pantallas 4K y en la de 13".
- Tamaño en pantalla: **mínimo 280 × 280 px CSS**, centrado, con 4 módulos de margen blanco (el "quiet zone" que exige el estándar; sin él, muchas cámaras no leen). En pantallas grandes crece hasta 480 px.
- Un click sobre el QR lo pone a pantalla completa (para mostrárselo a alguien del otro lado de la mesa).

### 4.8 UI de control (listener `control`)

**Protección:** al arrancar se genera un token aleatorio de 32 bytes (`t`). La URL que se abre lo lleva en la query; la página lo guarda en memoria (no en cookie) y lo manda en el header `X-Pasame-Token` en cada request a `/api/*` y como query en `/events`. Sin token válido → `403`. Además, se rechaza cualquier request cuyo header `Host` no sea `127.0.0.1:<puerto>` o `localhost:<puerto>` (mitiga DNS rebinding). Con esto, una página web maliciosa abierta en el mismo navegador no puede operar la app (no conoce `t`; `Host` no coincide; y el navegador bloquea lecturas cross-origin).

**API (JSON, todo local):**

| Ruta | Acción |
|---|---|
| `GET /` | La UI (una sola página, estados manejados en JS). |
| `GET /events` | SSE: `state` (sesión, direcciones, stats, túnel, avisos) cada vez que algo cambia + `ping` cada 5 s. También es el heartbeat inverso: si el servidor no tiene ningún cliente SSE, considera que no hay pestaña abierta. |
| `POST /api/pick` `{"kind":"files"|"folder"}` | Abre el diálogo nativo (bloqueante en el servidor, en un goroutine) y, si el usuario eligió algo, crea la sesión y empieza a compartir. Devuelve `202`; el resultado llega por SSE. |
| `POST /api/receive-only` | Crea una sesión sin archivos (modo "Solo recibir"). |
| `POST /api/stop` | Termina la sesión (token inválido, 410 para links viejos). La app queda en pantalla inicial. |
| `POST /api/quit` | Termina la sesión y el proceso. |
| `POST /api/tunnel` `{"on":true|false}` | Activa/desactiva el túnel. Progreso por SSE (`descargando 40 %`, `conectando`, `listo`, `error`). |
| `POST /api/iface` `{"ip":"10.0.0.5"}` | Fija la interfaz LAN manualmente. |
| `POST /api/name` `{"name":"Martín"}` | Cambia el nombre que ve el receptor. |
| `POST /api/open-folder` | Abre la carpeta de cuarentena en el explorador del OS (`open`, `explorer`, `xdg-open`). |
| `GET /api/qr.svg?url=...` | SVG del QR de una URL dada (solo URLs presentes en `Book.Active()`; cualquier otra → 400). |

**Frontend:** un `index.html`, un `app.css`, un `app.js` (vanilla, ~300 líneas, sin build). Texto grande, contraste alto, un solo botón primario por pantalla. Sin modo oscuro en v1.

### 4.9 Selección de archivos

Tres formas de entrada, todas convergen en `session.New(paths []string)`:

1. **Diálogo nativo** desde la UI (`zenity.SelectFileMultiple()` / `zenity.SelectFile(zenity.Directory())`). Nota macOS: el diálogo lo abre `osascript` desde un proceso sin ventana, y a veces aparece **detrás** del navegador. Mitigación verificable en el primer test manual: `zenity` en macOS ya activa el proceso antes del diálogo; si igual queda atrás, la UI muestra durante el `pick` un cartel *"Se abrió una ventana para elegir archivos. Si no la ves, fijate detrás de esta."*
2. **Argumentos de línea de comando** (`pasame /ruta/a.jpg /ruta/b.mp4 /ruta/carpeta`): arrastrar sobre el ícono, "Abrir con", "Enviar a", terminal. Rutas inexistentes o sin permiso de lectura se listan en la UI como *"No pude leer: b.mp4"* y se continúa con el resto.
3. **Campo de texto "pegar ruta"** (solo Linux, solo si no hay `zenity`/`kdialog` instalado): último recurso, se muestra únicamente cuando el diálogo falla.

**Descartado:** arrastrar archivos *dentro de la página* de control. El navegador entrega contenido, no rutas (por diseño de seguridad); habría que copiar cada archivo a través de localhost al disco, duplicando 8 GB para compartir 8 GB. No cumple el requisito de streaming y confunde ("¿por qué tarda si es mi propia compu?"). La zona de drop existe pero solo muestra: *"Para elegir archivos usá el botón (o arrastralos sobre el ícono de Pasame)."*

### 4.10 Ciclo de vida del proceso

- **Single instance:** lock file `<config dir>/pasame.lock` con el puerto de control y el token, protegido con `flock`/`LockFileEx` (Go: `syscall.Flock` en Unix; en Windows, `windows.LockFileEx` vía `golang.org/x/sys`, que es stdlib extendida y sin CGO). Segunda instancia con argv: le pasa los archivos a la primera vía `POST /api/add` (con el token del lock file) y termina. Sin argv: abre el navegador en la UI de la primera y termina.
- **Apagado limpio** (`SIGINT`, `SIGTERM`, `/api/quit`, regla de inactividad): parar `cloudflared`, cerrar listeners con `Shutdown(ctx 10s)` (deja terminar descargas cortas; las largas se cortan, el receptor puede reanudar si el emisor vuelve a compartir el mismo archivo… no: la sesión será otra. Se acepta), borrar `.part` huérfanos, borrar lock file.
- **Regla de inactividad:** sin cliente SSE **y** sin transferencias durante 2 min → `quit`. Con transferencias → esperar. Se muestra en el README y en la UI ("Si cerrás esta pestaña, Pasame se cierra solo a los 2 minutos").
- **Configuración persistente** (`<config dir>/config.json`, se crea al primer cambio): `name`, `iface_override`, `port_override`, `lang`. Nada más.
- **Logs:** a `stderr` y a `<config dir>/pasame.log` (rotación simple: se trunca al superar 5 MB). Sin telemetría, sin llamadas a internet salvo el túnel (opt-in) y la descarga de `cloudflared` (opt-in, una vez).

---

## 5. Seguridad, por transporte

Principio (heredado): **la seguridad es función del transporte activo.** Regla única y explicable: *"En tu WiFi, cualquiera de tu WiFi puede entrar con el link. Por internet, solo quien tenga el PIN."*

| Amenaza | Modo LAN (normal) | Modo túnel (estricto) |
|---|---|---|
| Alguien de la misma red ve los archivos | Aceptado por diseño: la LAN es el círculo de confianza (igual que AirDrop "todos" o una carpeta compartida). Mitigado por el token de 5 caracteres (no adivinable a ciegas) y porque la sesión dura minutos. | No aplica (el token + PIN cubren). |
| Alguien de internet ve los archivos | Imposible: nada sale del router. | URL aleatoria de Cloudflare (no enumerable) + token + **PIN obligatorio en todas las rutas**, cookie HMAC con clave rotada por sesión. 5 fallos → bloqueo global de 60 s (a través del túnel todas las IPs son loopback, así que el bloqueo no puede ser por IP; se usa `Cf-Connecting-Ip` si está presente para el log, no para decidir). |
| Alguien te escribe archivos en el disco | Solo tu red. Cuarentena fija + nombres saneados. | PIN + cuarentena + nombres saneados. |
| Path traversal en subidas | `filepath.Base(nombre)`; se eliminan caracteres de control, `/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`; se rechazan nombres vacíos, `.`, `..`, y en Windows los reservados (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`9`, `LPT1`–`9`); se trunca a 200 bytes preservando la extensión; colisión → `nombre (2).ext`. El destino se calcula como `filepath.Join(cuarentena, saneado)` y se verifica que `filepath.Dir(destino) == cuarentena`. | Ídem. |
| Path traversal en descargas | Imposible por construcción: las URLs usan índices (`/f/2`), nunca nombres ni rutas. | Ídem. |
| Archivos ejecutables recibidos | Se guardan como llegan, sin bit de ejecución (`0644`). En macOS se agrega el atributo `com.apple.quarantine` al archivo recibido para que Gatekeeper lo trate como descarga. En Windows, `Zone.Identifier` (ADS) con `ZoneId=3`. Es lo que hacen los navegadores. | Ídem. |
| Control de la app desde otro dispositivo o desde una web maliciosa | Listener `control` solo en loopback + token `t` + chequeo de `Host`. | Ídem, y el túnel no apunta al listener `control`. |
| `cloudflared` adulterado | Versión y SHA-256 fijados en el código; descarga por HTTPS desde GitHub. | Ídem. |
| Cloudflare ve el tráfico | No aplica. | **Sí: el túnel termina TLS en Cloudflare**; los bytes pasan en claro por su infraestructura. Se dice en la UI al activarlo: *"Los archivos pasan por los servidores de Cloudflare."* No hay cifrado extremo a extremo en v1 (fuera de alcance, ver 11). |
| Denegación de servicio | Sin límites: es tu red. | 200 requests en vuelo (límite de Cloudflare). Sin más. |

**Qué NO se promete:** cifrado en LAN (HTTP plano, decidido), anonimato, protección contra un atacante que ya está en tu red y hace ARP spoofing. Se escribe en el README en un párrafo de 3 líneas, sin jerga.

---

## 6. Flujo de UX pantalla por pantalla

Voz: rioplatense, tuteo con vos, sin términos técnicos. Nunca aparecen las palabras "IP", "puerto", "servidor", "túnel", "firewall", "mDNS", "token". Se reemplazan por "dirección", "esta red WiFi", "por internet", "permiso de Windows".

**Conteo de clicks (camino feliz, PC → celular):**

| Quién | Acción | Clicks/toques |
|---|---|---|
| Emisor | Doble clic en Pasame | 1 |
| Emisor | Click en "Elegir archivos" | 1 |
| Emisor | Seleccionar en el diálogo y "Abrir" | 1 (+ la selección) |
| **→ el QR ya está en pantalla** | | **3 clicks** |
| Receptor | Abrir la cámara y apuntar al QR; tocar el aviso "Abrir en el navegador" | 1 |
| Receptor | Tocar "Descargar todo" (o un archivo) | 1 |
| **→ el archivo está bajando** | | **2 toques** |

Total: **5 interacciones** entre las dos personas. Con arrastrar-sobre-ícono, el emisor baja a 1. Con "Solo recibir" (celular → PC), el emisor hace 2 clicks y el receptor 2 toques + elegir fotos.

### 6.1 Emisor — pantalla inicial

```
┌────────────────────────────────────────────────────────┐
│  Pasame                                       Tu nombre: Martín ✎ │
│                                                        │
│                                                        │
│            Pasá archivos a cualquier celular           │
│            o computadora que esté cerca.               │
│                                                        │
│          ┌──────────────────────────────────┐          │
│          │       📂  Elegir archivos        │          │  ← botón primario, 64 px de alto
│          └──────────────────────────────────┘          │
│                                                        │
│               o elegir una carpeta entera              │  ← link secundario
│                                                        │
│            Solo quiero recibir archivos                │  ← link secundario (modo "Solo recibir")
│                                                        │
│                                                        │
│  Los archivos van directo de tu computadora al otro    │
│  dispositivo. No pasan por ningún servidor.            │
└────────────────────────────────────────────────────────┘
```

Si el sistema es Windows y es la **primera ejecución** (no existe `config.json`), antes de crear la sesión se muestra un aviso in-line (no un modal):

> **Windows te va a pedir permiso.** Cuando aparezca una ventana que dice "Firewall de Windows Defender", tocá **Permitir acceso**. Es para que el celular pueda ver tu computadora.

En macOS solo si el firewall del sistema está activado (se detecta con `defaults read /Library/Preferences/com.apple.alf globalstate` ≠ 0; está desactivado por defecto): *"Si tu Mac pregunta si permitís que Pasame acepte conexiones, tocá Permitir."*

### 6.2 Emisor — compartiendo (la pantalla principal)

```
┌────────────────────────────────────────────────────────────────────┐
│  Pasame                                                            │
│                                                                    │
│   Compartiendo 3 archivos (1,2 GB)                    [ Terminar ] │
│                                                                    │
│   ┌──────────────────────┐   Desde un celular:                     │
│   │                      │   apuntá la cámara a este código        │
│   │   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓   │   y tocá el aviso que aparece.          │
│   │   ▓▓  QR  280px  ▓▓  │                                         │
│   │   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓   │   Desde una computadora:                │
│   │                      │   escribí esto en el navegador          │
│   └──────────────────────┘                                         │
│                              ┌───────────────────────────────────┐ │
│                              │      192.168.1.42:8080            │ │  ← 40 px, monoespaciada
│                              └───────────────────────────────────┘ │
│                              o probá: pasame.local:8080            │  ← solo si mDNS anunció OK, 14 px
│                                                                    │
│   ¿Están lejos o en otra red?  [ Compartir por internet ]          │  ← botón secundario
│                                                                    │
│   ── Actividad ──────────────────────────────────────────────────  │
│   ● Nadie entró todavía. Tiene que estar en la misma red WiFi.     │
│                                                                    │
│   ── Archivos ───────────────────────────────────────────────────  │
│   vacaciones.mp4   980 MB                                          │
│   contrato.pdf     1,3 MB                                          │
│   foto-abuela.jpg  4,2 MB                                          │
│                                                                    │
│   Red: Wi-Fi (192.168.1.42)  ·  Cambiar red ▾                      │  ← 13 px, gris
└────────────────────────────────────────────────────────────────────┘
```

**Actividad** (se actualiza en vivo por SSE):

- `● Nadie entró todavía. Tiene que estar en la misma red WiFi.` (estado inicial)
- `● 1 dispositivo conectado` / `● 3 dispositivos conectados`
- `⬇ Descargando vacaciones.mp4 (2 personas)`
- `✓ vacaciones.mp4 descargado 2 veces`
- `⬆ Recibiendo IMG_2231.jpg…` → `✓ Recibido IMG_2231.jpg  [Abrir carpeta]`

**A los 45 segundos sin ningún dispositivo conectado**, debajo de Actividad aparece (sin modal, sin sonido):

> **¿No pueden entrar?** Probá en este orden:
> 1. Los dos dispositivos tienen que estar en **la misma red WiFi** (fijate el nombre de la red en el celular).
> 2. Si están en un WiFi de un bar, hotel o aeropuerto, esas redes suelen bloquear esto. Tocá **Compartir por internet**.
> 3. Si tenés una VPN, apagala un momento.
> 4. En Windows: si no viste el aviso de permiso, tocá **Revisar permiso de Windows**.

(El botón "Revisar permiso de Windows" ejecuta `control firewall.cpl`, que abre la configuración de aplicaciones permitidas del firewall. Solo se muestra en Windows.)

Es la respuesta honesta al AP isolation: no se puede detectar, así que se anticipa la salida a tiempo.

**Aviso de VPN** (si aplica, desde el inicio, arriba de Actividad): *"Parece que tenés una VPN activa. Si el celular no puede entrar, apagala un momento o tocá Cambiar red."*

**Cambio de red detectado:** el QR se regenera y aparece por 5 s: *"Cambió la red. El código se actualizó."*

**Puerto ocupado:** invisible (se usa 8081, etc.). Si ninguno de 8080–8089 está libre se usa uno aleatorio; sin aviso.

### 6.3 Emisor — compartir por internet

Click en **Compartir por internet** → el panel del QR muestra estados:

1. *"Preparando… (la primera vez baja un componente de 40 MB, tarda un minuto)"* con barra de progreso. Solo la primera vez.
2. *"Conectando…"* (spinner, máx. 30 s).
3. Éxito: el QR cambia a la URL del túnel; el texto pasa a:

```
   ┌──────────────────────┐   Desde un celular: apuntá la cámara.
   │        QR            │
   │   (URL con PIN)      │   Desde una computadora, escribí:
   └──────────────────────┘   ┌────────────────────────────────┐
                              │  amber-cat-dream.trycloudflare.com │
                              └────────────────────────────────┘
                              y cuando te pida la clave:  ┌────────┐
                                                          │  4 8 1 3 │   ← 40 px
                                                          └────────┘
   Los archivos pasan por los servidores de Cloudflare (servicio gratuito, puede no estar disponible).
   [ Volver a compartir solo por WiFi ]
```

4. Error: *"No se pudo conectar por internet. Volvé a intentar en un momento."* + link "Ver detalle" (muestra la última línea del log). El QR de LAN sigue en pantalla.

Cuando el túnel está activo, la sesión es estricta: la UI lo dice en una línea: *"Ahora todos necesitan la clave, también en tu WiFi. El código QR ya la lleva."*

### 6.4 Receptor — camino feliz

1. **Escaneo:** la cámara del celular muestra el aviso del sistema ("Abrir en Safari" / "Abrir en Chrome"). Toca.
2. **Página** (ver 4.4). Título: *"Martín te comparte 3 archivos"*. Botón gigante *"Descargar todo (1,2 GB)"*; lista con cada archivo (tocar el nombre = descargar) y *"Ver"* para fotos/videos/PDF (abre inline: en iPhone permite "Guardar en Fotos"; en TV reproduce el video).
3. **Descarga:** el navegador muestra su propio progreso. En iOS 13+: flecha de descargas arriba a la derecha; el archivo queda en Archivos → Descargas. En Android: notificación de descarga. Si se corta el WiFi, el navegador reanuda (Range).
4. **Enviar algo de vuelta:** *"¿Querés mandarle algo a Martín?"* → botón *"Elegir archivos para enviar"* → selector nativo del celular (fotos, archivos) → con JS: sube al toque con barra *"Enviando 2 de 5 · 43 %"* → *"✓ Listo. Martín ya los tiene."* Sin JS: botón *"Enviar"* → progreso del navegador → página con *"✓ Enviados 5 archivos."*

### 6.5 Receptor — errores

| Situación | Lo que ve | Código HTTP |
|---|---|---|
| Sesión terminada / link viejo | *"Este envío ya terminó. Pedile un código nuevo a quien te lo compartió."* | 410 |
| Modo estricto sin PIN | *"Escribí la clave de 4 números que aparece en la pantalla de Martín."* + 4 casillas grandes + botón *"Entrar"*. | 401 |
| PIN incorrecto | *"La clave no es esa. Fijate bien en la pantalla de Martín."* | 401 |
| 5 PIN incorrectos | *"Demasiados intentos. Esperá un minuto y probá de nuevo."* | 429 |
| `/` en modo estricto | *"Para entrar necesitás el código QR o el link de quien comparte."* | 404 |
| El emisor cerró la app mientras bajaba | El navegador muestra su error de descarga ("Error de red"). No hay forma de mostrar otra cosa. El README lo aclara. | — |
| Subida: sin espacio en disco del emisor | *"No se pudo guardar: la computadora de Martín no tiene espacio."* | 507 |
| Subida: archivo vacío | *"Ese archivo está vacío."* | 400 |
| AP isolation / otra red | El navegador muestra "No se puede acceder al sitio" o carga eternamente. **No podemos mostrar nada**: nunca llegó. La solución está del lado del emisor (6.2). | — |

### 6.6 Emisor — cierre

- **Terminar**: vuelve a la pantalla inicial. Aviso de 3 s: *"Listo. Los links ya no funcionan."*
- **Cerrar pestaña**: sin diálogo de confirmación (los navegadores ya no los respetan bien). El proceso aplica la regla de 2 minutos. El README lo dice.
- **Salir**: menú "⋯" → *"Cerrar Pasame"* → `/api/quit` → la pestaña muestra *"Pasame está cerrado. Podés cerrar esta pestaña."*

---

## 7. Modos de falla → comportamiento del diseño

| Falla | Detección | Comportamiento | Dónde vive |
|---|---|---|---|
| VPN activa elige IP equivocada | Interfaces `tun/utun/wg/tailscale` con IP; puntaje del algoritmo 4.5.1 | Preferir rango privado de interfaz física; aviso de VPN; menú "Cambiar red"; re-evaluación cada 5 s | `internal/addr/lan.go` |
| AP isolation (WiFi público) | **No detectable** de forma fiable | Hint a los 45 s sin conexiones, con el botón de túnel como salida explícita | `web/control/app.js` + estado `stats.uniqueClients` |
| Firewall del OS pide permiso | Windows: primera ejecución. macOS: `alf globalstate` | Aviso anticipado antes de abrir el listener; botón que abre la configuración del firewall en el hint de 45 s | `internal/control` + `internal/platform` |
| VLAN de invitados | No detectable | Mismo hint de 45 s → túnel | ídem AP isolation |
| Android apaga WiFi con pantalla bloqueada | N/A (el emisor es una PC en v1) | Del lado receptor: la descarga se reanuda por Range al desbloquear | `http.ServeContent` |
| Puerto 8080 ocupado | `listen` falla | Fallback 8081–8089, luego aleatorio; siempre se muestra el real | `internal/share/listen.go` |
| Sin navegador / `xdg-open` falla | `exec` devuelve error | Se imprime la URL en stderr y en un diálogo nativo (`zenity.Info`): *"Abrí esta dirección en tu navegador: http://127.0.0.1:53211/?t=…"* | `internal/browser` |
| Sin diálogo nativo (Linux sin zenity) | `zenity` devuelve `ErrUnsupported`/exec error | Campo "pegar ruta" | `internal/dialog` |
| `cloudflared` no descarga (sin internet, GitHub caído, checksum falla) | Error HTTP / mismatch SHA | Mensaje claro, se borra el parcial, se puede reintentar; LAN sigue | `internal/tunnel/download.go` |
| Quick tunnel no da URL en 30 s / `config.yaml` presente | Timeout / stderr | Se mata el proceso, mensaje con "Ver detalle" | `internal/tunnel/run.go` |
| Túnel muere solo | `cmd.Wait()` retorna | Desregistrar proveedor, volver a modo normal, aviso | `internal/addr/tunnel.go` |
| Disco lleno en subida | `write` devuelve `ENOSPC` | Borrar `.part`, 507 al receptor, aviso al emisor | `internal/share/upload.go` |
| Subida se corta a mitad | Watchdog 60 s sin bytes | Borrar `.part`; nunca queda un archivo a medias con nombre final | `internal/share/upload.go` |
| Dos instancias | Lock file | La segunda delega y termina | `internal/lifecycle` |
| Cambio de red durante la sesión | Re-evaluación de 5 s | QR nuevo + aviso | `internal/addr/book.go` |
| Archivo elegido desaparece/cambia durante la sesión | `os.Open` falla / `Size` distinto | 404 en ese archivo con *"Este archivo ya no está disponible"*; el ZIP lo omite y sigue; el emisor ve el aviso | `internal/share/file.go` |

---

## 8. Estructura del proyecto

```
pasame/
├── go.mod                          # module github.com/<usuario>/pasame; go 1.27
├── go.sum
├── README.md                       # en español, con sección "Cómo abrir en Mac/Windows" con capturas
├── LICENSE                         # MIT
├── Makefile                        # build local, test, lint, release-dry
├── .goreleaser.yaml                # 6 targets, universal macOS, .dmg, .deb, checksums
├── .github/workflows/
│   ├── ci.yml                      # go vet, go test -race, build de los 6 targets en cada PR
│   └── release.yml                 # tag v* → goreleaser → GitHub Release (+ notarización si hay secrets)
│
├── cmd/pasame/
│   └── main.go                     # parsea argv, single-instance, arma el core, abre el navegador, señales
│
├── internal/
│   ├── session/
│   │   ├── session.go              # Session/File, New(paths), token, PIN, Stop; sin I/O de red
│   │   ├── walk.go                 # snapshot de carpetas (omite ocultos, no sigue symlinks)
│   │   ├── stats.go                # contadores concurrentes (descargas, subidas, clientes únicos)
│   │   └── *_test.go
│   ├── share/                      # listener público: SOLO lee la sesión, nunca la controla
│   │   ├── server.go               # mux, listen con fallback de puertos, timeouts, modo estricto (middleware PIN)
│   │   ├── page.go                 # GET /s/<token>: template + i18n es/en por Accept-Language
│   │   ├── file.go                 # GET /f/<i>: ServeContent, Content-Disposition, inline
│   │   ├── zip.go                  # GET /zip: archive/zip al vuelo, Store, omite archivos que fallan
│   │   ├── upload.go               # POST /up: multipart streaming, .part + rename, watchdog, ENOSPC
│   │   ├── pin.go                  # GET/POST /pin: cookie HMAC, rate limit
│   │   └── *_test.go               # httptest: range, zip64 con sparse files, uploads, 410, PIN
│   ├── control/                    # listener local: la UI del emisor
│   │   ├── server.go               # mux en 127.0.0.1:0, token t, chequeo de Host
│   │   ├── api.go                  # /api/* → llama a core
│   │   ├── events.go               # SSE: estado + ping; cuenta clientes para el heartbeat
│   │   └── *_test.go
│   ├── core/
│   │   ├── core.go                 # orquesta: sesión actual, Book de direcciones, túnel, stats → un solo State
│   │   ├── state.go                # struct State serializable (lo que ve la UI)
│   │   └── lifecycle.go            # regla de 2 min, shutdown ordenado
│   ├── addr/
│   │   ├── book.go                 # Provider, Address, Book.Active(), notificación de cambios
│   │   ├── lan.go                  # enumeración + puntaje de interfaces, override manual, aviso VPN
│   │   ├── tunnel.go               # Provider que envuelve internal/tunnel
│   │   ├── mdns.go                 # (build tag `mdns`) responder pasame.local
│   │   └── lan_test.go             # tabla de interfaces simuladas: VPN, docker, doble WiFi, sin red
│   ├── tunnel/
│   │   ├── download.go             # URL por OS/arch, SHA-256 fijado, progreso, xattr
│   │   ├── run.go                  # exec cloudflared, parseo de URL en stderr, healthz, stop
│   │   ├── versions.go             # const Version = "2026.9.1" + map asset → sha256
│   │   └── *_test.go               # con un cloudflared FALSO (script que imprime una URL y sirve nada)
│   ├── qr/
│   │   ├── qr.go                   # SVG(url) string
│   │   └── qr_test.go              # decodifica el SVG con una lib de test y compara la URL
│   ├── files/
│   │   ├── sanitize.go             # nombres seguros (tabla de casos), colisiones
│   │   ├── quarantine.go           # ruta de ~/Descargas/Pasame por OS, creación, atributo de cuarentena
│   │   └── *_test.go
│   ├── dialog/
│   │   └── dialog.go               # PickFiles(), PickFolder() sobre ncruces/zenity; ErrUnsupported
│   ├── browser/
│   │   └── open.go                 # open / rundll32 / xdg-open; fallback zenity.Info
│   ├── lifecycle/
│   │   ├── lock.go                 # lock file + flock/LockFileEx (x/sys), handoff de argv a la instancia viva
│   │   └── lock_test.go
│   ├── platform/
│   │   ├── firewall_darwin.go      # alf globalstate
│   │   ├── firewall_windows.go     # abrir firewall.cpl
│   │   ├── firewall_other.go
│   │   ├── openfolder.go           # abrir carpeta en el explorador
│   │   └── configdir.go            # rutas de config por OS (os.UserConfigDir + "Pasame")
│   └── i18n/
│       ├── es.go                   # todos los strings de la página del receptor
│       └── en.go
│
├── web/
│   ├── control/                    # embebido con embed.FS
│   │   ├── index.html
│   │   ├── app.css
│   │   └── app.js                  # vanilla, estados: inicio / compartiendo / túnel / error
│   └── share/
│       ├── page.html               # template del receptor, funciona sin JS
│       ├── page.css                # < 4 KB, sin grid obligatorio
│       ├── page.js                 # < 4 KB, ES5, XHR upload con progreso
│       ├── pin.html
│       └── gone.html
│
├── packaging/
│   ├── macos/
│   │   ├── Info.plist              # LSUIElement=true, CFBundleIdentifier, versión
│   │   ├── droplet.applescript     # on open / on run → ejecuta Contents/Resources/pasame
│   │   ├── build-app.sh            # osacompile + copia del binario universal + icono
│   │   └── icon.icns
│   ├── windows/
│   │   ├── icon.ico
│   │   └── versioninfo.json        # metadatos del .exe (nombre, versión, icono) vía goversioninfo o rsrc
│   └── linux/
│       ├── pasame.desktop          # Terminal=false, MimeType para "Abrir con", %F para argv
│       └── icon.png
│
├── site/                           # pasame.com.ar (GitHub Pages): una página, un botón "Descargar" que detecta el OS
│   ├── index.html
│   └── como-abrir.md               # capturas: SmartScreen y Gatekeeper
│
├── test/
│   ├── manual/
│   │   ├── matriz.md               # checklist multi-dispositivo (ver 9.3)
│   │   └── tunel.md                # checklist del túnel real
│   └── fixtures/
│       └── fake-cloudflared/       # script para tests de tunnel
│
└── docs/superpowers/specs/
    └── 2026-09-21-share-now-design.md   # este documento
```

Reglas de dependencia entre paquetes (se verifican con un test que usa `go list -deps`):

- `session`, `files`, `qr`, `i18n` no importan nada de `net/http` ni de `os/exec`. Puros, testeables en milisegundos.
- `share` importa `session`, `files`, `i18n`. **No** importa `control`, `core`, `addr`, `tunnel`. Es el paquete que un día podría correr en Android.
- `addr` no importa `control` ni `share`. `tunnel` no importa `addr` (es `addr/tunnel.go` el que adapta).
- `core` es el único que conoce a todos. `control` solo conoce a `core`.
- `cmd/pasame` solo arma y conecta.

---

## 9. Estrategia de testing

### 9.1 Unit tests (rápidos, sin red, sin disco salvo `t.TempDir()`)

| Paquete | Qué se prueba | Cómo |
|---|---|---|
| `session` | Generación de token (alfabeto, longitud, unicidad), PIN de 4 dígitos, snapshot de carpeta (omite ocultos y symlinks, `Rel` correcto), stats concurrentes (`-race`). | Tabla + `t.TempDir()` con árbol de archivos. |
| `files` | Sanitización: tabla de 40+ casos (`../../x`, `CON`, `foto.jpg\x00.exe`, nombres unicode, 300 caracteres, vacío, solo puntos, barras de ambos tipos, colisión `x (2).jpg` → `x (3).jpg`). Cuarentena por OS. | Tabla. |
| `addr/lan` | Selección de interfaz con 8 escenarios simulados: solo WiFi; WiFi + Tailscale; WiFi + VPN corporativa full-tunnel; Ethernet + WiFi; Docker + WiFi; solo VPN; sin red; override manual. Aviso de VPN. | Interfaz `ifaceLister` inyectable con datos fijos. |
| `addr/book` | Orden de direcciones (túnel primero), notificación de cambios, registro/desregistro. | Providers falsos. |
| `qr` | El SVG generado decodifica a la URL original. | Lib de decodificación de QR solo en `_test`. |
| `tunnel` | Parseo de la URL en stderr (formatos reales de log de `cloudflared`, incluidos warnings intercalados), timeout, tabla de assets por OS/arch, verificación de SHA (mismatch → error y archivo borrado). | Fixtures de logs reales; servidor `httptest` que sirve un "binario" con checksum conocido. |
| `i18n` | Ninguna clave falta en `en` respecto de `es`. | Reflexión sobre las dos tablas. |
| `lifecycle/lock` | Segunda instancia detecta la primera; lock huérfano (proceso muerto) se recupera. | Dos procesos con `os/exec` del propio test binary. |

### 9.2 Tests de integración (segundos, `httptest` + procesos locales)

| Escenario | Verifica |
|---|---|
| Descarga completa de un archivo de 100 MB (sparse) | Bytes idénticos, `Content-Length`, `Content-Disposition` con UTF-8. |
| Descarga con `Range: bytes=50000000-` | `206`, `Content-Range`, bytes correctos. Reanudación real. |
| 20 descargas concurrentes del mismo archivo | Todas correctas, memoria del proceso estable (< 50 MB RSS, medido con `runtime.MemStats`). |
| ZIP al vuelo con 3 archivos, uno de 5 GB sparse | El ZIP descomprime (Go `archive/zip` lector) con ZIP64, y el proceso no supera 50 MB RSS. Un archivo que falla al abrir se omite y el ZIP sigue válido. |
| Subida multipart de 3 archivos (uno de 1 GB) | Llegan a cuarentena, nombres saneados, sin `.part` residual, RSS estable. Corte a mitad → `.part` borrado. Disco lleno simulado (interfaz de FS inyectable) → 507. |
| Modo estricto | Sin cookie → 401 en `/s/`, `/f/`, `/zip`, `/up`. `?pin=` correcto → cookie → 200. 5 fallos → 429. Cambio de clave HMAC → cookies viejas inválidas. |
| `/` en ambos modos, `/s/viejo` → 410 | Códigos y textos. |
| Listener `control` | Sin `t` → 403. `Host` extraño → 403. SSE entrega `state` y `ping`. Sin clientes SSE 2 min (reloj inyectado) → `quit`. |
| Túnel con `cloudflared` **falso** | Script en `test/fixtures` que imprime una URL de trycloudflare a stderr y hace proxy a `127.0.0.1:<share>`. Verifica registro del proveedor, modo estricto activado, healthz, apagado, y muerte inesperada del hijo. |
| Página del receptor sin JS | Con Playwright (`javaScriptEnabled: false`) en Chromium y WebKit: los links de descarga funcionan, el `<form>` sube. Con JS: barra de progreso aparece y llega a 100 %. Viewport 375 × 667 (iPhone SE) y 1920 × 1080. |
| Build de los 6 targets | `CGO_ENABLED=0 go build` para cada `GOOS/GOARCH` en CI; falla el PR si alguno no compila. |

### 9.3 Prueba manual (inevitable, con checklist versionado en `test/manual/matriz.md`)

Lo que **no** se puede automatizar honestamente:

- **Escaneo real de QR** con la cámara de iPhone y de Android (distintas versiones), incluida la distancia y el brillo de pantalla.
- **Diálogos de firewall** de Windows y macOS: aparecen, el texto de anticipación coincide, "Cancelar" deja el estado esperado (hint a los 45 s).
- **Gatekeeper y SmartScreen**: el flujo documentado en `site/como-abrir.md` es exactamente el que se ve, con capturas actualizadas por versión de OS.
- **Diálogo nativo de archivos** en cada OS (¿aparece adelante?, ¿multi-selección?, ¿carpeta?).
- **Arrastrar sobre el ícono** en Windows, macOS (droplet) y GNOME/KDE.
- **AP isolation real**: en un WiFi de bar/hotel, confirmar que el hint aparece y que el túnel funciona desde ahí.
- **VPN real**: Tailscale, WireGuard, una VPN corporativa full-tunnel; confirmar la interfaz elegida y el aviso.
- **Túnel real** (`cloudflared` verdadero): descarga del binario, URL, PIN embebido en QR, healthz, apagado, comportamiento con `config.yaml` presente.
- **Descarga en iOS Safari** (¿queda en Archivos?, ¿"Ver" permite guardar en Fotos?), **Android Chrome**, **Samsung Internet**.
- **Smart TV**: al menos una (Samsung o LG) para confirmar "ver video" y documentar que no descarga.
- **Video de 8 GB** real por WiFi: velocidad, memoria, reanudación al apagar/prender WiFi del celular.

Matriz mínima por release: Windows 11 + macOS actual + Ubuntu LTS como emisores × iPhone (iOS actual y iOS-2) + Android (actual y Android 9) + una PC como receptores. Se marca cada celda con fecha y resultado. Sin esta matriz completa **no se publica** una release marcada como estable.

---

## 10. Distribución

**Canal principal: GitHub Releases** generadas por GoReleaser en cada tag `v*`.

| Target | Asset | Formato | Notas |
|---|---|---|---|
| Windows x64 | `Pasame-Windows.exe` | Un `.exe` portable con icono y metadatos de versión | Sin instalador: un instalador sin firmar da la misma advertencia que un `.exe` sin firmar y agrega pasos. Se documenta "guardalo en Descargas o en el Escritorio y hacé doble clic". |
| Windows arm64 | `Pasame-Windows-arm64.exe` | Ídem | Sin túnel (no hay `cloudflared` para este target). |
| macOS universal | `Pasame-macOS.dmg` | `.dmg` con `Pasame.app` y un acceso directo a `Aplicaciones` | El `.app` es el droplet + binario universal. |
| Linux x64 / arm64 / armv7 | `pasame-linux-<arch>.tar.gz` y `pasame_<ver>_<arch>.deb` | Binario + `.desktop` + icono | El `.deb` instala en `/usr/bin` y registra el `.desktop`. |
| Todos | `checksums.txt` | SHA-256 | Firmado con `cosign` keyless (Sigstore) desde CI: gratis, sin secretos. |

**Sitio de descarga (`site/` → `pasame.com.ar`, GitHub Pages):** una página con un botón gigante *"Descargar Pasame"* que detecta el OS por `navigator.userAgent` y apunta al asset correcto de la última release (vía la API de GitHub o un JSON regenerado en cada release). Debajo, *"¿Windows dice que protegió tu PC?"* y *"¿Mac dice que no se puede abrir?"* con capturas paso a paso. **Esta página es parte del producto**, no un extra: para el usuario no técnico, el primer contacto con la app es la advertencia del OS.

**Advertencias de seguridad del OS (verificado 2026-09-21):**

| OS | Sin firma | Con firma | Decisión v1 |
|---|---|---|---|
| Windows 10/11 | SmartScreen: "Windows protegió tu PC" → "Más información" → "Ejecutar de todas formas". 2 clicks extra, con un botón que está semi-escondido. La reputación se construye sola con descargas limpias, lentamente. | Certificado OV (~US$ 200–300/año, requiere identidad verificada; la reputación igual tarda semanas) o EV (US$ 250–700/año, solo empresas registradas). Azure Artifact Signing (~US$ 10/mes) **solo acepta individuos de EE. UU. y Canadá** → no aplica desde Argentina como persona física. | **Sin firma.** Página "cómo abrir" con capturas. Revaluar firma OV vía Certum (tiene precio reducido para open source; estimación no verificada: ~€ 70–100/año) cuando haya usuarios reales. |
| macOS 12–14 | Diálogo "no se puede abrir porque no se puede verificar el desarrollador" → clic derecho → Abrir → Abrir. Molesto pero descubrible. | Notarización con Apple Developer Program (US$ 99/año): abre sin ningún diálogo. | **Notarizar apenas el proyecto sea público.** Sin notarización, macOS 15+ (abajo) rompe el objetivo 2. GoReleaser notariza desde Linux con `quill` (verificado en su documentación), sin necesitar un Mac en CI. |
| macOS 15+ | **Ya no hay clic derecho → Abrir.** Hay que ir a Ajustes del Sistema → Privacidad y seguridad → bajar hasta "Abrir de todos modos", que solo aparece durante ~1 hora después del intento fallido. Para una persona de 60 años es una pared. | Igual que arriba: sin diálogo. | Ídem: **la notarización es requisito para que la v1 sea usable en Mac**, no un lujo. Hasta tenerla, el sitio muestra el paso a paso con capturas y se considera "beta en Mac". |
| Linux | Ninguna advertencia. | N/A | `chmod +x` lo hace el `.deb` o el `tar.gz` conserva permisos. |

**Canales secundarios (para técnicos, después de la v1.0):**

- Homebrew: `brew install --cask <usuario>/tap/pasame` (tap propio; no requiere aprobación de homebrew-cask). Con la app notarizada, no hay fricción.
- winget: manifiesto en `microsoft/winget-pkgs` (PR con revisión; requiere URL estable de release y hash; el `.exe` portable califica como `portable`).
- `go install github.com/<usuario>/pasame/cmd/pasame@latest` funciona desde el primer día para quien tenga Go, sin hacer nada.

**Auto-update:** fuera de v1. La app no llama a ningún servidor propio. Cuando exista una v2, se evaluará un chequeo opt-in.

---

## 11. Fuera de la v1 (explícitamente)

| Descartado | Por qué |
|---|---|
| App móvil (Android/iOS) como emisor | Duplica el proyecto; iOS suspende servidores en segundo plano; la dirección inversa cubre el caso principal. Candidato a v2 vía `gomobile` si hay demanda. |
| HTTPS en LAN (certificado autofirmado o CA local) | Pantalla roja = muerte del objetivo 2. La LAN es el círculo de confianza. |
| Cifrado extremo a extremo por el túnel | Requiere JS obligatorio en el receptor (rompe el "sin JS") y manejo de claves. El túnel es opt-in y ya va cifrado hasta Cloudflare. |
| Descubrimiento automático de pares (mDNS/UDP broadcast) | Es la parte frágil de LocalSend; el QR lo reemplaza. |
| Modo "app a app" (P2P entre dos instancias) | El navegador ya es el cliente universal. |
| Múltiples sesiones simultáneas | Complejidad de UI para un caso que no existe en el uso presencial. |
| Compartir texto / links / portapapeles | Feature popular, pero es otra pantalla y otro flujo. Se evalúa para v1.1 (es barato: un archivo `.txt` virtual). |
| Historial de transferencias | Estado persistente que hay que mostrar, borrar y explicar. Nada. |
| Cuentas, contactos, favoritos, "dispositivos confiables" | Nada de estado entre sesiones salvo nombre e interfaz. |
| Instaladores (MSI, PKG), tray icon, menú nativo | Sin firma no mejoran nada; con firma, el portable ya alcanza. |
| Auto-update, telemetría, crash reports | Cero llamadas a servidores propios. |
| Límite de velocidad, cola de transferencias, prioridades | HTTP concurrente ya resuelve. |
| Previsualización de archivos en la página (miniaturas) | Genera trabajo en el emisor por cada visita; el botón "Ver" alcanza. |
| Temas, modo oscuro, personalización | Un tema, contraste alto. |
| Otros proveedores de túnel (ngrok, Tailscale Funnel, bore) | La interfaz `Provider` los deja para después en un archivo cada uno. |
| IPv6 en LAN | Los QR con `[fe80::…%en0]` no son tipeables ni escaneables de forma fiable. IPv4 privado cubre todas las redes domésticas. |
| Selección de puerto en la UI | Existe `port_override` en `config.json` para quien lo necesite; sin UI. |
| Reanudar ZIP (`Range` en `/zip`) | Un ZIP al vuelo no es reproducible byte a byte sin fijar mucho; se descarga de nuevo. |
| Compresión en el ZIP | `Store`: fotos y videos ya están comprimidos; `Deflate` reduce la velocidad 10×. |
| Idiomas más allá de es/en en la página del receptor | La página tiene ~15 strings; se agregan cuando alguien los pida. La UI del emisor solo en español en v1. |

---

## 12. Riesgos principales y mitigación

| # | Riesgo | Probabilidad | Impacto | Mitigación en el diseño |
|---|---|---|---|---|
| 1 | **Gatekeeper (macOS 15+) y SmartScreen frenan al usuario antes de arrancar.** | Alta (es seguro que pasa sin firma) | Crítico para el objetivo 2 | Notarización de macOS como requisito de la v1.0 (US$ 99/año). Página "cómo abrir" con capturas para Windows. Solo **una** persona del grupo necesita pasar esta pared: los receptores nunca instalan nada. |
| 2 | **AP isolation / VLAN de invitados no se puede detectar.** | Alta en bares, hoteles, oficinas | Alto: el usuario mira un QR que no funciona | Hint a los 45 s con la salida (túnel) en el mismo panel. Texto que nombra "bar, hotel, aeropuerto" para que la persona se reconozca. |
| 3 | **Interfaz de red equivocada (VPN, Docker, doble WiFi).** | Alta en máquinas de gente técnica; media en el resto | Alto | Algoritmo de puntaje con tests de 8 escenarios, aviso de VPN, "Cambiar red" siempre visible, re-evaluación cada 5 s. |
| 4 | **TryCloudflare cambia, limita o desaparece.** | Media a largo plazo ("solo testing", sin SLA) | Medio: solo afecta el opt-in | Versión de `cloudflared` fijada, error claro, LAN nunca depende del túnel, `Provider` enchufable para reemplazarlo en un archivo. |
| 5 | **"Cerré la pestaña y no sé si sigue abierto".** | Media | Medio (confusión, no pérdida de datos) | Regla de 2 minutos, single-instance que reabre la misma UI, texto en la UI y README. |
| 6 | **Diálogo nativo de archivos aparece detrás del navegador (macOS) o tarda (PowerShell en Windows, ~1 s).** | Media | Medio | Cartel "se abrió una ventana…" durante el pick; `ncruces/zenity` usa Win32 directo (no PowerShell) en Windows. Verificar en el primer test manual. |
| 7 | **Diálogo de firewall de Windows sin contexto, o el usuario toca "Cancelar".** | Media | Alto (nada funciona y no se sabe por qué) | Aviso anticipado en la primera ejecución; botón que abre la configuración del firewall en el hint de 45 s. |
| 8 | **Descarga en iOS se comporta distinto por versión** (inline vs. Archivos vs. Fotos). | Media | Medio | `Content-Disposition: attachment` por defecto + botón "Ver" para inline. Matriz manual con iOS actual y iOS-2. |
| 9 | **Antivirus marca el `.exe` sin firma como sospechoso** (falso positivo típico en binarios Go poco conocidos). | Baja-media | Alto para ese usuario | Sin ofuscación, sin UPX (UPX dispara heurísticas), binario reproducible, checksums firmados con Sigstore, reporte de falso positivo a Microsoft cuando ocurra. |
| 10 | **Un receptor sube 50 GB y llena el disco del emisor.** | Baja | Medio | Cuarentena separada, `ENOSPC` manejado, `.part` borrado. Sin límite artificial (el emisor ve "Recibiendo…" en vivo y puede Terminar). |
| 11 | **Multicast mDNS trae un diálogo de firewall extra o comportamiento raro.** | Media | Bajo | Está detrás de un build tag y se corta sin costo. |
| 12 | **Un solo desarrollador y tres OS que testear a mano.** | Alta | Medio (releases lentas) | Matriz manual mínima definida (9.3), CI compila los 6 targets en cada PR, Playwright cubre la página del receptor en Chromium y WebKit. |

---

## 13. Apéndice: detalles fijados para el plan de implementación

- **Alfabeto del token:** `abcdefghjkmnpqrstuvwxyz23456789` (31 símbolos), longitud 5, `crypto/rand`.
- **PIN:** 4 dígitos `0000`–`9999`, `crypto/rand`, se muestra con espacios (`4 8 1 3`).
- **Cookie de PIN:** `pasame_pin=<hex(HMAC-SHA256(claveSesión, token))>`, `HttpOnly`, `SameSite=Lax`, `Path=/s/<token>`, sin `Secure` (en LAN es HTTP; a través del túnel el navegador la manda igual porque el origen es HTTPS).
- **Puertos:** `share` 8080 → 8081…8089 → aleatorio. `control` aleatorio siempre.
- **Timeouts HTTP (`share`):** `ReadHeaderTimeout 10s`, `IdleTimeout 120s`, sin `ReadTimeout`/`WriteTimeout`; watchdog de subida 60 s sin bytes.
- **Buffer de copia:** `io.Copy` con buffer de 256 KB (`io.CopyBuffer`) en zip y upload; `ServeContent` usa el suyo.
- **Regla de inactividad:** 120 s sin clientes SSE y sin transferencias en curso.
- **Re-evaluación de interfaces:** cada 5 s.
- **Hint de "¿no pueden entrar?":** 45 s desde que se muestra el QR sin ningún request a `/s/<token>` desde una IP distinta de loopback.
- **Tamaño de QR:** mínimo 280 px CSS, máximo 480 px, quiet zone de 4 módulos, corrección `M`.
- **Carpeta de cuarentena:** macOS `~/Downloads/Pasame`, Windows `%USERPROFILE%\Downloads\Pasame`, Linux `$XDG_DOWNLOAD_DIR/Pasame` (fallback `~/Descargas` o `~/Downloads` si existe, si no `~/Pasame`).
- **Directorio de configuración:** `os.UserConfigDir()` + `/Pasame` (macOS: `~/Library/Application Support/Pasame`; Windows: `%APPDATA%\Pasame`; Linux: `~/.config/pasame`).
- **`cloudflared`:** versión fijada `2026.9.1` (actualizar a mano en `versions.go` junto con los SHA-256 de la release). Flags: `tunnel --url http://127.0.0.1:<port> --no-autoupdate --config <NUL|/dev/null> --loglevel info`. Regex de URL: `https://[a-z0-9-]+\.trycloudflare\.com`. Timeout 30 s. Verificación con `GET /healthz` a través del túnel.
- **Flags de build:** `CGO_ENABLED=0`, `-trimpath`, `-ldflags="-s -w -X main.version=<tag>"`, Windows además `-H=windowsgui`. Sin UPX.
- **Targets:** `windows/amd64`, `windows/arm64`, `darwin/amd64` + `darwin/arm64` → universal con `lipo` (GoReleaser `universal_binaries`), `linux/amd64`, `linux/arm64`, `linux/arm` (GOARM=7).
- **Detección de tipo para "Ver":** por extensión → `image/*`, `video/mp4|webm`, `audio/*`, `application/pdf`. `Content-Type` real vía `mime.TypeByExtension` con fallback `application/octet-stream`.
- **Nombre del emisor:** `os/user` → `user.Name` si no está vacío, si no `user.Username`; capitalizado; editable.
- **Strings de la UI:** todos en `internal/i18n` (receptor) y en `web/control/app.js` como un objeto `T` (emisor), para que ningún texto quede hardcodeado en templates.

### Fuentes verificadas para este documento (2026-09-21)

- Cloudflare, "TryCloudflare" (docs oficiales): sin cuenta; 200 requests en vuelo → `429`; sin SSE; no funciona con `config.yaml`; "testing y desarrollo", sin SLA.
- `cloudflare/cloudflared` releases: versión `2026.9.1`; assets `darwin-amd64.tgz`, `darwin-arm64.tgz`, `windows-amd64.exe`, `linux-amd64`, `linux-arm64` (sin `windows-arm64` ni `linux-arm`).
- Apple / prensa especializada: macOS Sequoia (15) eliminó el clic derecho → Abrir para apps no notarizadas; el camino es Ajustes → Privacidad y seguridad → "Abrir de todos modos", visible ~1 hora tras el intento.
- Microsoft Learn: SmartScreen, reputación, opciones de firma; Azure Artifact Signing disponible para individuos solo en EE. UU. y Canadá.
- Esper / Android Police: resolución de `.local` en el resolver de Android desde Android 12 (no backporteado). Wikipedia/Microsoft: Windows 10 1903+ resuelve hostnames por mDNS.
- `go.dev/dl`: Go 1.27.1 estable.
- `ncruces/zenity` (README): sin CGO; Win32 nativo; `osascript` en macOS; `zenity`/`qarma`/`matedialog` en Linux; último push 2026-08.
- `skip2/go-qrcode`: sin cambios desde 2024-03.
- GoReleaser docs: binarios universales de macOS y notarización con `quill` desde Linux.
- GitHub: `github.com/pasame/pasame` existe (vacío, 2023); `pasame.app` con DNS activo; `pasame.dev`, `pasame.com.ar`, `acercame.app` sin DNS.

**Estimaciones no verificadas (marcadas como tales en el texto):** tamaño final del binario (~8–10 MB), tamaño de `cloudflared` (~35–40 MB), precio de certificado OV para open source en Certum, comportamiento de `--config /dev/null` frente a un `config.yaml` existente (primer test de integración del túnel).
