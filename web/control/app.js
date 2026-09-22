(function () {
  'use strict';
  var t = new URLSearchParams(location.search).get('t') || '';
  var state = null;
  var closed = false;

  // Todos los textos del emisor (spec §13). Voz rioplatense, sin palabras técnicas.
  var T = {
    who: 'Tu nombre: ',
    edit: '✎',
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
    nobody: '● Nadie entró todavía. Tiene que estar en la misma red WiFi.',
    devices: function (n) { return '● ' + n + (n === 1 ? ' dispositivo conectado' : ' dispositivos conectados'); },
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
    closedMsg: 'Pasame está cerrado. Podés cerrar esta pestaña.',
    dropHint: 'Para elegir archivos usá el botón (o arrastralos sobre el ícono de Pasame).',
    noNetwork: 'Esta computadora no está conectada a ninguna red. Conectate a un WiFi y esperá unos segundos.'
  };
  T.cantEnterPlan2 = ''; // el Plan 2 agrega acá el paso "Tocá Compartir por internet"

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
      return r.json().then(function (j) { toast(j.error || 'No se pudo.'); }, function () { toast('No se pudo.'); });
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
    document.getElementById('who').innerHTML = esc(T.who) + '<b>' + esc(s.name) + '</b> <button class="link" id="edit-name" aria-label="Cambiar nombre">' + T.edit + '</button>';
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
    h += '<p class="small" id="drop-hint" hidden>' + esc(T.dropHint) + '</p></div>';
    return h;
  }

  function activity(s) {
    var st = s.stats || {}, lines = [];
    lines.push(st.clients ? T.devices(st.clients) : T.nobody);
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
    if (s.netChanged) h += '<div class="notice ok">' + esc(T.netChanged) + '</div>';
    if (s.vpn) h += '<div class="notice">' + esc(T.vpn) + '</div>';
    if ((s.unreadable || []).length) h += '<div class="notice">' + esc(T.unreadable + s.unreadable.join(', ')) + '</div>';
    if (!main) return h + '<div class="notice">' + esc(T.noNetwork) + '</div>';
    h += '<div class="row"><div class="qr" id="qr" title="Tocá para agrandar">' + s.qr + '</div><div class="side">';
    h += '<p>' + T.phone + '</p><p style="margin-top:24px">' + T.pc + '</p>';
    h += '<div class="addr">' + esc(main.display) + '</div>';
    addrs.slice(1).forEach(function (a) {
      if (a.kind === 'mdns') h += '<p class="small">' + esc(T.alsoTry + a.display) + '</p>';
    });
    h += '<div id="plan2-slot"></div></div></div>';

    h += '<section><h2>' + esc(T.activity) + '</h2>' + activity(s);
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
    h += '<p class="small" style="margin-top:28px">' + esc(T.network) + (cur ? esc(cur.human + ' (' + cur.ip + ')') : '—') +
      ' · <label>' + esc(T.changeNet) + ' <select id="iface"><option value="">' + esc(T.auto) + '</option>';
    (s.ifaces || []).forEach(function (c) {
      h += '<option value="' + esc(c.ip) + '">' + esc(c.human + ' (' + c.ip + ')') + '</option>';
    });
    h += '</select></label></p><p class="small">' + esc(T.tabHint) + '</p>';
    return h;
  }

  function render() {
    if (!state || closed) return;
    // Re-render cada 5 s: no pisar un control que la persona está usando (se le cerraría el menú de red).
    var f = document.activeElement;
    if (f && (f.id === 'iface' || f.id === 'paste')) return;
    renderWho(state);
    var app = document.getElementById('app');
    var qrFull = document.querySelector('.qr.full') !== null;
    app.innerHTML = state.phase === 'sharing' ? renderSharing(state) : renderIdle(state);
    if (qrFull && document.getElementById('qr')) document.getElementById('qr').classList.add('full');
  }

  function showClosed() {
    closed = true;
    document.getElementById('app').innerHTML = '<h1 class="center" style="margin-top:80px">' + esc(T.closedMsg) + '</h1>';
  }

  document.addEventListener('click', function (e) {
    var el = e.target.closest('[data-act], #qr, #edit-name, #menu, #quit');
    if (!el) return;
    if (el.id === 'qr') return el.classList.toggle('full');
    if (el.id === 'menu') { var m = document.getElementById('menu-box'); m.hidden = !m.hidden; return; }
    if (el.id === 'quit') return api('/api/quit').then(showClosed);
    if (el.id === 'edit-name') {
      var n = prompt('¿Cómo te llamás? (lo ve quien recibe)', state.name); // prompt: único diálogo, lo abre la persona
      if (n !== null) api('/api/name', { name: n });
      return;
    }
    switch (el.getAttribute('data-act')) {
      case 'pick-files': return api('/api/pick', { kind: 'files' });
      case 'pick-folder': return api('/api/pick', { kind: 'folder' });
      case 'receive-only': return api('/api/receive-only');
      case 'stop': return api('/api/stop').then(function () { toast(T.stopped); });
      case 'open-folder': return api('/api/open-folder');
      case 'firewall': return api('/api/firewall');
      case 'paste':
        var v = document.getElementById('paste').value.trim();
        if (v) api('/api/add', { paths: [v] });
        return;
    }
  });

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
  es.addEventListener('state', function (e) { state = JSON.parse(e.data); render(); });
  es.addEventListener('quit', function () { es.close(); showClosed(); });
  var fails = 0;
  es.onopen = function () { fails = 0; };
  es.onerror = function () { if (++fails > 3) { es.close(); showClosed(); } };

  // El hint de 45 s depende del reloj, no de un evento: se re-evalúa solo.
  setInterval(render, 5000);
})();
