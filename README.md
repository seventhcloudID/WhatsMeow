# WhatsApp Gateway (WhatsMeow)

REST API server / WhatsApp gateway berbasis [whatsmeow](https://github.com/tulir/whatsmeow) untuk mengirim dan menerima pesan WhatsApp.

## Fitur

- **Multi-session / multi-tenant**: satu gateway melayani banyak nomor WhatsApp (pilih via header `X-Session-ID`)
- Login via **QR Code** atau **Pairing Code** (nomor telepon)
- Kirim pesan **teks**, **gambar**, dan **dokumen**
- Webhook untuk pesan masuk (per-session)
- Session persisten (SQLite, satu file per tenant)
- Autentikasi API Key

## Persyaratan

- Go 1.21+ (tidak perlu GCC — menggunakan SQLite pure Go)

## Instalasi

```bash
# Clone / masuk ke folder proyek
cd WhatsMeow

# Salin konfigurasi
copy .env.example .env

# Install dependensi
go mod tidy

# Jalankan server
go run .
```

Server berjalan di `http://localhost:8081` secara default.

> **Catatan:** Port `8080` sering sudah dipakai aplikasi lain di Windows. Jika perlu, ubah `PORT` di `.env`.

## Konfigurasi (.env)

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `PORT` | `8081` | Port HTTP server |
| `API_KEY` | `changeme` | API key untuk autentikasi |
| `DB_PATH` | `data/session.db` | Basis path DB; folder-nya dipakai untuk `data/sessions/<id>.db` per tenant |
| `WEBHOOK_URL` | _(kosong)_ | URL webhook pesan masuk untuk session `default` |
| `LOG_LEVEL` | `INFO` | Level log |

## API Lengkap

Gateway expose **50+ endpoint** REST yang wrap fitur whatsmeow. Daftar lengkap:

```bash
curl -H "X-API-Key: KEY" https://meow.solusijasa.com/api/endpoints
```

### Kategori endpoint

| Kategori | Endpoint |
|----------|----------|
| **Sessions** | `/sessions` (list/create), `/sessions/{id}` (delete) — header `X-Session-ID` untuk pilih tenant |
| **Session** | `/session/status`, `/connect`, `/qr`, `/pair`, `/reset`, `/logout` |
| **Pesan** | `/message/text`, `/image`, `/video`, `/audio`, `/document`, `/sticker`, `/location`, `/contact`, `/poll`, `/reaction`, `/revoke`, `/edit` |
| **Grup** | `/groups`, `/groups/info`, `/groups/join`, `/groups/participants`, dll. |
| **User** | `/users/check`, `/users/info`, `/users/profile-picture`, `/users/blocklist` |
| **Presence** | `/presence`, `/presence/typing`, `/presence/subscribe` |
| **Chat** | `/chats/read`, `/chats/action` (mute/archive/pin) |
| **Media** | `/media/download` |
| **Newsletter** | `/newsletters`, `/newsletters/follow`, `/newsletters/messages` |
| **Webhook** | `/webhook` |

## Dokumentasi API

Halaman dokumentasi interaktif tersedia di:

```
https://meow.solusijasa.com/docs
```

Halaman ini **dilindungi secret key**. Masukkan API key saat login, atau akses langsung:

```
https://meow.solusijasa.com/docs?key=YOUR_API_KEY
```

Session login docs berlaku 24 jam via cookie.

## Autentikasi

Semua endpoint (kecuali `/health`) memerlukan API key via header:

```
X-API-Key: your-secret-api-key
```

atau:

```
Authorization: Bearer your-secret-api-key
```

## Multi-Session (Multi-Tenant)

Satu gateway bisa memegang banyak nomor WhatsApp sekaligus. Pilih tenant dengan header **opsional**:

```
X-Session-ID: nama-tenant
```

Tanpa header ini dipakai session `default` (kompatibel dengan pemakaian lama). Semua endpoint `/session/*`, `/message/*`, `/groups/*`, dst berlaku untuk session pada header tersebut. Webhook diset **per-session**, dan payload webhook menyertakan `session_id`.

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/sessions` | Daftar semua session + status |
| POST | `/sessions` | Buat session baru `{"id":"tenant-1","label":"Toko A"}` |
| DELETE | `/sessions/{id}` | Hapus session (logout + hapus DB-nya) |

```bash
# Buat session baru
curl -X POST -H "X-API-Key: changeme" \
  -H "Content-Type: application/json" \
  -d '{"id":"tenant-1","label":"Toko A"}' \
  http://localhost:8081/sessions

# Cek status nomor untuk tenant-1
curl -H "X-API-Key: changeme" -H "X-Session-ID: tenant-1" \
  http://localhost:8081/session/status
```

## API Endpoints

### Health Check

```bash
curl http://localhost:8081/health
```

### Session

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/session/status` | Status koneksi & login |
| POST | `/session/connect` | Mulai koneksi / QR login |
| GET | `/session/qr` | Ambil QR code (base64 PNG) |
| POST | `/session/pair` | Login via pairing code |
| POST | `/session/logout` | Logout & hapus session |
| POST | `/session/disconnect` | Putuskan koneksi |

**Status session:**

```bash
curl -H "X-API-Key: changeme" http://localhost:8081/session/status
```

**Login QR:**

```bash
# 1. Connect
curl -X POST -H "X-API-Key: changeme" http://localhost:8081/session/connect

# 2. Ambil QR (scan dengan WhatsApp di HP)
curl -H "X-API-Key: changeme" http://localhost:8081/session/qr
```

**Login Pairing Code:**

```bash
curl -X POST -H "X-API-Key: changeme" \
  -H "Content-Type: application/json" \
  -d '{"phone": "6281234567890"}' \
  http://localhost:8081/session/pair
```

### Kirim Pesan

**Teks:**

```bash
curl -X POST -H "X-API-Key: changeme" \
  -H "Content-Type: application/json" \
  -d '{"to": "6281234567890", "text": "Halo dari API!"}' \
  http://localhost:8081/message/text
```

**Gambar:**

```bash
curl -X POST -H "X-API-Key: changeme" \
  -F "to=6281234567890" \
  -F "caption=Caption gambar" \
  -F "file=@/path/to/image.jpg" \
  http://localhost:8081/message/image
```

**Dokumen:**

```bash
curl -X POST -H "X-API-Key: changeme" \
  -F "to=6281234567890" \
  -F "filename=laporan.pdf" \
  -F "file=@/path/to/laporan.pdf" \
  http://localhost:8081/message/document
```

Format nomor tujuan: `6281234567890` (kode negara tanpa `+`) atau JID penuh `6281234567890@s.whatsapp.net`.

### Webhook

Atur URL webhook untuk menerima pesan masuk:

```bash
curl -X PUT -H "X-API-Key: changeme" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://your-server.com/webhook"}' \
  http://localhost:8081/webhook
```

Payload webhook:

```json
{
  "event": "message.received",
  "session_id": "default",
  "timestamp": "2026-06-28T10:00:00Z",
  "data": {
    "message_id": "ABC123",
    "from": "6281234567890@s.whatsapp.net",
    "chat": "6281234567890@s.whatsapp.net",
    "text": "Halo!",
    "push_name": "John",
    "is_group": false
  }
}
```

## Build Binary

```bash
go build -o gateway.exe .
```

## Struktur Proyek

```
WhatsMeow/
├── main.go                 # Entry point
├── internal/
│   ├── config/             # Konfigurasi environment
│   ├── wa/                 # WhatsApp client (whatsmeow)
│   └── api/                # REST API handlers & router
├── data/
│   ├── sessions.json       # Metadata tiap session (id, label, webhook)
│   └── sessions/           # DB SQLite per tenant: <id>.db (auto-created)
├── .env.example
└── README.md
```

## Catatan

- WhatsApp Web multi-device API tidak resmi — gunakan dengan risiko sendiri.
- Jangan commit file `.env` atau folder `data/` (berisi session & `sessions.json`).
- Ganti `API_KEY` default sebelum deploy ke production.
- Jalankan **satu** proses gateway saja; tiap tenant sudah punya DB terpisah sehingga tidak saling mengunci.

## Lisensi

Proyek ini menggunakan library [whatsmeow](https://github.com/tulir/whatsmeow) (MPL-2.0).
