package api

import (
	"fmt"
	"net/http"
)

// TestPage menyajikan halaman uji koneksi (login QR/pairing, status, kirim pesan).
// Halaman ini statis; API key dimasukkan user lewat form lalu dipakai di header fetch.
func (h *Handler) TestPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, testPageHTML)
}

const testPageHTML = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WhatsMeow Gateway — Test Connect</title>
<style>
  :root{--bg:#0f172a;--surface:#1e293b;--border:#334155;--text:#e2e8f0;--muted:#94a3b8;--accent:#22c55e;--danger:#ef4444;--warn:#f59e0b}
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:system-ui,-apple-system,sans-serif;background:var(--bg);color:var(--text);line-height:1.5;padding-bottom:3rem}
  a{color:var(--accent)}
  header{background:var(--surface);border-bottom:1px solid var(--border);padding:1rem 1.5rem;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:.5rem;position:sticky;top:0;z-index:10}
  header h1{font-size:1.1rem}
  .dot{display:inline-block;width:10px;height:10px;border-radius:50%;background:var(--muted);margin-right:.4rem;vertical-align:middle}
  .dot.on{background:var(--accent);box-shadow:0 0 8px var(--accent)}
  .dot.off{background:var(--danger)}
  .wrap{max-width:960px;margin:1.5rem auto;padding:0 1rem;display:grid;grid-template-columns:1fr 1fr;gap:1rem}
  .card{background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:1.25rem}
  .card.full{grid-column:1 / -1}
  h2{font-size:1rem;margin-bottom:.75rem;color:#f1f5f9}
  label{display:block;font-size:.8rem;color:var(--muted);margin:.5rem 0 .25rem}
  input,textarea{width:100%;padding:.6rem .75rem;border:1px solid var(--border);border-radius:8px;background:#0c1222;color:#f1f5f9;font-size:.9rem;font-family:inherit}
  input:focus,textarea:focus{outline:none;border-color:var(--accent)}
  textarea{resize:vertical;min-height:70px}
  .row{display:flex;gap:.5rem;flex-wrap:wrap;margin-top:.75rem}
  button{padding:.55rem .9rem;border:none;border-radius:8px;font-weight:600;font-size:.85rem;cursor:pointer;background:var(--accent);color:#052e16}
  button:hover{filter:brightness(1.1)}
  button.sec{background:#334155;color:#e2e8f0}
  button.danger{background:var(--danger);color:#fff}
  button.warn{background:var(--warn);color:#1c1300}
  button:disabled{opacity:.5;cursor:not-allowed}
  pre{background:#0c1222;border:1px solid var(--border);border-radius:8px;padding:.75rem;font-size:.78rem;overflow:auto;max-height:240px;white-space:pre-wrap;word-break:break-word}
  .qrbox{display:flex;flex-direction:column;align-items:center;text-align:center;gap:.5rem}
  .qrbox img{width:240px;height:240px;background:#fff;border-radius:8px;padding:6px}
  .muted{color:var(--muted);font-size:.8rem}
  .pill{display:inline-block;padding:.15rem .55rem;border-radius:999px;font-size:.72rem;font-weight:600}
  .pill.green{background:#14532d;color:#86efac}
  .pill.red{background:#450a0a;color:#fca5a5}
  .pill.gray{background:#334155;color:#cbd5e1}
  .code{font-family:ui-monospace,monospace;font-size:1.6rem;letter-spacing:.25rem;color:#fff;background:#0c1222;border:1px dashed var(--accent);border-radius:8px;padding:.6rem 1rem;display:inline-block;margin-top:.5rem}
  @media(max-width:760px){.wrap{grid-template-columns:1fr}}
</style>
</head>
<body>
<header>
  <h1>WhatsMeow Gateway — Test Connect</h1>
  <div><span id="dot" class="dot"></span><span id="connText" class="muted">menghubungkan...</span> &middot; <a href="/docs">Docs</a></div>
</header>

<div class="wrap">

  <div class="card full">
    <h2>API Key &amp; Session</h2>
    <label>X-API-Key (disimpan di browser ini saja)</label>
    <input id="apiKey" type="password" placeholder="Masukkan API key dari .env">
    <label>Session ID (tenant) &mdash; kosong = "default"</label>
    <input id="sessionId" type="text" placeholder="default">
    <div class="row">
      <button onclick="saveKey()">Simpan</button>
      <button class="sec" onclick="refreshStatus()">Cek Status</button>
    </div>
    <p class="muted" style="margin-top:.5rem">Tiap Session ID = satu nomor WhatsApp terpisah. Semua aksi di bawah berlaku untuk session ini.</p>
  </div>

  <div class="card">
    <h2>Status Session</h2>
    <div id="statusPills"><span class="pill gray">memuat...</span></div>
    <pre id="statusBox">-</pre>
    <div class="row">
      <button onclick="doConnect()">Connect</button>
      <button class="sec" onclick="apiCall('/session/disconnect','POST')">Disconnect</button>
      <button class="warn" onclick="apiCall('/session/reset','POST')">Reset</button>
      <button class="danger" onclick="if(confirm('Logout dan hapus session?'))apiCall('/session/logout','POST')">Logout</button>
    </div>
  </div>

  <div class="card">
    <h2>Login via QR</h2>
    <div class="qrbox">
      <img id="qrImg" alt="QR akan muncul di sini" style="display:none">
      <div id="qrMsg" class="muted">Klik Connect untuk memunculkan QR. Scan dari WhatsApp &gt; Perangkat Tertaut.</div>
      <div id="qrTimer" class="muted"></div>
    </div>
  </div>

  <div class="card">
    <h2>Login via Pairing Code</h2>
    <label>Nomor (format 628xxx, tanpa +)</label>
    <input id="pairPhone" type="text" placeholder="628123456789">
    <div class="row"><button onclick="doPair()">Minta Kode</button></div>
    <div id="pairCodeBox"></div>
    <p class="muted" style="margin-top:.5rem">Di HP: Perangkat Tertaut &gt; Tautkan perangkat &gt; Tautkan dengan nomor telepon.</p>
  </div>

  <div class="card">
    <h2>Kirim Pesan Test</h2>
    <label>Tujuan (nomor/JID)</label>
    <input id="msgTo" type="text" placeholder="628123456789">
    <label>Teks</label>
    <textarea id="msgText" placeholder="Halo dari test page!"></textarea>
    <div class="row"><button onclick="doSend()" id="sendBtn">Kirim</button></div>
  </div>

  <div class="card">
    <h2>Webhook</h2>
    <label>URL penerima pesan masuk</label>
    <input id="webhookUrl" type="text" placeholder="http://localhost/whatsmeow/webhook.php">
    <div class="row">
      <button onclick="getWebhook()">Lihat</button>
      <button class="sec" onclick="setWebhook()">Simpan</button>
    </div>
  </div>

  <div class="card full">
    <h2>Log Respons</h2>
    <pre id="log">siap.</pre>
  </div>

</div>

<script>
var KEY_STORE = 'whatsmeow_test_apikey';
var SESSION_STORE = 'whatsmeow_test_session';
var qrTimer = null;
var qrSecondsLeft = 0;

function getKey(){ return document.getElementById('apiKey').value.trim(); }
function getSession(){ return document.getElementById('sessionId').value.trim() || 'default'; }
function saveKey(){
  localStorage.setItem(KEY_STORE, getKey());
  localStorage.setItem(SESSION_STORE, getSession());
  log('API key & session (' + getSession() + ') disimpan.');
  refreshStatus();
}
function ts(){ return new Date().toLocaleTimeString(); }
function log(msg, obj){
  var el = document.getElementById('log');
  var line = '[' + ts() + '] ' + msg;
  if(obj !== undefined){ line += '\n' + JSON.stringify(obj, null, 2); }
  el.textContent = line + '\n\n' + el.textContent;
}

function api(path, method, body){
  var opts = { method: method || 'GET', headers: { 'X-API-Key': getKey(), 'X-Session-ID': getSession() } };
  if(body){ opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  return fetch(path, opts).then(function(r){
    return r.json().catch(function(){ return { success:false, message:'respons bukan JSON (HTTP '+r.status+')' }; })
      .then(function(j){ return { status:r.status, body:j }; });
  });
}

function apiCall(path, method, body){
  return api(path, method, body).then(function(res){
    log(method + ' ' + path + ' -> ' + res.status, res.body);
    refreshStatus();
    return res;
  }).catch(function(e){ log('ERROR ' + path + ': ' + e.message); });
}

function setConnDot(connected, loggedIn){
  var dot = document.getElementById('dot');
  var txt = document.getElementById('connText');
  dot.className = 'dot ' + (connected ? 'on' : 'off');
  txt.textContent = loggedIn ? 'login aktif' : (connected ? 'terhubung (belum login)' : 'terputus');
}

function renderPills(d){
  var pills = '';
  pills += '<span class="pill ' + (d.connected?'green':'red') + '">' + (d.connected?'connected':'disconnected') + '</span> ';
  pills += '<span class="pill ' + (d.logged_in?'green':'gray') + '">' + (d.logged_in?'logged in':'belum login') + '</span>';
  if(d.push_name){ pills += ' <span class="pill gray">' + d.push_name + '</span>'; }
  document.getElementById('statusPills').innerHTML = pills;
}

function refreshStatus(){
  if(!getKey()){ document.getElementById('statusBox').textContent = 'Masukkan API key dulu.'; return Promise.resolve(); }
  return api('/session/status').then(function(res){
    if(res.status === 200 && res.body.success){
      var d = res.body.data;
      document.getElementById('statusBox').textContent = JSON.stringify(d, null, 2);
      renderPills(d);
      setConnDot(d.connected, d.logged_in);
      if(d.logged_in){ hideQR(); }
    } else {
      document.getElementById('statusBox').textContent = JSON.stringify(res.body, null, 2);
      if(res.status === 401){ document.getElementById('statusPills').innerHTML = '<span class="pill red">API key salah</span>'; }
    }
  });
}

function doConnect(){ apiCall('/session/connect','POST').then(function(){ setTimeout(loadQR, 1500); }); }

function hideQR(){
  document.getElementById('qrImg').style.display = 'none';
  document.getElementById('qrMsg').textContent = 'Sudah login. QR tidak diperlukan.';
  document.getElementById('qrTimer').textContent = '';
}

function loadQR(){
  if(!getKey()) return;
  api('/session/qr').then(function(res){
    var img = document.getElementById('qrImg');
    var msg = document.getElementById('qrMsg');
    if(res.status === 200 && res.body.success && res.body.data.qr_base64){
      img.src = res.body.data.qr_base64;
      img.style.display = 'block';
      msg.textContent = 'Scan QR ini dari WhatsApp di HP.';
      qrSecondsLeft = res.body.data.timeout_seconds || 60;
    } else {
      var d = res.body.data || {};
      if(d.event === 'success'){ hideQR(); }
      else { img.style.display = 'none'; msg.textContent = 'QR belum siap (' + (d.event||res.body.message||'-') + '). Klik Connect.'; }
    }
  });
}

function startTimer(){
  if(qrTimer) return;
  qrTimer = setInterval(function(){
    if(qrSecondsLeft > 0){
      qrSecondsLeft--;
      var t = document.getElementById('qrTimer');
      if(document.getElementById('qrImg').style.display !== 'none'){ t.textContent = 'QR refresh dalam ~' + qrSecondsLeft + 's'; }
    }
  }, 1000);
}

function doPair(){
  var phone = document.getElementById('pairPhone').value.trim();
  if(!phone){ alert('Isi nomor dulu'); return; }
  apiCall('/session/pair','POST',{ phone: phone }).then(function(res){
    if(res && res.body && res.body.success && res.body.data.code){
      document.getElementById('pairCodeBox').innerHTML = '<div class="code">' + res.body.data.code + '</div>';
    }
  });
}

function doSend(){
  var to = document.getElementById('msgTo').value.trim();
  var text = document.getElementById('msgText').value;
  if(!to || !text){ alert('Isi tujuan dan teks'); return; }
  var btn = document.getElementById('sendBtn');
  btn.disabled = true;
  apiCall('/message/text','POST',{ to: to, text: text }).then(function(){ btn.disabled = false; });
}

function getWebhook(){ apiCall('/webhook','GET').then(function(res){ if(res&&res.body&&res.body.success){ document.getElementById('webhookUrl').value = res.body.data.webhook_url || ''; } }); }
function setWebhook(){ var url = document.getElementById('webhookUrl').value.trim(); apiCall('/webhook','PUT',{ url: url }); }

// init
(function(){
  var saved = localStorage.getItem(KEY_STORE);
  if(saved){ document.getElementById('apiKey').value = saved; }
  var savedSession = localStorage.getItem(SESSION_STORE);
  if(savedSession){ document.getElementById('sessionId').value = savedSession; }
  refreshStatus();
  startTimer();
  setInterval(function(){
    if(!getKey()) return;
    refreshStatus();
  }, 4000);
  setInterval(function(){
    if(!getKey()) return;
    var pill = document.getElementById('statusPills').textContent;
    if(pill.indexOf('logged in') === -1){ loadQR(); }
  }, 6000);
})();
</script>
</body>
</html>`
