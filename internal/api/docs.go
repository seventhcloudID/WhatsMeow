package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const docsCookieName = "gateway_docs"

func docsToken(apiKey string) string {
	m := hmac.New(sha256.New, []byte("whatsmeow-gateway-docs"))
	m.Write([]byte(apiKey))
	return hex.EncodeToString(m.Sum(nil))
}

func (h *Handler) docsAuthorized(r *http.Request) bool {
	key := extractAPIKey(r)
	if key != "" && subtle.ConstantTimeCompare([]byte(key), []byte(h.apiKey)) == 1 {
		return true
	}
	cookie, err := r.Cookie(docsCookieName)
	if err != nil {
		return false
	}
	expected := docsToken(h.apiKey)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) == 1
}

func extractAPIKey(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("key")
}

func (h *Handler) setDocsCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     docsCookieName,
		Value:    docsToken(h.apiKey),
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func (h *Handler) Docs(w http.ResponseWriter, r *http.Request) {
	if key := r.URL.Query().Get("key"); key != "" {
		if subtle.ConstantTimeCompare([]byte(key), []byte(h.apiKey)) == 1 {
			h.setDocsCookie(w, r)
			http.Redirect(w, r, "/docs", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/docs?error=1", http.StatusSeeOther)
		return
	}

	if !h.docsAuthorized(r) {
		h.writeDocsLogin(w, r.URL.Query().Get("error") == "1")
		return
	}

	baseURL := docsBaseURL(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderDocsPage(baseURL))
}

func (h *Handler) DocsLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/docs?error=1", http.StatusSeeOther)
		return
	}

	key := r.FormValue("api_key")
	if subtle.ConstantTimeCompare([]byte(key), []byte(h.apiKey)) != 1 {
		http.Redirect(w, r, "/docs?error=1", http.StatusSeeOther)
		return
	}

	h.setDocsCookie(w, r)
	http.Redirect(w, r, "/docs", http.StatusSeeOther)
}

func (h *Handler) DocsLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     docsCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/docs", http.StatusSeeOther)
}

func docsBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8081"
	}
	return scheme + "://" + host
}

func (h *Handler) writeDocsLogin(w http.ResponseWriter, showError bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	errMsg := ""
	if showError {
		errMsg = `<p class="error">Secret key salah. Coba lagi.</p>`
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WhatsMeow Gateway — Login Docs</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:system-ui,-apple-system,sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:1rem}
  .card{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:2rem;width:100%%;max-width:400px;box-shadow:0 25px 50px -12px rgba(0,0,0,.5)}
  h1{font-size:1.25rem;margin-bottom:.5rem;color:#f8fafc}
  p.sub{color:#94a3b8;font-size:.875rem;margin-bottom:1.5rem}
  label{display:block;font-size:.875rem;color:#cbd5e1;margin-bottom:.5rem}
  input[type=password]{width:100%%;padding:.75rem 1rem;border:1px solid #475569;border-radius:8px;background:#0f172a;color:#f1f5f9;font-size:1rem;margin-bottom:1rem}
  input:focus{outline:none;border-color:#22c55e}
  button{width:100%%;padding:.75rem;background:#22c55e;color:#052e16;border:none;border-radius:8px;font-weight:600;font-size:1rem;cursor:pointer}
  button:hover{background:#16a34a}
  .error{background:#450a0a;color:#fca5a5;padding:.75rem;border-radius:8px;font-size:.875rem;margin-bottom:1rem}
</style>
</head>
<body>
<div class="card">
  <h1>WhatsMeow Gateway</h1>
  <p class="sub">Masukkan secret key untuk melihat dokumentasi API.</p>
  %s
  <form method="POST" action="/docs/login">
    <label for="api_key">Secret Key (API Key)</label>
    <input type="password" id="api_key" name="api_key" placeholder="Masukkan API key..." required autofocus>
    <button type="submit">Masuk</button>
  </form>
</div>
</body>
</html>`, errMsg)
}

// renderDocsPage builds the interactive API docs page.
//
// Untuk menambah / mengubah dokumentasi endpoint, cukup edit blok HTML di bawah.
// Gunakan placeholder {BASE} untuk base URL (otomatis diganti) dan {YEAR} untuk tahun.
// Tidak perlu lagi menghitung argumen %s — jauh lebih mudah diedit.
func renderDocsPage(baseURL string) string {
	page := docsPageTemplate
	page = strings.ReplaceAll(page, "{BASE}", baseURL)
	page = strings.ReplaceAll(page, "{YEAR}", strconv.Itoa(time.Now().Year()))
	return page
}

const docsPageTemplate = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WhatsMeow Gateway — API Docs</title>
<style>
  :root{--bg:#0f172a;--surface:#1e293b;--border:#334155;--text:#e2e8f0;--muted:#94a3b8;--accent:#22c55e;--accent-dim:#166534;--get:#3b82f6;--post:#22c55e;--put:#f59e0b}
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:system-ui,-apple-system,sans-serif;background:var(--bg);color:var(--text);line-height:1.6}
  a{color:var(--accent);text-decoration:none}
  a:hover{text-decoration:underline}
  header{background:var(--surface);border-bottom:1px solid var(--border);padding:1rem 2rem;display:flex;justify-content:space-between;align-items:center;position:sticky;top:0;z-index:10}
  header h1{font-size:1.125rem;font-weight:600}
  header .base{font-size:.8rem;color:var(--muted);font-family:monospace}
  .logout{font-size:.875rem;color:var(--muted)}
  .layout{display:flex;max-width:1200px;margin:0 auto}
  nav{width:220px;padding:1.5rem 1rem;position:sticky;top:57px;height:calc(100vh - 57px);overflow-y:auto;border-right:1px solid var(--border);flex-shrink:0}
  nav a{display:block;padding:.4rem .75rem;color:var(--muted);font-size:.875rem;border-radius:6px;margin-bottom:2px}
  nav a:hover{background:var(--surface);color:var(--text);text-decoration:none}
  nav .group{font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;margin:1rem .75rem .25rem}
  main{flex:1;padding:2rem;min-width:0}
  section{margin-bottom:2.5rem}
  h2{font-size:1.375rem;margin-bottom:1rem;padding-bottom:.5rem;border-bottom:1px solid var(--border)}
  h3{font-size:1rem;margin:1.25rem 0 .75rem;color:#f1f5f9}
  p,li{color:var(--muted);margin-bottom:.75rem}
  ul{padding-left:1.25rem;margin-bottom:1rem}
  .endpoint{background:var(--surface);border:1px solid var(--border);border-radius:10px;margin-bottom:1rem;overflow:hidden}
  .ep-head{display:flex;align-items:center;gap:.75rem;padding:.875rem 1rem;border-bottom:1px solid var(--border);flex-wrap:wrap}
  .method{font-size:.75rem;font-weight:700;padding:.25rem .6rem;border-radius:4px;font-family:monospace;min-width:52px;text-align:center}
  .method.get{background:#1e3a5f;color:#93c5fd}
  .method.post{background:#14532d;color:#86efac}
  .method.put{background:#451a03;color:#fcd34d}
  .path{font-family:monospace;font-size:.9rem;color:#f1f5f9}
  .ep-body{padding:1rem}
  pre{background:#0c1222;border:1px solid var(--border);border-radius:8px;padding:1rem;overflow-x:auto;font-size:.8rem;line-height:1.5;margin:.5rem 0;color:#cbd5e1}
  code{font-family:ui-monospace,monospace;font-size:.85em}
  table{width:100%;border-collapse:collapse;font-size:.875rem;margin:.75rem 0}
  th,td{text-align:left;padding:.5rem .75rem;border-bottom:1px solid var(--border);vertical-align:top}
  th{color:var(--muted);font-weight:500}
  .badge{display:inline-block;background:var(--accent-dim);color:#bbf7d0;font-size:.75rem;padding:.15rem .5rem;border-radius:4px}
  .req{color:#fca5a5;font-size:.75rem}
  footer{text-align:center;padding:2rem;color:var(--muted);font-size:.8rem;border-top:1px solid var(--border)}
  @media(max-width:768px){.layout{flex-direction:column}nav{width:100%;height:auto;position:static;border-right:none;border-bottom:1px solid var(--border)}}
</style>
</head>
<body>
<header>
  <div>
    <h1>WhatsMeow Gateway API</h1>
    <div class="base">{BASE}</div>
  </div>
  <form method="POST" action="/docs/logout" style="margin:0"><button type="submit" class="logout" style="background:none;border:none;cursor:pointer;color:var(--muted);font-size:.875rem">Keluar</button></form>
</header>
<div class="layout">
<nav>
  <a href="#auth">Autentikasi</a>
  <div class="group">Dasar</div>
  <a href="#session">Session</a>
  <a href="#message">Kirim Pesan</a>
  <a href="#message-advanced">Pesan Lanjutan</a>
  <div class="group">Lanjutan</div>
  <a href="#groups">Grup</a>
  <a href="#users">User</a>
  <a href="#presence">Presence</a>
  <a href="#chats">Chat</a>
  <a href="#media">Media</a>
  <a href="#newsletter">Newsletter</a>
  <a href="#webhook">Webhook</a>
  <div class="group">Lainnya</div>
  <a href="#errors">Response &amp; Error</a>
  <a href="#fullapi">Daftar Lengkap</a>
</nav>
<main>

<section id="auth">
  <h2>Autentikasi</h2>
  <p>Semua endpoint API (kecuali <code>/health</code> dan <code>/docs</code>) memerlukan secret key via header:</p>
  <pre>X-API-Key: YOUR_SECRET_KEY</pre>
  <p>Alternatif:</p>
  <pre>Authorization: Bearer YOUR_SECRET_KEY</pre>
  <h3 style="margin-top:1rem">Multi-session (multi-tenant)</h3>
  <p>Satu gateway bisa melayani banyak nomor WhatsApp. Pilih tenant dengan header opsional:</p>
  <pre>X-Session-ID: nama-tenant</pre>
  <p>Tanpa header ini, dipakai session <code>default</code>. Semua endpoint session/message/groups/users/dst berlaku untuk session pada header tersebut. Kelola daftar tenant lewat endpoint <code>/sessions</code>:</p>
  <pre>curl -H "X-API-Key: KEY" {BASE}/sessions
curl -X POST -H "X-API-Key: KEY" -d '{"id":"tenant-1","label":"Toko A"}' {BASE}/sessions
curl -X DELETE -H "X-API-Key: KEY" {BASE}/sessions/tenant-1</pre>
  <p>Webhook diset per-session: kirim <code>PUT /webhook</code> dengan header <code>X-Session-ID</code> tenant tersebut. Payload webhook menyertakan field <code>session_id</code> agar penerima tahu asal pesannya.</p>
  <p>Format nomor tujuan: <code>6281234567890</code> (kode negara, tanpa <code>+</code>) atau JID penuh <code>628...@s.whatsapp.net</code>. JID grup berakhiran <code>@g.us</code>, channel/newsletter <code>@newsletter</code>.</p>
</section>

<section id="session">
  <h2>Session</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/session/status</span></div>
    <div class="ep-body"><p>Cek status koneksi dan login WhatsApp.</p>
    <pre>curl -H "X-API-Key: KEY" {BASE}/session/status</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/connect</span></div>
    <div class="ep-body"><p>Mulai koneksi ke WhatsApp. Jika belum login, QR code akan di-generate.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" {BASE}/session/connect</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/session/qr</span></div>
    <div class="ep-body"><p>Ambil QR code untuk scan login. Response berisi <code>code</code> dan <code>qr_base64</code> (PNG).</p>
    <pre>curl -H "X-API-Key: KEY" {BASE}/session/qr</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/pair</span></div>
    <div class="ep-body"><p>Login via pairing code (tanpa scan QR). Kode akan muncul untuk dimasukkan di HP.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"phone":"6281234567890"}' \
  {BASE}/session/pair</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/reset</span></div>
    <div class="ep-body"><p>Reset session (hapus state lokal) untuk memperbaiki link device yang gagal, lalu login ulang.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" {BASE}/session/reset</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/logout</span></div>
    <div class="ep-body"><p>Logout dan hapus session WhatsApp.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" {BASE}/session/logout</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/disconnect</span></div>
    <div class="ep-body"><p>Putuskan koneksi sementara (session tetap tersimpan).</p>
    <pre>curl -X POST -H "X-API-Key: KEY" {BASE}/session/disconnect</pre></div>
  </div>
</section>

<section id="message">
  <h2>Kirim Pesan</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/text</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Nomor tujuan</td></tr>
    <tr><td>text <span class="req">*</span></td><td>string</td><td>Isi pesan</td></tr>
    <tr><td>reply_to</td><td>string</td><td>Message ID yang dibalas (opsional)</td></tr>
    <tr><td>reply_jid</td><td>string</td><td>JID chat konteks balasan (opsional)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","text":"Halo!"}' \
  {BASE}/message/text</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/image</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>Nomor tujuan</td></tr>
    <tr><td>file <span class="req">*</span></td><td>File gambar</td></tr>
    <tr><td>caption</td><td>Caption opsional</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "caption=Caption" \
  -F "file=@/path/gambar.jpg" \
  {BASE}/message/image</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/video</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>Nomor tujuan</td></tr>
    <tr><td>file <span class="req">*</span></td><td>File video (maks 64MB)</td></tr>
    <tr><td>caption</td><td>Caption opsional</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "file=@/path/video.mp4" \
  {BASE}/message/video</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/audio</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>Nomor tujuan</td></tr>
    <tr><td>file <span class="req">*</span></td><td>File audio</td></tr>
    <tr><td>ptt</td><td><code>true</code>/<code>1</code> untuk voice note</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "ptt=true" \
  -F "file=@/path/voice.ogg" \
  {BASE}/message/audio</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/document</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>Nomor tujuan</td></tr>
    <tr><td>file <span class="req">*</span></td><td>File dokumen</td></tr>
    <tr><td>filename</td><td>Nama file opsional</td></tr>
    <tr><td>mimetype</td><td>MIME type opsional</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "filename=laporan.pdf" \
  -F "file=@/path/laporan.pdf" \
  {BASE}/message/document</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/sticker</span><span class="badge">multipart</span></div>
    <div class="ep-body"><p>Kirim sticker (disarankan format WebP).</p>
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>Nomor tujuan</td></tr>
    <tr><td>file <span class="req">*</span></td><td>File sticker (WebP)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "file=@/path/sticker.webp" \
  {BASE}/message/sticker</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/location</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Nomor tujuan</td></tr>
    <tr><td>latitude <span class="req">*</span></td><td>number</td><td>Lintang</td></tr>
    <tr><td>longitude <span class="req">*</span></td><td>number</td><td>Bujur</td></tr>
    <tr><td>name</td><td>string</td><td>Nama lokasi</td></tr>
    <tr><td>address</td><td>string</td><td>Alamat</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","latitude":-6.2,"longitude":106.8,"name":"Jakarta"}' \
  {BASE}/message/location</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/contact</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Nomor tujuan</td></tr>
    <tr><td>vcard <span class="req">*</span></td><td>string</td><td>Data kontak format vCard</td></tr>
    <tr><td>display_name</td><td>string</td><td>Nama tampilan</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","display_name":"Budi","vcard":"BEGIN:VCARD\nVERSION:3.0\nFN:Budi\nTEL:+6281111\nEND:VCARD"}' \
  {BASE}/message/contact</pre></div>
  </div>
</section>

<section id="message-advanced">
  <h2>Pesan Lanjutan</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/poll</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Nomor tujuan</td></tr>
    <tr><td>name <span class="req">*</span></td><td>string</td><td>Pertanyaan poll</td></tr>
    <tr><td>options <span class="req">*</span></td><td>string[]</td><td>Minimal 2 pilihan</td></tr>
    <tr><td>selectable_option_count</td><td>number</td><td>Jumlah jawaban yang boleh dipilih</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","name":"Makan apa?","options":["Nasi","Mie"]}' \
  {BASE}/message/poll</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/poll/vote</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>chat <span class="req">*</span></td><td>string</td><td>JID chat poll</td></tr>
    <tr><td>message_id <span class="req">*</span></td><td>string</td><td>ID pesan poll</td></tr>
    <tr><td>sender</td><td>string</td><td>JID pembuat poll (grup)</td></tr>
    <tr><td>options</td><td>string[]</td><td>Pilihan yang divote</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"chat":"6281234567890@s.whatsapp.net","message_id":"ABC","options":["Nasi"]}' \
  {BASE}/message/poll/vote</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/reaction</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Chat tujuan</td></tr>
    <tr><td>message_id <span class="req">*</span></td><td>string</td><td>ID pesan yang direact</td></tr>
    <tr><td>emoji</td><td>string</td><td>Emoji reaksi (kosong = hapus)</td></tr>
    <tr><td>sender</td><td>string</td><td>JID pengirim (wajib untuk grup)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","message_id":"ABC","emoji":"👍"}' \
  {BASE}/message/reaction</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/revoke</span></div>
    <div class="ep-body"><p>Hapus pesan untuk semua orang.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Chat tujuan</td></tr>
    <tr><td>message_id <span class="req">*</span></td><td>string</td><td>ID pesan</td></tr>
    <tr><td>sender</td><td>string</td><td>JID pengirim (grup)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","message_id":"ABC"}' \
  {BASE}/message/revoke</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/edit</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Chat tujuan</td></tr>
    <tr><td>message_id <span class="req">*</span></td><td>string</td><td>ID pesan</td></tr>
    <tr><td>text <span class="req">*</span></td><td>string</td><td>Teks baru</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","message_id":"ABC","text":"Teks revisi"}' \
  {BASE}/message/edit</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/disappearing</span></div>
    <div class="ep-body"><p>Atur timer pesan menghilang untuk sebuah chat.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Chat tujuan</td></tr>
    <tr><td>seconds</td><td>number</td><td>Durasi (detik). 0 = nonaktif</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","seconds":604800}' \
  {BASE}/message/disappearing</pre></div>
  </div>
</section>

<section id="groups">
  <h2>Grup</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/groups</span></div>
    <div class="ep-body"><p>Daftar grup yang diikuti.</p>
    <pre>curl -H "X-API-Key: KEY" {BASE}/groups</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/groups/info?jid=</span></div>
    <div class="ep-body"><p>Info detail grup.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/groups/info?jid=12036xxx@g.us"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/groups</span></div>
    <div class="ep-body"><p>Buat grup baru.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>name <span class="req">*</span></td><td>string</td><td>Nama grup</td></tr>
    <tr><td>participants</td><td>string[]</td><td>Nomor anggota awal</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tim","participants":["6281111","6282222"]}' \
  {BASE}/groups</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/groups/leave</span></div>
    <div class="ep-body"><pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@g.us"}' \
  {BASE}/groups/leave</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/groups/invite?jid=</span></div>
    <div class="ep-body"><p>Ambil link undangan grup. Tambah <code>&amp;reset=true</code> untuk reset link lama.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/groups/invite?jid=12036xxx@g.us"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/groups/preview?code=</span></div>
    <div class="ep-body"><p>Preview info grup dari kode/link undangan tanpa join.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/groups/preview?code=AbCdEf"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/groups/join</span></div>
    <div class="ep-body"><p>Join grup via kode undangan.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"code":"AbCdEf"}' \
  {BASE}/groups/join</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/groups/participants</span></div>
    <div class="ep-body"><p>Kelola anggota grup.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>group_jid <span class="req">*</span></td><td>string</td><td>JID grup</td></tr>
    <tr><td>action <span class="req">*</span></td><td>string</td><td><code>add</code>, <code>remove</code>, <code>promote</code>, <code>demote</code></td></tr>
    <tr><td>participants</td><td>string[]</td><td>Nomor target</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"group_jid":"12036xxx@g.us","action":"add","participants":["6281111"]}' \
  {BASE}/groups/participants</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/groups/name</span></div>
    <div class="ep-body"><pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@g.us","name":"Nama Baru"}' \
  {BASE}/groups/name</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/groups/description</span></div>
    <div class="ep-body"><pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@g.us","description":"Deskripsi"}' \
  {BASE}/groups/description</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/groups/photo</span><span class="badge">multipart</span></div>
    <div class="ep-body"><pre>curl -X POST -H "X-API-Key: KEY" \
  -F "jid=12036xxx@g.us" \
  -F "file=@/path/foto.jpg" \
  {BASE}/groups/photo</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/groups/locked</span></div>
    <div class="ep-body"><p>Hanya admin yang dapat mengubah info grup.</p>
    <pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@g.us","locked":true}' \
  {BASE}/groups/locked</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/groups/announce</span></div>
    <div class="ep-body"><p>Hanya admin yang dapat mengirim pesan.</p>
    <pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@g.us","announce":true}' \
  {BASE}/groups/announce</pre></div>
  </div>
</section>

<section id="users">
  <h2>User</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/users/check</span></div>
    <div class="ep-body"><p>Cek apakah nomor terdaftar di WhatsApp.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"phones":["6281111","6282222"]}' \
  {BASE}/users/check</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/users/info</span></div>
    <div class="ep-body"><pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jids":["6281111@s.whatsapp.net"]}' \
  {BASE}/users/info</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/users/profile-picture?jid=</span></div>
    <div class="ep-body"><p>URL foto profil. Tambah <code>&amp;preview=true</code> untuk thumbnail.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/users/profile-picture?jid=6281111@s.whatsapp.net"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/users/status</span></div>
    <div class="ep-body"><p>Ubah teks "about" akun sendiri.</p>
    <pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"status":"Sibuk"}' \
  {BASE}/users/status</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/users/business?jid=</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" "{BASE}/users/business?jid=6281111@s.whatsapp.net"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/users/blocklist</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" {BASE}/users/blocklist</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/users/blocklist</span></div>
    <div class="ep-body"><p>Block / unblock kontak.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>jid <span class="req">*</span></td><td>string</td><td>JID kontak</td></tr>
    <tr><td>action</td><td>string</td><td><code>block</code> / <code>unblock</code></td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"6281111@s.whatsapp.net","action":"block"}' \
  {BASE}/users/blocklist</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/users/privacy</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" {BASE}/users/privacy</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/users/devices</span></div>
    <div class="ep-body"><p>Daftar device dari user.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jids":["6281111@s.whatsapp.net"]}' \
  {BASE}/users/devices</pre></div>
  </div>
</section>

<section id="presence">
  <h2>Presence</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/presence</span></div>
    <div class="ep-body"><p>Set status online/offline global.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"state":"available"}' \
  {BASE}/presence</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/presence/typing</span></div>
    <div class="ep-body"><p>Indikator mengetik / merekam.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to <span class="req">*</span></td><td>string</td><td>Chat tujuan</td></tr>
    <tr><td>state</td><td>string</td><td><code>composing</code> / <code>paused</code></td></tr>
    <tr><td>media</td><td>string</td><td><code>audio</code> untuk indikator rekam</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","state":"composing"}' \
  {BASE}/presence/typing</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/presence/subscribe</span></div>
    <div class="ep-body"><p>Subscribe presence (online/last seen) seorang user.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"6281111@s.whatsapp.net"}' \
  {BASE}/presence/subscribe</pre></div>
  </div>
</section>

<section id="chats">
  <h2>Chat</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/chats/read</span></div>
    <div class="ep-body"><p>Tandai pesan sudah dibaca.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>chat <span class="req">*</span></td><td>string</td><td>JID chat</td></tr>
    <tr><td>message_ids <span class="req">*</span></td><td>string[]</td><td>ID pesan</td></tr>
    <tr><td>sender</td><td>string</td><td>JID pengirim (grup)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"chat":"6281234567890@s.whatsapp.net","message_ids":["ABC"]}' \
  {BASE}/chats/read</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/chats/action</span></div>
    <div class="ep-body"><p>Mute / archive / pin chat. Sertakan hanya field yang ingin diubah.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>chat <span class="req">*</span></td><td>string</td><td>JID chat</td></tr>
    <tr><td>mute</td><td>bool</td><td>Mute/unmute</td></tr>
    <tr><td>hours</td><td>number</td><td>Durasi mute (jam)</td></tr>
    <tr><td>archive</td><td>bool</td><td>Arsip/unarsip</td></tr>
    <tr><td>pin</td><td>bool</td><td>Pin/unpin</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"chat":"6281234567890@s.whatsapp.net","pin":true}' \
  {BASE}/chats/action</pre></div>
  </div>
</section>

<section id="media">
  <h2>Media</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/media/download</span></div>
    <div class="ep-body"><p>Download & dekripsi media dari metadata pesan (didapat dari webhook). Response berupa file biner.</p>
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>url <span class="req">*</span></td><td>string</td><td>URL media terenkripsi</td></tr>
    <tr><td>media_key <span class="req">*</span></td><td>string</td><td>Kunci media (base64)</td></tr>
    <tr><td>direct_path</td><td>string</td><td>Direct path</td></tr>
    <tr><td>mimetype</td><td>string</td><td>MIME type</td></tr>
    <tr><td>file_sha256</td><td>string</td><td>SHA256 (base64)</td></tr>
    <tr><td>file_enc_sha256</td><td>string</td><td>Enc SHA256 (base64)</td></tr>
    <tr><td>file_length</td><td>number</td><td>Ukuran file</td></tr>
    <tr><td>media_type</td><td>string</td><td><code>image</code>/<code>video</code>/<code>audio</code>/<code>document</code></td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://...","media_key":"base64...","media_type":"image"}' \
  --output hasil.jpg \
  {BASE}/media/download</pre></div>
  </div>
</section>

<section id="newsletter">
  <h2>Newsletter / Channel</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/newsletters</span></div>
    <div class="ep-body"><p>Daftar channel yang diikuti.</p>
    <pre>curl -H "X-API-Key: KEY" {BASE}/newsletters</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/newsletters/info?jid=</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" "{BASE}/newsletters/info?jid=12036xxx@newsletter"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/newsletters/invite?invite=</span></div>
    <div class="ep-body"><p>Info channel dari kode/link invite.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/newsletters/invite?invite=AbCdEf"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/newsletters/follow</span></div>
    <div class="ep-body"><pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@newsletter"}' \
  {BASE}/newsletters/follow</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/newsletters/unfollow</span></div>
    <div class="ep-body"><pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@newsletter"}' \
  {BASE}/newsletters/unfollow</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/newsletters</span><span class="badge">multipart</span></div>
    <div class="ep-body"><p>Buat channel baru.</p>
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>name <span class="req">*</span></td><td>Nama channel</td></tr>
    <tr><td>description</td><td>Deskripsi</td></tr>
    <tr><td>file</td><td>Foto channel (opsional)</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "name=Channel Saya" \
  -F "description=Update produk" \
  {BASE}/newsletters</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/newsletters/messages?jid=</span></div>
    <div class="ep-body"><p>Ambil pesan terbaru channel.</p>
    <pre>curl -H "X-API-Key: KEY" "{BASE}/newsletters/messages?jid=12036xxx@newsletter"</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/newsletters/mute</span></div>
    <div class="ep-body"><pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"jid":"12036xxx@newsletter","mute":true}' \
  {BASE}/newsletters/mute</pre></div>
  </div>
</section>

<section id="webhook">
  <h2>Webhook</h2>
  <p>Atur URL untuk menerima notifikasi pesan masuk.</p>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/webhook</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" {BASE}/webhook</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/webhook</span></div>
    <div class="ep-body">
    <pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://your-server.com/webhook"}' \
  {BASE}/webhook</pre>
    <h3>Payload webhook</h3>
    <pre>{
  "event": "message.received",
  "timestamp": "{YEAR}-06-29T10:00:00Z",
  "data": {
    "message_id": "ABC123",
    "from": "6281234567890@s.whatsapp.net",
    "chat": "6281234567890@s.whatsapp.net",
    "text": "Halo!",
    "push_name": "John",
    "is_group": false
  }
}</pre></div>
  </div>
</section>

<section id="errors">
  <h2>Response &amp; Error</h2>
  <p>Semua response JSON dengan format:</p>
  <pre>{
  "success": true,
  "data": { ... }
}

{
  "success": false,
  "message": "deskripsi error"
}</pre>
  <table>
    <tr><th>HTTP</th><th>Keterangan</th></tr>
    <tr><td>200</td><td>Berhasil</td></tr>
    <tr><td>400</td><td>Request tidak valid</td></tr>
    <tr><td>401</td><td>API key salah / tidak ada</td></tr>
    <tr><td>500</td><td>Error server / WhatsApp</td></tr>
  </table>
</section>

<section id="fullapi">
  <h2>Daftar Lengkap (versi mesin)</h2>
  <p>Daftar semua endpoint dalam format JSON (berguna untuk generate client otomatis):</p>
  <pre>curl -H "X-API-Key: KEY" {BASE}/api/endpoints</pre>
</section>

</main>
</div>
<footer>WhatsMeow Gateway &copy; {YEAR} — <a href="/health">Health Check</a></footer>
</body>
</html>`
