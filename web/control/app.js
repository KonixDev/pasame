(function () {
  'use strict';
  var t = new URLSearchParams(location.search).get('t') || '';
  var state = null;
  var closed = false;

  // Todos los textos del emisor, en los dos idiomas (spec §13). Voz rioplatense en es; llana en en.
  // Ninguno usa palabras técnicas: lo controla internal/control/ui_test.go.
  var TT = {
    es: {
      who: 'Tu nombre: ',
      edit: 'Cambiar',
      headline: 'Pasá archivos a cualquier celular<br>o computadora que esté cerca.',
      pickFiles: '📂  Elegir archivos',
      pickFolder: 'o elegir una carpeta entera',
      receiveOnly: 'Solo quiero recibir archivos',
      // El spec §6.1 dice "No pasan por ningún servidor", pero §6 prohíbe la palabra "servidor": gana la regla de voz.
      direct: 'Los archivos van directo de tu computadora al otro dispositivo, sin pasar por internet.',
      picking: 'Se abrió una ventana para elegir archivos. Si no la ves, fijate detrás de esta.',
      noDialog: 'No se pudo abrir la ventana para elegir. Pegá acá la ubicación del archivo o carpeta:',
      paste: 'Compartir',
      fwWindows: '<b>Windows te va a pedir permiso.</b> Cuando aparezca una ventana que dice "Firewall de Windows Defender", tocá <b>Permitir acceso</b>. Es para que el celular pueda ver tu computadora.',
      fwMac: 'Si tu Mac pregunta si permitís que Pasame acepte conexiones, tocá <b>Permitir</b>.',
      sharingN: function (n, total) { return 'Compartiendo ' + n + (n === 1 ? ' archivo' : ' archivos') + ' (' + total + ')'; },
      receiving: 'Listo para recibir archivos',
      stop: 'Terminar',
      phone: '<b>Desde un celular:</b><br>apuntá la cámara a este código y tocá el aviso que aparece.',
      pc: '<b>Desde una computadora:</b><br>escribí esto en el navegador',
      alsoTry: 'o probá: ',
      activity: 'Actividad',
      files: 'Archivos',
      nobody: 'Nadie entró todavía. Tiene que estar en la misma red WiFi.',
      devices: function (n) { return n + (n === 1 ? ' dispositivo conectado' : ' dispositivos conectados'); },
      downloading: function (name, k) { return '⬇ Descargando ' + name + (k > 1 ? ' (' + k + ' personas)' : ''); },
      downloaded: function (name, k) { return '✓ ' + name + ' descargado' + (k > 1 ? ' ' + k + ' veces' : ''); },
      uploading: function (name) { return '⬆ Recibiendo ' + name + '…'; },
      received: function (name) { return '✓ Recibido ' + name; },
      openFolder: 'Abrir carpeta',
      cantEnter: '<b>¿No pueden entrar?</b> Probá en este orden:<ol>' +
        '<li>Los dos dispositivos tienen que estar en <b>la misma red WiFi</b> (fijate el nombre de la red en el celular).</li>' +
        '<li>Si están en un WiFi de un bar, hotel o aeropuerto, esas redes suelen bloquear esto.</li>' +
        '<li>Si tenés una VPN, apagala un momento.</li></ol>',
      checkWindows: 'Revisar permiso de Windows',
      vpn: 'Parece que tenés una VPN activa. Si el celular no puede entrar, apagala un momento o tocá Cambiar red.',
      netChanged: 'Cambió la red. El código se actualizó.',
      unreadable: 'No pude leer: ',
      network: 'Red: ',
      changeNet: 'Cambiar red',
      auto: 'Automática',
      tabHint: 'Si cerrás esta pestaña, Pasame se cierra solo a los 2 minutos.',
      stopped: 'Listo. Los links ya no funcionan.',
      copy: 'Copiar link',
      copied: 'Link copiado. Pegalo en un mensaje para alguien de esta red.',
      copyFail: 'No se pudo copiar. Seleccioná la dirección y copiala a mano.',
      qrHint: 'Tocá el código para agrandarlo.',
      busyStop: function (n) { return n === 1 ? 'Hay 1 transferencia en curso. Si terminás ahora, se corta.' : 'Hay ' + n + ' transferencias en curso. Si terminás ahora, se cortan.'; },
      stopAnyway: 'Terminar igual',
      keepSharing: 'Seguir compartiendo',
      closedMsg: 'Pasame está cerrado. Podés cerrar esta pestaña.',
      oss: 'Pasame es gratis y de código abierto. Si te sirvió,',
      star: 'dale una estrella en GitHub',
      madeBy: 'Creado por',
      dropHint: 'Para elegir archivos usá el botón (o arrastralos sobre el ícono de Pasame).',
      noNetwork: 'Esta computadora no está conectada a ninguna red. Conectate a un WiFi y esperá unos segundos.',
      editLabel: 'Cambiar nombre',
      askName: '¿Cómo te llamás? (lo ve quien recibe)',
      cantEnterPlan2: '<p>Si están en un bar, hotel o aeropuerto, o en redes distintas: <button class="secondary" data-act="tunnel-on">Compartir por internet</button></p>',
      far: '¿Están lejos o en otra red?',
      tunnelOn: 'Compartir por internet',
      tunnelOff: 'Volver a compartir solo por WiFi',
      preparing: 'Preparando… (la primera vez baja un componente de 40 MB, tarda un minuto)',
      connecting: 'Conectando…',
      askKey: 'y cuando te pida la clave:',
      privacy: 'Los archivos pasan por los servidores de Cloudflare (servicio gratuito, puede no estar disponible).',
      strictNote: 'Ahora todos necesitan la clave, también en tu WiFi. El código QR ya la lleva.',
      tunnelError: 'No se pudo conectar por internet. Volvé a intentar en un momento.',
      detail: 'Ver detalle',
      dropped: 'Se cortó la conexión por internet. Podés volver a activarla.',
      noTunnel: 'Compartir por internet no está disponible en esta computadora.',
      more: 'Más opciones',
      menuStar: '★ Dar una estrella en GitHub',
      menuMadeBy: 'Creado por martincoll.dev',
      quit: 'Cerrar Pasame',
      otherLang: 'English',
      loading: 'Cargando…',
      errors: { bad_name: 'Escribí un nombre.', bad_iface: 'Esa dirección no es de esta computadora.', no_paths: 'No llegó ningún archivo.', bad_lang: 'Idioma no disponible.', generic: 'No se pudo. Probá de nuevo.' }
    },
    en: {
      who: 'Your name: ',
      edit: 'Change',
      editLabel: 'Change name',
      askName: 'What is your name? (the people receiving will see it)',
      headline: 'Send files to any phone<br>or computer nearby.',
      pickFiles: '📂  Choose files',
      pickFolder: 'or choose a whole folder',
      receiveOnly: 'I only want to receive files',
      direct: 'Files go straight from your computer to the other device, without going through the internet.',
      picking: 'A window opened to choose files. If you cannot see it, look behind this one.',
      noDialog: 'The window to choose files could not open. Paste the location of the file or folder here:',
      paste: 'Share',
      fwWindows: '<b>Windows will ask for permission.</b> When a window titled "Windows Defender Firewall" appears, click <b>Allow access</b>. This lets the phone see your computer.',
      fwMac: 'If your Mac asks whether Pasame may accept incoming connections, click <b>Allow</b>.',
      sharingN: function (n, total) { return 'Sharing ' + n + (n === 1 ? ' file' : ' files') + ' (' + total + ')'; },
      receiving: 'Ready to receive files',
      stop: 'Stop sharing',
      phone: '<b>From a phone:</b><br>point the camera at this code and tap the notice that appears.',
      pc: '<b>From a computer:</b><br>type this into the browser',
      alsoTry: 'or try: ',
      activity: 'Activity',
      files: 'Files',
      nobody: 'No one has joined yet. They need to be on the same WiFi network.',
      devices: function (n) { return n + (n === 1 ? ' device connected' : ' devices connected'); },
      downloading: function (name, k) { return '⬇ Downloading ' + name + (k > 1 ? ' (' + k + ' people)' : ''); },
      downloaded: function (name, k) { return '✓ ' + name + ' downloaded' + (k > 1 ? ' ' + k + ' times' : ''); },
      uploading: function (name) { return '⬆ Receiving ' + name + '…'; },
      received: function (name) { return '✓ Received ' + name; },
      openFolder: 'Open folder',
      cantEnter: '<b>Can\'t connect?</b> Try this, in order:<ol>' +
        '<li>Both devices need to be on <b>the same WiFi network</b> (check the network name on the phone).</li>' +
        '<li>WiFi in cafés, hotels and airports often blocks this.</li>' +
        '<li>If you use a VPN, turn it off for a moment.</li></ol>',
      cantEnterPlan2: '<p>If you are in a café, hotel or airport, or on different networks: <button class="secondary" data-act="tunnel-on">Share over the internet</button></p>',
      far: 'Far away or on another network?',
      tunnelOn: 'Share over the internet',
      tunnelOff: 'Go back to sharing over WiFi only',
      preparing: 'Getting ready… (the first time it downloads a 40 MB component, it takes a minute)',
      connecting: 'Connecting…',
      askKey: 'and when it asks for the code:',
      privacy: 'Files go through Cloudflare\'s servers (free service, may be unavailable).',
      strictNote: 'Now everyone needs the code, also on your WiFi. The QR code already includes it.',
      tunnelError: 'Could not connect over the internet. Try again in a moment.',
      detail: 'Show details',
      dropped: 'The internet connection dropped. You can turn it on again.',
      noTunnel: 'Sharing over the internet is not available on this computer.',
      checkWindows: 'Check Windows permission',
      vpn: 'It looks like a VPN is on. If the phone cannot connect, turn it off for a moment or use Change network.',
      netChanged: 'The network changed. The code was updated.',
      unreadable: 'Could not read: ',
      network: 'Network: ',
      changeNet: 'Change network',
      auto: 'Automatic',
      tabHint: 'If you close this tab, Pasame quits by itself after 2 minutes.',
      stopped: 'Done. The links no longer work.',
      copy: 'Copy link',
      copied: 'Link copied. Paste it in a message to someone on this network.',
      copyFail: 'Could not copy. Select the address and copy it by hand.',
      qrHint: 'Tap the code to make it bigger.',
      busyStop: function (n) { return n === 1 ? '1 transfer is in progress. If you stop now, it will be cut off.' : n + ' transfers are in progress. If you stop now, they will be cut off.'; },
      stopAnyway: 'Stop anyway',
      keepSharing: 'Keep sharing',
      closedMsg: 'Pasame is closed. You can close this tab.',
      oss: 'Pasame is free and open source. If it helped you,',
      star: 'give it a star on GitHub',
      madeBy: 'Made by',
      dropHint: 'To choose files, use the button (or drag them onto the Pasame icon).',
      noNetwork: 'This computer is not connected to any network. Connect to a WiFi network and wait a few seconds.',
      more: 'More options',
      menuStar: '★ Star on GitHub',
      menuMadeBy: 'Made by martincoll.dev',
      quit: 'Quit Pasame',
      otherLang: 'Español',
      loading: 'Loading…',
      errors: { bad_name: 'Type a name.', bad_iface: 'That address does not belong to this computer.', no_paths: 'No file arrived.', bad_lang: 'Language not available.', generic: 'That did not work. Try again.' }
    }
  };
  var T = TT.es;

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function api(path, body) {
    return fetch(path, {
      method: 'POST',
      headers: { 'X-Pasame-Token': t, 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {})
    }).then(function (r) {
      if (r.ok) return null;
      return r.json().then(function (j) { toast(T.errors[j.code] || T.errors.generic); }, function () { toast(T.errors.generic); });
    });
  }

  function toast(msg) {
    var el = document.getElementById('toast');
    el.textContent = msg;
    el.hidden = false;
    clearTimeout(toast.timer);
    toast.timer = setTimeout(function () { el.hidden = true; }, 3000);
  }

  function renderWho(s) {
    document.getElementById('who').innerHTML = esc(T.who) + '<b>' + esc(s.name) + '</b> <button class="link" id="edit-name" aria-label="' + esc(T.editLabel) + '">' + esc(T.edit) + '</button>';
  }

  function renderIdle(s) {
    var h = '<div class="center">';
    h += '<h1 style="margin:40px 0 32px;font-weight:600">' + T.headline + '</h1>';
    if (s.firewallHint === 'windows') h += '<div class="notice">' + T.fwWindows + '</div>';
    if (s.firewallHint === 'mac') h += '<div class="notice">' + T.fwMac + '</div>';
    if (s.phase === 'picking') h += '<div class="notice">' + esc(T.picking) + '</div>';
    h += '<p><button class="primary" data-act="pick-files">' + esc(T.pickFiles) + '</button></p>';
    h += '<p><button class="link" data-act="pick-folder">' + esc(T.pickFolder) + '</button></p>';
    h += '<p><button class="link" data-act="receive-only">' + esc(T.receiveOnly) + '</button></p>';
    if (s.pickUnsupported) {
      h += '<div class="notice"><p>' + esc(T.noDialog) + '</p><input type="text" id="paste" size="40"> ' +
        '<button class="secondary" data-act="paste">' + esc(T.paste) + '</button></div>';
    }
    h += '<p class="muted" style="margin-top:48px">' + esc(T.direct) + '</p>';
    h += credits();
    h += '<p class="small" id="drop-hint" hidden>' + esc(T.dropHint) + '</p></div>';
    return h;
  }

  // Isotipo de la marca como indicador: el centro es esta compu, los de alrededor, quienes entraron (hasta 6).
  function ronda(n) {
    var pts = [[32, 12.5], [48.9, 22.25], [48.9, 41.75], [32, 51.5], [15.1, 41.75], [15.1, 22.25]], h = '';
    for (var i = 0; i < 6; i++) h += '<circle cx="' + pts[i][0] + '" cy="' + pts[i][1] + '" r="5.5" fill="' + (i < n ? 'var(--brand)' : 'var(--off)') + '"/>';
    return '<svg class="ronda" viewBox="0 0 64 64" aria-hidden="true"><circle cx="32" cy="32" r="10" fill="var(--accent)"/>' + h + '</svg>';
  }

  function activity(s) {
    var st = s.stats || {}, lines = [];
    lines.push(ronda(st.clients || 0) + '<span>' + esc(st.clients ? T.devices(st.clients) : T.nobody) + '</span>');
    var files = s.files || [];
    Object.keys(st.active || {}).forEach(function (i) {
      if (files[i]) lines.push(esc(T.downloading(files[i].name, st.active[i])));
    });
    Object.keys(st.completed || {}).forEach(function (i) {
      if (files[i]) lines.push(esc(T.downloaded(files[i].name, st.completed[i])));
    });
    (st.uploading || []).forEach(function (n) { lines.push(esc(T.uploading(n))); });
    (st.received || []).forEach(function (n) {
      lines.push(esc(T.received(n)) + ' <button class="link" data-act="open-folder">' + esc(T.openFolder) + '</button>');
    });
    return '<ul class="plain">' + lines.map(function (l) { return '<li>' + l + '</li>'; }).join('') + '</ul>';
  }

  function renderSharing(s) {
    var addrs = s.addresses || [];
    var main = addrs[0];
    var h = '<div class="topbar"><h1>' + esc(s.receiveOnly ? T.receiving : T.sharingN(s.count, s.total)) + '</h1>' +
      '<button class="secondary" data-act="stop">' + esc(T.stop) + '</button></div>';
    if (confirmStop) {
      var n = inFlight(s);
      if (n > 0) {
        h += '<div class="notice" role="alert"><p style="margin:0 0 10px">' + esc(T.busyStop(n)) + '</p>' +
          '<button class="secondary" data-act="stop-now">' + esc(T.stopAnyway) + '</button> ' +
          '<button class="link" data-act="stop-cancel">' + esc(T.keepSharing) + '</button></div>';
      } else {
        confirmStop = false;
      }
    }
    if (s.netChanged) h += '<div class="notice ok">' + esc(T.netChanged) + '</div>';
    if (s.vpn) h += '<div class="notice">' + esc(T.vpn) + '</div>';
    if ((s.unreadable || []).length) h += '<div class="notice">' + esc(T.unreadable + s.unreadable.join(', ')) + '</div>';
    if (!main) return h + '<div class="notice">' + esc(T.noNetwork) + '</div>';
    h += '<div class="row"><div><div class="qr" id="qr" role="button" tabindex="0" aria-label="' + esc(T.qrHint) + '">' + s.qr + '</div>' +
      '<p class="small center" style="margin:8px 0 0">' + esc(T.qrHint) + '</p></div><div class="side">';
    h += '<p class="step">' + T.phone + '</p><p>' + T.pc + '</p>';
    h += '<div class="addr" id="addr">' + addrHTML(main.display) + '</div>';
    h += '<p style="margin-top:8px"><button class="secondary" data-act="copy">' + esc(T.copy) + '</button></p>';
    addrs.slice(1).forEach(function (a) {
      if (a.kind === 'mdns') h += '<p class="small">' + esc(T.alsoTry + a.display) + '</p>';
    });
    h += tunnelBlock(s) + '</div></div>';

    h += '<section aria-live="polite"><h2>' + esc(T.activity) + '</h2>' + activity(s);
    var st = s.stats || {};
    if (!st.clients && s.sharedAt && Date.now() - s.sharedAt > 45000) {
      h += '<div class="notice">' + T.cantEnter + T.cantEnterPlan2;
      if (s.firewallHint === 'windows' || navigator.userAgent.indexOf('Windows') >= 0) {
        h += '<button class="secondary" data-act="firewall">' + esc(T.checkWindows) + '</button>';
      }
      h += '</div>';
    }
    h += '</section>';

    if (!s.receiveOnly) {
      h += '<section><h2>' + esc(T.files) + '</h2><ul class="plain">';
      (s.files || []).forEach(function (f) { h += '<li>' + esc(f.name) + ' <span class="small">' + esc(f.size) + '</span></li>'; });
      h += '</ul></section>';
    }

    var cur = (s.ifaces || [])[0];
    h += '<p class="small foot">' + esc(T.network) + (cur ? esc(cur.human + ' (' + cur.ip + ')') : '—') +
      ' · <label>' + esc(T.changeNet) + ' <select id="iface"><option value="">' + esc(T.auto) + '</option>';
    (s.ifaces || []).forEach(function (c) {
      h += '<option value="' + esc(c.ip) + '">' + esc(c.human + ' (' + c.ip + ')') + '</option>';
    });
    h += '</select></label></p><p class="small">' + esc(T.tabHint) + '</p>';
    return h;
  }

  var confirmStop = false;

  function inFlight(s) {
    var st = s.stats || {}, n = (st.uploading || []).length;
    for (var k in (st.active || {})) n += st.active[k];
    return n;
  }

  function copyLink() {
    var main = (state.addresses || [])[0];
    if (!main) return;
    var ok = function () { toast(T.copied); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(main.url).then(ok, function () { toast(T.copyFail); });
    } else {
      toast(T.copyFail);
    }
  }

  // Si la pestaña está en segundo plano, el título cuenta lo que llegó: "(2) Pasame".
  var seenReceived = 0, unseen = 0;
  function trackReceived(s) {
    var n = ((s.stats || {}).received || []).length;
    if (s.phase !== 'sharing') n = 0;
    if (n < seenReceived) seenReceived = n;
    if (document.hidden && n > seenReceived) unseen += n - seenReceived;
    seenReceived = n;
    document.title = unseen ? '(' + unseen + ') Pasame' : 'Pasame';
  }
  document.addEventListener('visibilitychange', function () {
    if (!document.hidden) { unseen = 0; document.title = 'Pasame'; }
  });

  var detected = false;
  // Idioma: lo que eligió la persona gana; si no eligió, se toma del navegador. Español para
  // idiomas cercanos (pt, it, ca, gl); inglés para el resto, que es lo que más gente entiende.
  function applyLang(s) {
    T = TT[s.lang] || TT.es;
    document.documentElement.lang = s.lang || 'es';
    if (!s.langExplicit && !detected) {
      detected = true;
      var nav = (navigator.language || 'es').slice(0, 2).toLowerCase();
      var want = ['es', 'pt', 'it', 'ca', 'gl'].indexOf(nav) >= 0 ? 'es' : 'en';
      if (want !== s.lang) api('/api/lang', { lang: want, auto: true });
    }
    var b = document.getElementById('menu');
    b.setAttribute('aria-label', T.more);
    document.getElementById('m-star').textContent = T.menuStar;
    document.getElementById('m-made').textContent = T.menuMadeBy;
    document.getElementById('m-lang').textContent = T.otherLang;
    document.getElementById('m-lang').setAttribute('lang', s.lang === 'en' ? 'es' : 'en');
    document.getElementById('quit').textContent = T.quit;
  }

  function render() {
    if (!state || closed) return;
    // Re-render cada 5 s: no pisar un control que la persona está usando (se le cerraría el menú de red).
    var f = document.activeElement;
    if (f && (f.id === 'iface' || f.id === 'paste')) return;
    var detail = document.getElementById('detail'); // el detalle abierto se cerraría a los 5 s
    if (detail && !detail.hidden) return;
    renderWho(state);
    var app = document.getElementById('app');
    var qrFull = document.querySelector('.qr.full') !== null;
    app.innerHTML = state.phase === 'sharing' ? renderSharing(state) : renderIdle(state);
    if (qrFull && document.getElementById('qr')) document.getElementById('qr').classList.add('full');
    fitAddr();
  }

  // Solo se puede partir antes de "/s/…": el nombre (con guiones) nunca se corta al medio.
  function addrHTML(d) {
    var i = d.indexOf('/s/');
    if (i < 0) return esc(d);
    return '<span class="nw">' + esc(d.slice(0, i)) + '</span><wbr><span class="nw">' + esc(d.slice(i)) + '</span>';
  }

  // La dirección nunca se corta: se achica hasta entrar (y recién ahí se parte en "/s/").
  function fitAddr() {
    var el = document.getElementById('addr');
    if (!el) return;
    el.style.fontSize = '';
    el.classList.remove('wrap');
    var px = parseFloat(getComputedStyle(el).fontSize);
    while (el.scrollWidth > el.clientWidth && px > 16) { px -= 1; el.style.fontSize = px + 'px'; }
    if (el.scrollWidth <= el.clientWidth) return;
    el.classList.add('wrap');
    while (el.scrollWidth > el.clientWidth && px > 11) { px -= 1; el.style.fontSize = px + 'px'; }
  }

  function tunnelBlock(s) {
    var tn = s.tunnel || {}, h = '<div class="tunnel">';
    switch (tn.status) {
      case 'on':
        h += '<p>' + esc(T.askKey) + '</p><div class="addr pin">' + esc(tn.pin) + '</div>';
        h += '<p class="small">' + esc(T.privacy) + '</p><p class="small">' + esc(T.strictNote) + '</p>';
        h += '<button class="secondary" data-act="tunnel-off">' + esc(T.tunnelOff) + '</button>';
        break;
      case 'downloading':
        h += '<p>' + esc(T.preparing) + '</p><div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="' + (tn.progress | 0) + '"><div style="width:' + (tn.progress | 0) + '%"></div></div>';
        break;
      case 'connecting':
        h += '<p>' + esc(T.connecting) + '</p>';
        break;
      case 'unavailable':
        h += '<p class="small">' + esc(T.noTunnel) + '</p>';
        break;
      case 'error':
        h += '<div class="notice">' + esc(T.tunnelError) + ' <button class="link" data-act="detail" aria-expanded="false">' + esc(T.detail) +
          '</button><pre id="detail" hidden>' + esc(tn.detail) + '</pre></div>';
        /* falls through */
      default:
        if (tn.dropped) h += '<div class="notice">' + esc(T.dropped) + '</div>';
        h += '<p>' + esc(T.far) + ' <button class="secondary" data-act="tunnel-on">' + esc(T.tunnelOn) + '</button></p>';
    }
    return h + '</div>';
  }

  // Créditos: solo en la pantalla inicial y al cerrar. Nunca mientras se comparte (no distraer del QR).
  function credits() {
    return '<p class="credits">' + esc(T.oss) + ' <a href="https://github.com/KonixDev/pasame" target="_blank" rel="noopener">★ ' + esc(T.star) + '</a>.<br>' +
      esc(T.madeBy) + ' <a href="https://martincoll.dev" target="_blank" rel="noopener">martincoll.dev</a></p>';
  }

  function showClosed() {
    closed = true;
    document.getElementById('app').innerHTML = '<h1 class="center" style="margin-top:80px">' + esc(T.closedMsg) + '</h1>' + credits();
  }

  function toggleMenu(force) {
    var m = document.getElementById('menu-box'), b = document.getElementById('menu');
    var open = force === undefined ? m.hidden : force;
    m.hidden = !open;
    b.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (open) m.querySelector('[role=menuitem]').focus();
  }

  document.addEventListener('click', function (e) {
    // Click afuera cierra el menú.
    if (!e.target.closest('#menu, #menu-box') && !document.getElementById('menu-box').hidden) toggleMenu(false);
    var el = e.target.closest('[data-act], #qr, #edit-name, #menu, #quit, #m-lang');
    if (!el) return;
    if (el.id === 'qr') return el.classList.toggle('full');
    if (el.id === 'menu') return toggleMenu();
    if (el.id === 'quit') return api('/api/quit').then(showClosed);
    if (el.id === 'm-lang') { toggleMenu(false); return api('/api/lang', { lang: state.lang === 'en' ? 'es' : 'en' }); }
    if (el.id === 'edit-name') {
      var n = prompt(T.askName, state.name); // prompt: único diálogo, lo abre la persona
      if (n !== null) api('/api/name', { name: n });
      return;
    }
    switch (el.getAttribute('data-act')) {
      case 'pick-files': return api('/api/pick', { kind: 'files' });
      case 'pick-folder': return api('/api/pick', { kind: 'folder' });
      case 'receive-only': return api('/api/receive-only');
      case 'stop':
        if (inFlight(state) > 0 && !confirmStop) { confirmStop = true; return render(); }
        confirmStop = false;
        return api('/api/stop').then(function () { toast(T.stopped); });
      case 'stop-now': confirmStop = false; return api('/api/stop').then(function () { toast(T.stopped); });
      case 'stop-cancel': confirmStop = false; return render();
      case 'copy': return copyLink();
      case 'open-folder': return api('/api/open-folder');
      case 'firewall': return api('/api/firewall');
      case 'tunnel-on': return api('/api/tunnel', { on: true });
      case 'tunnel-off': return api('/api/tunnel', { on: false });
      case 'detail':
        var d = document.getElementById('detail');
        d.hidden = !d.hidden;
        el.setAttribute('aria-expanded', d.hidden ? 'false' : 'true');
        return;
      case 'paste':
        var v = document.getElementById('paste').value.trim();
        if (v) api('/api/add', { paths: [v] });
        return;
    }
  });

  window.addEventListener('resize', fitAddr);

  document.addEventListener('change', function (e) {
    if (e.target.id === 'iface') api('/api/iface', { ip: e.target.value });
  });

  // Soltar archivos sobre la página no puede funcionar (el navegador no da rutas): se explica.
  document.addEventListener('dragover', function (e) { e.preventDefault(); });
  document.addEventListener('drop', function (e) {
    e.preventDefault();
    var hint = document.getElementById('drop-hint');
    if (hint) hint.hidden = false; else toast(T.dropHint);
  });

  var es = new EventSource('/events?t=' + encodeURIComponent(t));
  es.addEventListener('state', function (e) { state = JSON.parse(e.data); applyLang(state); trackReceived(state); render(); });

  // Teclado: Esc cierra el QR a pantalla completa; Enter o espacio sobre el QR lo agranda.
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && !document.getElementById('menu-box').hidden) { toggleMenu(false); document.getElementById('menu').focus(); }
    var q = document.getElementById('qr');
    if (!q) return;
    if (e.key === 'Escape' && q.classList.contains('full')) q.classList.remove('full');
    if ((e.key === 'Enter' || e.key === ' ') && document.activeElement === q) { e.preventDefault(); q.classList.toggle('full'); }
  });
  es.addEventListener('quit', function () { es.close(); showClosed(); });
  var fails = 0;
  es.onopen = function () { fails = 0; };
  es.onerror = function () { if (++fails > 3) { es.close(); showClosed(); } };

  // El hint de 45 s depende del reloj, no de un evento: se re-evalúa solo.
  setInterval(render, 5000);
})();
