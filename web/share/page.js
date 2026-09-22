(function () {
  var form = document.getElementById('up-form');
  var input = document.getElementById('up-input');
  var send = document.getElementById('up-send');
  var status = document.getElementById('up-status');
  var wrap = document.getElementById('up-bar-wrap');
  var bar = document.getElementById('up-bar');
  if (!form || !window.XMLHttpRequest || !window.FormData) return; // sin JS útil: queda el form clásico
  send.style.display = 'none';

  function fmt(i, n, pct) {
    var parts = [i, n, pct], k = 0;
    return T.sending.replace(/0/g, function () { return parts[k++]; });
  }

  function upload(files, i) {
    if (i >= files.length) {
      wrap.style.display = 'none';
      status.className = 'note ok';
      status.innerHTML = '';
      status.appendChild(document.createTextNode(T.done));
      input.value = '';
      return;
    }
    var fd = new FormData();
    fd.append('f', files[i]);
    var xhr = new XMLHttpRequest();
    xhr.open('POST', form.action);
    xhr.setRequestHeader('Accept', 'application/json');
    xhr.upload.onprogress = function (e) {
      if (!e.lengthComputable) return;
      var pct = Math.floor(e.loaded * 100 / e.total);
      bar.style.width = pct + '%';
      status.innerHTML = '';
      status.appendChild(document.createTextNode(fmt(i + 1, files.length, pct)));
    };
    xhr.onload = function () {
      if (xhr.status >= 200 && xhr.status < 300) return upload(files, i + 1);
      var msg = T.fail;
      try { msg = JSON.parse(xhr.responseText).error || msg; } catch (e) {}
      fail(msg);
    };
    xhr.onerror = function () { fail(T.fail); };
    wrap.style.display = 'block';
    bar.style.width = '0';
    xhr.send(fd);
  }

  function fail(msg) {
    wrap.style.display = 'none';
    status.className = 'note err';
    status.innerHTML = '';
    status.appendChild(document.createTextNode(msg));
  }

  input.onchange = function () {
    if (input.files && input.files.length) upload(input.files, 0);
  };
})();
