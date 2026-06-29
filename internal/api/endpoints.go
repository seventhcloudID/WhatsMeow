package api

func apiEndpoints() []map[string]string {
	return []map[string]string{
		// Session
		{"method": "GET", "path": "/health", "desc": "Health check"},
		// Sessions (multi-tenant). Pilih tenant via header X-Session-ID di semua endpoint lain.
		{"method": "GET", "path": "/sessions", "desc": "List semua session"},
		{"method": "POST", "path": "/sessions", "desc": "Buat session {id,label}"},
		{"method": "DELETE", "path": "/sessions/{id}", "desc": "Hapus session"},
		{"method": "GET", "path": "/session/status", "desc": "Status session (header X-Session-ID)"},
		{"method": "POST", "path": "/session/connect", "desc": "Connect WhatsApp"},
		{"method": "GET", "path": "/session/qr", "desc": "QR code login"},
		{"method": "POST", "path": "/session/pair", "desc": "Pairing code login"},
		{"method": "POST", "path": "/session/reset", "desc": "Reset session"},
		{"method": "POST", "path": "/session/logout", "desc": "Logout"},
		{"method": "POST", "path": "/session/disconnect", "desc": "Disconnect"},
		// Messages
		{"method": "POST", "path": "/message/text", "desc": "Kirim teks"},
		{"method": "POST", "path": "/message/image", "desc": "Kirim gambar"},
		{"method": "POST", "path": "/message/video", "desc": "Kirim video"},
		{"method": "POST", "path": "/message/audio", "desc": "Kirim audio/voice"},
		{"method": "POST", "path": "/message/document", "desc": "Kirim dokumen"},
		{"method": "POST", "path": "/message/sticker", "desc": "Kirim sticker"},
		{"method": "POST", "path": "/message/location", "desc": "Kirim lokasi"},
		{"method": "POST", "path": "/message/contact", "desc": "Kirim kontak"},
		{"method": "POST", "path": "/message/poll", "desc": "Buat poll"},
		{"method": "POST", "path": "/message/poll/vote", "desc": "Vote poll"},
		{"method": "POST", "path": "/message/reaction", "desc": "Reaction emoji"},
		{"method": "POST", "path": "/message/revoke", "desc": "Hapus pesan"},
		{"method": "POST", "path": "/message/edit", "desc": "Edit pesan"},
		{"method": "POST", "path": "/message/disappearing", "desc": "Disappearing timer"},
		// Groups
		{"method": "GET", "path": "/groups", "desc": "List grup joined"},
		{"method": "GET", "path": "/groups/info?jid=", "desc": "Info grup"},
		{"method": "POST", "path": "/groups", "desc": "Buat grup"},
		{"method": "POST", "path": "/groups/leave", "desc": "Keluar grup"},
		{"method": "GET", "path": "/groups/invite?jid=", "desc": "Invite link"},
		{"method": "GET", "path": "/groups/preview?code=", "desc": "Preview grup dari link"},
		{"method": "POST", "path": "/groups/join", "desc": "Join via link"},
		{"method": "POST", "path": "/groups/participants", "desc": "Add/remove/promote/demote"},
		{"method": "PUT", "path": "/groups/name", "desc": "Ubah nama grup"},
		{"method": "PUT", "path": "/groups/description", "desc": "Ubah deskripsi"},
		{"method": "POST", "path": "/groups/photo", "desc": "Ubah foto grup"},
		{"method": "PUT", "path": "/groups/locked", "desc": "Lock grup"},
		{"method": "PUT", "path": "/groups/announce", "desc": "Announce only"},
		// Users
		{"method": "POST", "path": "/users/check", "desc": "Cek nomor WA"},
		{"method": "POST", "path": "/users/info", "desc": "Info user"},
		{"method": "GET", "path": "/users/profile-picture?jid=", "desc": "Foto profil"},
		{"method": "PUT", "path": "/users/status", "desc": "Set about status"},
		{"method": "GET", "path": "/users/business?jid=", "desc": "Profil bisnis"},
		{"method": "GET", "path": "/users/blocklist", "desc": "Daftar block"},
		{"method": "POST", "path": "/users/blocklist", "desc": "Block/unblock"},
		{"method": "GET", "path": "/users/privacy", "desc": "Privacy settings"},
		{"method": "POST", "path": "/users/devices", "desc": "Device list user"},
		// Presence
		{"method": "POST", "path": "/presence", "desc": "Online/offline global"},
		{"method": "POST", "path": "/presence/typing", "desc": "Typing indicator"},
		{"method": "POST", "path": "/presence/subscribe", "desc": "Subscribe presence"},
		// Chats
		{"method": "POST", "path": "/chats/read", "desc": "Mark as read"},
		{"method": "POST", "path": "/chats/action", "desc": "Mute/archive/pin"},
		// Media
		{"method": "POST", "path": "/media/download", "desc": "Download media"},
		// Newsletter
		{"method": "GET", "path": "/newsletters", "desc": "List channel"},
		{"method": "GET", "path": "/newsletters/info?jid=", "desc": "Info channel"},
		{"method": "GET", "path": "/newsletters/invite?invite=", "desc": "Info dari invite"},
		{"method": "POST", "path": "/newsletters/follow", "desc": "Follow channel"},
		{"method": "POST", "path": "/newsletters/unfollow", "desc": "Unfollow channel"},
		{"method": "POST", "path": "/newsletters", "desc": "Buat channel"},
		{"method": "GET", "path": "/newsletters/messages?jid=", "desc": "Pesan channel"},
		{"method": "PUT", "path": "/newsletters/mute", "desc": "Mute channel"},
		// Webhook
		{"method": "GET", "path": "/webhook", "desc": "Get webhook URL"},
		{"method": "PUT", "path": "/webhook", "desc": "Set webhook URL"},
		{"method": "GET", "path": "/api/endpoints", "desc": "Daftar semua endpoint"},
	}
}
