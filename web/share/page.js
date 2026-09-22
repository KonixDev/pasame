(function () {
  var form = document.getElementById('up-form');
  var input = document.getElementById('up-input');
  var send = document.getElementById('up-send');
  var status = document.getElementById('up-status');
  var wrap = document.getElementById('up-bar-wrap');
  var bar = document.getElementById('up-bar');
  var drop = document.getElementById('drop');
  var hint = document.getElementById('dl-hint');
  if (!form || !window.XMLHttpRequest || !window.FormData) return; // sin JS útil: queda el form clásico
  send.style.display = 'none';
  var busy = false;

  function say(msg, cls) {
    status.className = 'note' + (cls ? ' ' + cls : '');
    status.innerHTML = '';
    status.appendChild(document.createTextNode(msg));
  }

  function fmt(i, n, pct) {
    var parts = [i, n, pct], k = 0;
    return T.sending.replace(/0/g, function () { return parts[k++]; });
  }

  // Sube de a un archivo: si uno falla, los anteriores ya llegaron y se dice cuáles.
  function upload(files, i, sent) {
    if (i >= files.length) {
      busy = false;
      wrap.style.display = 'none';
      input.value = '';
      say(sent.length ? T.sentList.replace('__FILES__', sent.join(', ')) : T.done, 'ok');
      return;
    }
    busy = true;
    var fd = new FormData();
    fd.append('f', files[i]);
    var xhr = new XMLHttpRequest();
    xhr.open('POST', form.action);
    xhr.setRequestHeader('Accept', 'application/json');
    xhr.upload.onprogress = function (e) {
      if (!e.lengthComputable) return;
      var pct = Math.floor(e.loaded * 100 / e.total);
      bar.style.width = pct + '%';
      say(fmt(i + 1, files.length, pct));
    };
    xhr.onload = function () {
      var resp = {};
      try { resp = JSON.parse(xhr.responseText); } catch (e) {}
      if (xhr.status >= 200 && xhr.status < 300) {
        sent = sent.concat(resp.files || [files[i].name]);
        return upload(files, i + 1, sent);
      }
      fail(resp.error || T.fail);
    };
    xhr.onerror = function () { fail(T.fail); };
    wrap.style.display = 'block';
    bar.style.width = '0';
    xhr.send(fd);
  }

  function fail(msg) {
    busy = false;
    wrap.style.display = 'none';
    say(msg, 'err');
  }

  input.onchange = function () {
    if (input.files && input.files.length) upload(input.files, 0, []);
  };

  // Cerrar la pestaña a mitad de una subida la corta: el navegador pregunta antes.
  window.onbeforeunload = function () { if (busy) return true; };

  // Arrastrar y soltar (sobre todo en computadoras): toda la página es la zona de soltar.
  var depth = 0;
  function hasFiles(e) {
    var t = e.dataTransfer && e.dataTransfer.types;
    if (!t) return false;
    for (var k = 0; k < t.length; k++) if (t[k] === 'Files') return true;
    return false;
  }
  if (drop && document.addEventListener) {
    document.addEventListener('dragenter', function (e) { if (hasFiles(e)) { depth++; drop.className = 'on'; } }, false);
    document.addEventListener('dragleave', function () { if (--depth <= 0) { depth = 0; drop.className = ''; } }, false);
    document.addEventListener('dragover', function (e) { if (hasFiles(e)) e.preventDefault(); }, false);
    document.addEventListener('drop', function (e) {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth = 0;
      drop.className = '';
      if (!busy && e.dataTransfer.files.length) upload(e.dataTransfer.files, 0, []);
    }, false);
  }

  // Tocar una descarga no muestra nada en la página: se avisa que empezó y dónde queda.
  function showHint() {
    if (!hint) return;
    hint.style.display = 'block';
    clearTimeout(showHint.t);
    showHint.t = setTimeout(function () { hint.style.display = 'none'; }, 9000);
  }
  var links = document.getElementsByTagName('a');
  for (var j = 0; j < links.length; j++) {
    var a = links[j];
    if (a.id === 'dl-all' || a.className === 'name') a.onclick = showHint;
  }
})();
