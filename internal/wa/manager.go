package wa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"github.com/skip2/go-qrcode"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type SessionStatus struct {
	Connected  bool   `json:"connected"`
	LoggedIn   bool   `json:"logged_in"`
	JID        string `json:"jid,omitempty"`
	Phone      string `json:"phone,omitempty"`
	PushName   string `json:"push_name,omitempty"`
}

type QRResponse struct {
	Code      string `json:"code"`
	QRBase64  string `json:"qr_base64,omitempty"`
	Timeout   int    `json:"timeout_seconds"`
	Event     string `json:"event,omitempty"`
}

type SendResult struct {
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

type Manager struct {
	mu         sync.RWMutex
	id         string
	client     *whatsmeow.Client
	container  *sqlstore.Container
	dbPath     string
	logLevel   string
	webhookURL string
	log        waLog.Logger

	qrCode    string
	qrTimeout time.Duration
	qrEvent   string
}

// sqliteDSN menambahkan pragma penting agar SQLite tidak kena
// "database is locked" (SQLITE_BUSY) saat whatsmeow menulis banyak data
// sekaligus (identity, prekey, app state). busy_timeout = tunggu lock,
// journal_mode WAL = pembaca tidak memblokir penulis.
func sqliteDSN(path string) string {
	return path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
}

func NewManager(id, dbPath, logLevel string) (*Manager, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbLog := waLog.Stdout("Database["+id+"]", logLevel, true)
	ctx := context.Background()

	container, err := sqlstore.New(ctx, "sqlite", sqliteDSN(dbPath), dbLog)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	clientLog := waLog.Stdout("Client["+id+"]", logLevel, true)
	client := whatsmeow.NewClient(device, clientLog)

	m := &Manager{
		id:        id,
		client:    client,
		container: container,
		dbPath:    dbPath,
		logLevel:  logLevel,
		log:       clientLog,
	}

	client.AddEventHandler(m.eventHandler)
	return m, nil
}

// ID mengembalikan id session tenant ini.
func (m *Manager) ID() string { return m.id }

// Close memutus, logout (bila masih login), dan menutup DB. Dipakai saat
// session dihapus.
func (m *Manager) Close(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		if m.client.Store.ID != nil {
			_ = m.client.Logout(ctx)
		}
		if m.client.IsConnected() {
			m.client.Disconnect()
		}
	}
	if m.container != nil {
		_ = m.container.Close()
	}
}

func (m *Manager) SetWebhookURL(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookURL = url
}

func (m *Manager) GetWebhookURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.webhookURL
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startQRLocked(ctx)
}

func (m *Manager) startQRLocked(ctx context.Context) error {
	if m.client.Store.ID != nil {
		if m.client.IsConnected() {
			return nil
		}
		return m.client.Connect()
	}

	if m.client.IsConnected() {
		return nil
	}

	// PENTING: masa hidup login QR TIDAK boleh terikat context request HTTP.
	// Kalau pakai ctx request, begitu request /session/connect selesai context
	// dibatalkan -> whatsmeow membatalkan login & memutus koneksi (~2 detik),
	// sehingga QR yang discan tak pernah tuntas. Pakai background context.
	qrChan, err := m.client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("get qr channel: %w", err)
	}

	if err := m.client.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	go m.watchQR(qrChan)
	return nil
}

// resetLocked memutus koneksi, menutup & MENGHAPUS file DB session, lalu
// membuat client baru dari device kosong. Dipakai bersama oleh reset, logout,
// dan saat device dilepas dari HP, agar state benar-benar bersih dan client
// baru siap untuk penautan ulang. Pemanggil WAJIB memegang m.mu.
func (m *Manager) resetLocked() error {
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect()
	}
	if m.container != nil {
		if err := m.container.Close(); err != nil {
			m.log.Warnf("close db: %v", err)
		}
	}
	removeSQLiteFiles(m.dbPath)
	return m.reinitClient(context.Background())
}

// ResetSession hapus session lama dan buat QR baru (fix pairing gagal).
func (m *Manager) ResetSession(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.resetLocked(); err != nil {
		return fmt.Errorf("reinit client: %w", err)
	}

	return m.startQRLocked(ctx)
}

func (m *Manager) reinitClient(ctx context.Context) error {
	dbLog := waLog.Stdout("Database["+m.id+"]", m.logLevel, true)
	container, err := sqlstore.New(ctx, "sqlite", sqliteDSN(m.dbPath), dbLog)
	if err != nil {
		return err
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}

	clientLog := waLog.Stdout("Client["+m.id+"]", m.logLevel, true)
	client := whatsmeow.NewClient(device, clientLog)
	client.AddEventHandler(m.eventHandler)

	m.container = container
	m.client = client
	m.log = clientLog
	m.qrCode = ""
	m.qrEvent = ""
	m.qrTimeout = 0
	return nil
}

func removeSQLiteFiles(path string) {
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		_ = os.Remove(p)
	}
}

func (m *Manager) watchQR(qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		m.mu.Lock()
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			m.qrCode = item.Code
			m.qrTimeout = item.Timeout
			m.qrEvent = "code"
			m.log.Infof("QR code updated, expires in %s", item.Timeout)
		case "success":
			m.qrCode = ""
			m.qrEvent = "success"
			m.log.Infof("Pairing successful")
		default:
			m.qrEvent = item.Event
			if item.Error != nil {
				m.log.Errorf("QR error: %v", item.Error)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Manager) Status() SessionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := SessionStatus{
		Connected: m.client.IsConnected(),
		LoggedIn:  m.client.Store.ID != nil,
	}

	if m.client.Store.ID != nil {
		status.JID = m.client.Store.ID.String()
		status.Phone = m.client.Store.ID.User
	}
	if m.client.Store.PushName != "" {
		status.PushName = m.client.Store.PushName
	}

	return status
}

func (m *Manager) GetQR() (*QRResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client.Store.ID != nil {
		return nil, fmt.Errorf("sudah login, tidak perlu QR")
	}

	if m.qrCode == "" {
		msg := m.qrEvent
		if msg == "" {
			msg = "waiting"
		}
		return &QRResponse{
			Event: msg,
		}, nil
	}

	png, err := qrcode.Encode(m.qrCode, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generate qr image: %w", err)
	}

	return &QRResponse{
		Code:     m.qrCode,
		QRBase64: fmt.Sprintf("data:image/png;base64,%s", encodeBase64(png)),
		Timeout:  int(m.qrTimeout.Seconds()),
		Event:    "code",
	}, nil
}

func (m *Manager) PairPhone(ctx context.Context, phone string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client.Store.ID != nil {
		return "", fmt.Errorf("sudah login")
	}

	if !m.client.IsConnected() {
		// Background context: koneksi pairing harus bertahan setelah request selesai.
		qrChan, err := m.client.GetQRChannel(context.Background())
		if err != nil {
			return "", fmt.Errorf("get qr channel: %w (coba POST /session/reset dulu)", err)
		}
		go m.watchQR(qrChan)
		if err := m.client.Connect(); err != nil {
			return "", fmt.Errorf("connect: %w", err)
		}
	}

	time.Sleep(3 * time.Second)

	code, err := m.client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("pair phone: %w", err)
	}

	return code, nil
}

func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client.IsConnected() {
		return nil
	}
	return m.client.Connect()
}

func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.client.Disconnect()
}

func (m *Manager) Logout(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client.Store.ID == nil {
		return fmt.Errorf("belum login")
	}

	// Beritahu WhatsApp untuk melepas device (best-effort). client.Logout TIDAK
	// memancarkan event LoggedOut, jadi pembersihan state lokal harus dilakukan
	// manual di bawah.
	if err := m.client.Logout(ctx); err != nil {
		m.log.Warnf("logout ke WhatsApp gagal (tetap bersihkan state lokal): %v", err)
	}

	// Hapus DB session & buat client baru agar siap tautkan ulang (tanpa ini,
	// store ditandai Deleted dan /session/connect berikutnya gagal 500).
	if err := m.resetLocked(); err != nil {
		return fmt.Errorf("bersihkan session: %w", err)
	}
	return nil
}

func (m *Manager) autoReconnect() {
	m.mu.RLock()
	loggedIn := m.client.Store.ID != nil
	m.mu.RUnlock()

	if !loggedIn {
		return
	}

	time.Sleep(3 * time.Second)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client.Store.ID == nil || m.client.IsConnected() {
		return
	}

	m.log.Infof("Mencoba reconnect otomatis...")
	if err := m.client.Connect(); err != nil {
		m.log.Errorf("Reconnect gagal: %v", err)
	}
}

// handleLoggedOut dipanggil saat WhatsApp melepas device (mis. "logged out from
// another device"). Tanpa ini, client lama nyangkut dan /session/connect gagal
// (500) karena GetQRChannel menolak. Di sini kita bersihkan & buat client baru
// dari device kosong, sehingga connect berikutnya bisa memunculkan QR baru.
func (m *Manager) handleLoggedOut() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.resetLocked(); err != nil {
		m.log.Errorf("reset setelah logout: %v", err)
	}
}

func (m *Manager) eventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		m.handleIncomingMessage(evt)
	case *events.Connected:
		m.log.Infof("Terhubung ke WhatsApp")
	case *events.Disconnected:
		m.log.Warnf("Terputus dari WhatsApp")
		go m.autoReconnect()
	case *events.LoggedOut:
		m.log.Warnf("Logout: %s (membersihkan & reinit client agar bisa tautkan ulang)", evt.Reason)
		go m.handleLoggedOut()
	case *events.PairSuccess:
		m.log.Infof("Pairing berhasil: %s", evt.ID)
	}
}

func (m *Manager) handleIncomingMessage(evt *events.Message) {
	// Gateway tertaut sebagai linked device, sehingga menerima SEMUA pesan yang
	// diterima akun: DM pribadi, grup, status/story (broadcast), dan channel
	// (newsletter). Untuk inbox CRM kita hanya teruskan DM pribadi dari orang lain.
	if evt.Info.IsFromMe {
		return
	}
	switch evt.Info.Chat.Server {
	case types.DefaultUserServer, types.HiddenUserServer:
		// DM pribadi (nomor "@s.whatsapp.net" atau "@lid") — diteruskan.
	default:
		// Grup (g.us), status/broadcast, newsletter, hosted, dll — diabaikan.
		return
	}

	text := extractMessageText(evt)
	sender := m.resolveSenderIdentity(evt)
	payload := map[string]interface{}{
		"message_id":   evt.Info.ID,
		"from":         evt.Info.Sender.String(),
		"from_phone":   sender.phone,
		"chat":         evt.Info.Chat.String(),
		"timestamp":    evt.Info.Timestamp,
		"push_name":    evt.Info.PushName,
		"contact_name": sender.name,
		"is_group":     false,
		"is_from_me":   false,
		"media_type":   evt.Info.MediaType,
		"text":         text,
	}
	if evt.Message.GetImageMessage() != nil {
		payload["media_type"] = "image"
	} else if evt.Message.GetVideoMessage() != nil {
		payload["media_type"] = "video"
	} else if evt.Message.GetAudioMessage() != nil {
		payload["media_type"] = "audio"
	} else if evt.Message.GetDocumentMessage() != nil {
		payload["media_type"] = "document"
	} else if evt.Message.GetStickerMessage() != nil {
		payload["media_type"] = "sticker"
	} else if evt.Message.GetLocationMessage() != nil {
		payload["media_type"] = "location"
	} else if evt.Message.GetContactMessage() != nil {
		payload["media_type"] = "contact"
	}

	logName := sender.name
	logFrom := evt.Info.Sender.String()
	if sender.phone != "" {
		logFrom = sender.phone
	}
	if logName != "" {
		m.log.Infof("Pesan masuk dari %s (%s): %s", logName, logFrom, text)
	} else {
		m.log.Infof("Pesan masuk dari %s: %s", logFrom, text)
	}
	m.sendWebhook("message.received", payload)
}

// senderIdentity berisi identitas pengirim yang sudah di-resolve dari berbagai
// sumber di session store (peta LID->PN dan buku kontak WA).
type senderIdentity struct {
	phone string // nomor telepon digit (mis. "628123456789"); kosong bila "ID privasi"
	name  string // nama tampilan terbaik (buku kontak WA, fallback push name)
}

// resolveSenderIdentity meniru cara wafin-chatbot melengkapi data kontak:
//   - Nomor: dari alamat PN langsung, SenderAlt, atau peta LID->PN whatsmeow
//     (tabel whatsmeow_lid_map). Kosong hanya bila kontak benar-benar "ID privasi".
//   - Nama: dari buku kontak WA (tabel whatsmeow_contacts: full/first/business name),
//     dengan cross-reference JID LID maupun PN, fallback ke push name pesan.
func (m *Manager) resolveSenderIdentity(evt *events.Message) senderIdentity {
	from := evt.Info.Sender

	// Tentukan JID nomor (PN) bila memungkinkan.
	var pnJID types.JID
	if from.Server == types.DefaultUserServer {
		pnJID = from
	} else if alt := evt.Info.SenderAlt; alt.Server == types.DefaultUserServer {
		pnJID = alt
	}

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Resolve PN lewat peta LID->PN bila belum dapat & pengirim memakai LID.
	if pnJID.IsEmpty() && from.Server == types.HiddenUserServer &&
		client != nil && client.Store != nil && client.Store.LIDs != nil {
		if pn, err := client.Store.LIDs.GetPNForLID(ctx, from); err == nil && pn.Server == types.DefaultUserServer {
			pnJID = pn
		}
	}

	id := senderIdentity{}
	if !pnJID.IsEmpty() {
		id.phone = pnJID.User
	}

	// Nama dari buku kontak WA: coba JID pengirim (LID) lalu JID nomor (PN).
	if client != nil && client.Store != nil && client.Store.Contacts != nil {
		for _, jid := range []types.JID{from, pnJID} {
			if jid.IsEmpty() {
				continue
			}
			info, err := client.Store.Contacts.GetContact(ctx, jid)
			if err != nil || !info.Found {
				continue
			}
			if name := firstNonEmpty(info.FullName, info.FirstName, info.BusinessName, info.PushName); name != "" {
				id.name = name
				break
			}
		}
	}
	if id.name == "" {
		id.name = strings.TrimSpace(evt.Info.PushName)
	}
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func (m *Manager) sendWebhook(event string, data interface{}) {
	m.mu.RLock()
	url := m.webhookURL
	m.mu.RUnlock()

	if url == "" {
		return
	}

	body, err := json.Marshal(map[string]interface{}{
		"event":      event,
		"session_id": m.id,
		"timestamp":  time.Now(),
		"data":       data,
	})
	if err != nil {
		m.log.Errorf("marshal webhook: %v", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
		if err != nil {
			m.log.Errorf("webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			m.log.Errorf("webhook post: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}

func parseRecipient(arg string) (types.JID, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return types.EmptyJID, fmt.Errorf("nomor/jid kosong")
	}
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), nil
	}
	jid, err := types.ParseJID(arg)
	if err != nil {
		return types.EmptyJID, fmt.Errorf("jid tidak valid: %w", err)
	}
	return jid, nil
}

func extractMessageText(evt *events.Message) string {
	msg := evt.Message
	if msg.GetConversation() != "" {
		return msg.GetConversation()
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	if img := msg.GetImageMessage(); img != nil {
		return img.GetCaption()
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		return doc.GetCaption()
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		return vid.GetCaption()
	}
	return ""
}
