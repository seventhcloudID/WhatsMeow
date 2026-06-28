# WhatsApp Gateway (WhatsMeow)

REST API server / WhatsApp gateway berbasis [whatsmeow](https://github.com/tulir/whatsmeow) untuk mengirim dan menerima pesan WhatsApp.

## Fitur

- Login via **QR Code** atau **Pairing Code** (nomor telepon)
- Kirim pesan **teks**, **gambar**, dan **dokumen**
- Webhook untuk pesan masuk
- Session persisten (SQLite)
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
| `DB_PATH` | `data/session.db` | Path database session |
| `WEBHOOK_URL` | _(kosong)_ | URL webhook pesan masuk |
| `LOG_LEVEL` | `INFO` | Level log |

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
├── data/                   # Session database (auto-created)
├── .env.example
└── README.md
```

## Catatan

- WhatsApp Web multi-device API tidak resmi — gunakan dengan risiko sendiri.
- Jangan commit file `.env` atau `data/session.db`.
- Ganti `API_KEY` default sebelum deploy ke production.

## Lisensi

Proyek ini menggunakan library [whatsmeow](https://github.com/tulir/whatsmeow) (MPL-2.0).
