package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
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

func renderDocsPage(baseURL string) string {
	year := time.Now().Year()
	return fmt.Sprintf(`<!DOCTYPE html>
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
  table{width:100%%;border-collapse:collapse;font-size:.875rem;margin:.75rem 0}
  th,td{text-align:left;padding:.5rem .75rem;border-bottom:1px solid var(--border)}
  th{color:var(--muted);font-weight:500}
  .badge{display:inline-block;background:var(--accent-dim);color:#bbf7d0;font-size:.75rem;padding:.15rem .5rem;border-radius:4px}
  footer{text-align:center;padding:2rem;color:var(--muted);font-size:.8rem;border-top:1px solid var(--border)}
  @media(max-width:768px){.layout{flex-direction:column}nav{width:100%%;height:auto;position:static;border-right:none;border-bottom:1px solid var(--border)}}
</style>
</head>
<body>
<header>
  <div>
    <h1>WhatsMeow Gateway API</h1>
    <div class="base">%s</div>
  </div>
  <form method="POST" action="/docs/logout" style="margin:0"><button type="submit" class="logout" style="background:none;border:none;cursor:pointer;color:var(--muted);font-size:.875rem">Keluar</button></form>
</header>
<div class="layout">
<nav>
  <a href="#auth">Autentikasi</a>
  <a href="#session">Session</a>
  <a href="#message">Kirim Pesan</a>
  <a href="#webhook">Webhook</a>
  <a href="#errors">Error</a>
</nav>
<main>

<section id="auth">
  <h2>Autentikasi</h2>
  <p>Semua endpoint API (kecuali <code>/health</code> dan <code>/docs</code>) memerlukan secret key via header:</p>
  <pre>X-API-Key: YOUR_SECRET_KEY</pre>
  <p>Alternatif:</p>
  <pre>Authorization: Bearer YOUR_SECRET_KEY</pre>
</section>

<section id="session">
  <h2>Session</h2>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/session/status</span></div>
    <div class="ep-body"><p>Cek status koneksi dan login WhatsApp.</p>
    <pre>curl -H "X-API-Key: KEY" %s/session/status</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/connect</span></div>
    <div class="ep-body"><p>Mulai koneksi ke WhatsApp. Jika belum login, QR code akan di-generate.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" %s/session/connect</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/session/qr</span></div>
    <div class="ep-body"><p>Ambil QR code untuk scan login. Response berisi <code>code</code> dan <code>qr_base64</code> (PNG).</p>
    <pre>curl -H "X-API-Key: KEY" %s/session/qr</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/pair</span></div>
    <div class="ep-body"><p>Login via pairing code (tanpa scan QR).</p>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"phone":"6281234567890"}' \
  %s/session/pair</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/logout</span></div>
    <div class="ep-body"><p>Logout dan hapus session WhatsApp.</p>
    <pre>curl -X POST -H "X-API-Key: KEY" %s/session/logout</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/session/disconnect</span></div>
    <div class="ep-body"><p>Putuskan koneksi sementara (session tetap tersimpan).</p>
    <pre>curl -X POST -H "X-API-Key: KEY" %s/session/disconnect</pre></div>
  </div>
</section>

<section id="message">
  <h2>Kirim Pesan</h2>
  <p>Format nomor tujuan: <code>6281234567890</code> (kode negara, tanpa <code>+</code>) atau JID penuh <code>628...@s.whatsapp.net</code></p>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/text</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Tipe</th><th>Keterangan</th></tr>
    <tr><td>to</td><td>string</td><td>Nomor tujuan</td></tr>
    <tr><td>text</td><td>string</td><td>Isi pesan</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"6281234567890","text":"Halo!"}' \
  %s/message/text</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/image</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to</td><td>Nomor tujuan (wajib)</td></tr>
    <tr><td>file</td><td>File gambar (wajib)</td></tr>
    <tr><td>caption</td><td>Caption opsional</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "caption=Caption" \
  -F "file=@/path/gambar.jpg" \
  %s/message/image</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method post">POST</span><span class="path">/message/document</span><span class="badge">multipart</span></div>
    <div class="ep-body">
    <table><tr><th>Field</th><th>Keterangan</th></tr>
    <tr><td>to</td><td>Nomor tujuan (wajib)</td></tr>
    <tr><td>file</td><td>File dokumen (wajib)</td></tr>
    <tr><td>filename</td><td>Nama file opsional</td></tr>
    <tr><td>mimetype</td><td>MIME type opsional</td></tr></table>
    <pre>curl -X POST -H "X-API-Key: KEY" \
  -F "to=6281234567890" \
  -F "filename=laporan.pdf" \
  -F "file=@/path/laporan.pdf" \
  %s/message/document</pre></div>
  </div>
</section>

<section id="webhook">
  <h2>Webhook</h2>
  <p>Atur URL untuk menerima notifikasi pesan masuk.</p>

  <div class="endpoint">
    <div class="ep-head"><span class="method get">GET</span><span class="path">/webhook</span></div>
    <div class="ep-body"><pre>curl -H "X-API-Key: KEY" %s/webhook</pre></div>
  </div>

  <div class="endpoint">
    <div class="ep-head"><span class="method put">PUT</span><span class="path">/webhook</span></div>
    <div class="ep-body">
    <pre>curl -X PUT -H "X-API-Key: KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://your-server.com/webhook"}' \
  %s/webhook</pre>
    <h3>Payload webhook</h3>
    <pre>{
  "event": "message.received",
  "timestamp": "2026-06-29T10:00:00Z",
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

</main>
</div>
<footer>WhatsMeow Gateway &copy; %d — <a href="/health">Health Check</a></footer>
</body>
</html>`,
		baseURL,
		baseURL, baseURL, baseURL, baseURL, baseURL, baseURL,
		baseURL, baseURL, baseURL, baseURL,
		year,
	)
}
