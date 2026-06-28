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
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
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

func NewManager(dbPath, logLevel string) (*Manager, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbLog := waLog.Stdout("Database", logLevel, true)
	ctx := context.Background()

	container, err := sqlstore.New(ctx, "sqlite", dbPath+"?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", logLevel, true)
	client := whatsmeow.NewClient(device, clientLog)

	m := &Manager{
		client:    client,
		container: container,
		dbPath:    dbPath,
		logLevel:  logLevel,
		log:       clientLog,
	}

	client.AddEventHandler(m.eventHandler)
	return m, nil
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

	qrChan, err := m.client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("get qr channel: %w", err)
	}

	if err := m.client.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	go m.watchQR(qrChan)
	return nil
}

// ResetSession hapus session lama dan buat QR baru (fix pairing gagal).
func (m *Manager) ResetSession(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client.IsConnected() {
		m.client.Disconnect()
	}

	if err := m.container.Close(); err != nil {
		m.log.Warnf("close db: %v", err)
	}

	removeSQLiteFiles(m.dbPath)

	if err := m.reinitClient(ctx); err != nil {
		return fmt.Errorf("reinit client: %w", err)
	}

	return m.startQRLocked(ctx)
}

func (m *Manager) reinitClient(ctx context.Context) error {
	dbLog := waLog.Stdout("Database", m.logLevel, true)
	container, err := sqlstore.New(ctx, "sqlite", m.dbPath+"?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		return err
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}

	clientLog := waLog.Stdout("Client", m.logLevel, true)
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
		qrChan, err := m.client.GetQRChannel(ctx)
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

	if err := m.client.Logout(ctx); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	m.client.Disconnect()
	return nil
}

func (m *Manager) SendText(ctx context.Context, to, text string) (*SendResult, error) {
	jid, err := parseRecipient(to)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client.Store.ID == nil {
		return nil, fmt.Errorf("belum login")
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("tidak terhubung ke WhatsApp")
	}

	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return &SendResult{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

func (m *Manager) SendImage(ctx context.Context, to string, data []byte, caption string) (*SendResult, error) {
	jid, err := parseRecipient(to)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client.Store.ID == nil {
		return nil, fmt.Errorf("belum login")
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("tidak terhubung ke WhatsApp")
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}

	msg := &waProto.Message{ImageMessage: &waProto.ImageMessage{
		Caption:       proto.String(caption),
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(http.DetectContentType(data)),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("send image: %w", err)
	}

	return &SendResult{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

func (m *Manager) SendDocument(ctx context.Context, to string, data []byte, filename, mimetype string) (*SendResult, error) {
	jid, err := parseRecipient(to)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client.Store.ID == nil {
		return nil, fmt.Errorf("belum login")
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("tidak terhubung ke WhatsApp")
	}

	if mimetype == "" {
		mimetype = http.DetectContentType(data)
	}
	if filename == "" {
		filename = "document"
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("upload document: %w", err)
	}

	msg := &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		FileName:      proto.String(filename),
	}}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("send document: %w", err)
	}

	return &SendResult{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
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
		m.log.Warnf("Logout: %s", evt.Reason)
	case *events.PairSuccess:
		m.log.Infof("Pairing berhasil: %s", evt.ID)
	}
}

func (m *Manager) handleIncomingMessage(evt *events.Message) {
	text := extractMessageText(evt)
	payload := map[string]interface{}{
		"message_id": evt.Info.ID,
		"from":       evt.Info.Sender.String(),
		"chat":       evt.Info.Chat.String(),
		"timestamp":  evt.Info.Timestamp,
		"push_name":  evt.Info.PushName,
		"is_group":   evt.Info.IsGroup,
		"text":       text,
	}

	m.log.Infof("Pesan masuk dari %s: %s", evt.Info.Sender, text)
	m.sendWebhook("message.received", payload)
}

func (m *Manager) sendWebhook(event string, data interface{}) {
	m.mu.RLock()
	url := m.webhookURL
	m.mu.RUnlock()

	if url == "" {
		return
	}

	body, err := json.Marshal(map[string]interface{}{
		"event":     event,
		"timestamp": time.Now(),
		"data":      data,
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
