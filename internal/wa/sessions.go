package wa

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DefaultSessionID dipakai bila request tidak menyertakan header X-Session-ID,
// agar pemakaian single-tenant lama (dan halaman /test) tetap jalan.
const DefaultSessionID = "default"

// SessionMeta adalah metadata per-tenant yang dipersist ke data/sessions.json
// supaya daftar session + webhook bertahan setelah restart.
type SessionMeta struct {
	ID         string    `json:"id"`
	Label      string    `json:"label,omitempty"`
	WebhookURL string    `json:"webhook_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// SessionView dipakai untuk respons API daftar session.
type SessionView struct {
	ID         string        `json:"id"`
	Label      string        `json:"label,omitempty"`
	WebhookURL string        `json:"webhook_url,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	Status     SessionStatus `json:"status"`
}

// SessionManager memegang banyak Manager (satu per tenant), masing-masing
// dengan file DB sendiri di dir/<id>.db sehingga lock SQLite tidak saling
// mengganggu antar tenant.
type SessionManager struct {
	mu       sync.RWMutex
	dir      string
	metaPath string
	logLevel string
	sessions map[string]*Manager
	meta     map[string]*SessionMeta
}

func NewSessionManager(dir, logLevel string) (*SessionManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	sm := &SessionManager{
		dir:      dir,
		metaPath: filepath.Join(filepath.Dir(dir), "sessions.json"),
		logLevel: logLevel,
		sessions: make(map[string]*Manager),
		meta:     make(map[string]*SessionMeta),
	}
	sm.loadMeta()
	return sm, nil
}

func (sm *SessionManager) dbPathFor(id string) string {
	return filepath.Join(sm.dir, id+".db")
}

func (sm *SessionManager) loadMeta() {
	data, err := os.ReadFile(sm.metaPath)
	if err != nil {
		return
	}
	var metas []*SessionMeta
	if err := json.Unmarshal(data, &metas); err != nil {
		return
	}
	for _, m := range metas {
		if m == nil || m.ID == "" {
			continue
		}
		sm.meta[m.ID] = m
	}
}

func (sm *SessionManager) saveMetaLocked() {
	metas := make([]*SessionMeta, 0, len(sm.meta))
	for _, m := range sm.meta {
		metas = append(metas, m)
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].ID < metas[j].ID })
	data, err := json.MarshalIndent(metas, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(sm.metaPath, data, 0644)
}

// migrateLegacyLocked memindahkan DB single-session lama (data/session.db)
// menjadi session "default" supaya akun yang sudah tertaut tidak perlu pair ulang.
func (sm *SessionManager) migrateLegacyLocked() {
	legacy := filepath.Join(filepath.Dir(sm.dir), "session.db")
	target := sm.dbPathFor(DefaultSessionID)
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	if _, err := os.Stat(target); err == nil {
		return
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Rename(legacy+suffix, target+suffix)
	}
}

func (sm *SessionManager) newManagerLocked(id, webhookURL string) (*Manager, error) {
	mgr, err := NewManager(id, sm.dbPathFor(id), sm.logLevel)
	if err != nil {
		return nil, err
	}
	if webhookURL != "" {
		mgr.SetWebhookURL(webhookURL)
	}
	return mgr, nil
}

// Load merestorasi semua session dari metadata dan auto-connect yang sudah login.
func (sm *SessionManager) Load(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.migrateLegacyLocked()

	if _, ok := sm.meta[DefaultSessionID]; !ok {
		sm.meta[DefaultSessionID] = &SessionMeta{ID: DefaultSessionID, CreatedAt: time.Now()}
	}

	for id, meta := range sm.meta {
		mgr, err := sm.newManagerLocked(id, meta.WebhookURL)
		if err != nil {
			return fmt.Errorf("load session %s: %w", id, err)
		}
		sm.sessions[id] = mgr
		if mgr.Status().LoggedIn {
			if err := mgr.Start(ctx); err != nil {
				mgr.log.Warnf("auto-connect session %s: %v", id, err)
			}
		}
	}

	sm.saveMetaLocked()
	return nil
}

// GetOrCreate mengembalikan Manager untuk id; bila belum ada, dibuat lazily.
func (sm *SessionManager) GetOrCreate(ctx context.Context, id string) (*Manager, error) {
	if id == "" {
		id = DefaultSessionID
	}

	sm.mu.RLock()
	mgr, ok := sm.sessions[id]
	sm.mu.RUnlock()
	if ok {
		return mgr, nil
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	if mgr, ok := sm.sessions[id]; ok {
		return mgr, nil
	}

	meta, ok := sm.meta[id]
	if !ok {
		meta = &SessionMeta{ID: id, CreatedAt: time.Now()}
		sm.meta[id] = meta
	}
	mgr, err := sm.newManagerLocked(id, meta.WebhookURL)
	if err != nil {
		return nil, err
	}
	sm.sessions[id] = mgr
	sm.saveMetaLocked()
	return mgr, nil
}

// Create membuat session secara eksplisit (opsional dengan label).
func (sm *SessionManager) Create(ctx context.Context, id, label string) (*Manager, error) {
	if id == "" {
		return nil, fmt.Errorf("session id wajib")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if mgr, ok := sm.sessions[id]; ok {
		if label != "" {
			if meta := sm.meta[id]; meta != nil {
				meta.Label = label
				sm.saveMetaLocked()
			}
		}
		return mgr, nil
	}

	meta := &SessionMeta{ID: id, Label: label, CreatedAt: time.Now()}
	mgr, err := sm.newManagerLocked(id, "")
	if err != nil {
		return nil, err
	}
	sm.meta[id] = meta
	sm.sessions[id] = mgr
	sm.saveMetaLocked()
	return mgr, nil
}

func (sm *SessionManager) Get(id string) (*Manager, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	mgr, ok := sm.sessions[id]
	return mgr, ok
}

func (sm *SessionManager) List() []SessionView {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	views := make([]SessionView, 0, len(sm.sessions))
	for id, mgr := range sm.sessions {
		v := SessionView{ID: id, Status: mgr.Status()}
		if meta := sm.meta[id]; meta != nil {
			v.Label = meta.Label
			v.WebhookURL = meta.WebhookURL
			v.CreatedAt = meta.CreatedAt
		}
		views = append(views, v)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views
}

// Delete melepas (logout) lalu menghapus session beserta file DB-nya.
func (sm *SessionManager) Delete(ctx context.Context, id string) error {
	if id == DefaultSessionID {
		return fmt.Errorf("session 'default' tidak bisa dihapus")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	mgr, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session '%s' tidak ditemukan", id)
	}

	mgr.Close(ctx)
	delete(sm.sessions, id)
	delete(sm.meta, id)
	removeSQLiteFiles(sm.dbPathFor(id))
	sm.saveMetaLocked()
	return nil
}

// SetWebhook mengubah webhook session sekaligus mempersist-nya.
func (sm *SessionManager) SetWebhook(id, url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	mgr, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("session '%s' tidak ditemukan", id)
	}
	mgr.SetWebhookURL(url)

	meta, ok := sm.meta[id]
	if !ok {
		meta = &SessionMeta{ID: id, CreatedAt: time.Now()}
		sm.meta[id] = meta
	}
	meta.WebhookURL = url
	sm.saveMetaLocked()
	return nil
}

// CloseAll memutus semua koneksi (dipakai saat shutdown, tanpa logout).
func (sm *SessionManager) CloseAll() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, mgr := range sm.sessions {
		mgr.Disconnect()
	}
}
