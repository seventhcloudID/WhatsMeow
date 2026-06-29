package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/whatsmeow/gateway/internal/wa"
)

type Handler struct {
	wa       *wa.Manager
	sessions *wa.SessionManager
	apiKey   string
}

func NewHandler(sessions *wa.SessionManager, apiKey string) *Handler {
	return &Handler{sessions: sessions, apiKey: apiKey}
}

// sessionID mengambil id tenant dari header X-Session-ID; default bila kosong.
func sessionID(r *http.Request) string {
	id := strings.TrimSpace(r.Header.Get("X-Session-ID"))
	if id == "" {
		return wa.DefaultSessionID
	}
	return id
}

// withSession membungkus handler ber-sesi: me-resolve Manager tenant lalu
// memanggil handler dengan Handler yang sudah ter-scope ke Manager itu,
// sehingga seluruh method handler (yang memakai h.wa) tidak perlu diubah.
func (h *Handler) withSession(fn func(*Handler, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mgr, err := h.sessions.GetOrCreate(r.Context(), sessionID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		scoped := &Handler{wa: mgr, sessions: h.sessions, apiKey: h.apiKey}
		fn(scoped, w, r)
	}
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, h.sessions.List())
}

type createSessionRequest struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, http.StatusBadRequest, "field 'id' wajib diisi")
		return
	}
	if _, err := h.sessions.Create(r.Context(), strings.TrimSpace(req.ID), strings.TrimSpace(req.Label)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"id": strings.TrimSpace(req.ID), "message": "session dibuat"})
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib")
		return
	}
	if err := h.sessions.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "session dihapus"})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]string{"status": "ok"})
}

func (h *Handler) SessionStatus(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, h.wa.Status())
}

func (h *Handler) SessionConnect(w http.ResponseWriter, r *http.Request) {
	if err := h.wa.Start(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, h.wa.Status())
}

func (h *Handler) SessionQR(w http.ResponseWriter, r *http.Request) {
	qr, err := h.wa.GetQR()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(w, qr)
}

type pairRequest struct {
	Phone string `json:"phone"`
}

func (h *Handler) SessionPair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}

	code, err := h.wa.PairPhone(r.Context(), req.Phone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, map[string]string{
		"code":    code,
		"message": "Masukkan kode ini di WhatsApp > Perangkat Tertaut > Tautkan perangkat",
	})
}

func (h *Handler) SessionLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.wa.Logout(r.Context()); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "logout berhasil"})
}

func (h *Handler) SessionReset(w http.ResponseWriter, r *http.Request) {
	if err := h.wa.ResetSession(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{
		"message": "Session direset. Tunggu 5 detik lalu ambil QR baru via GET /session/qr",
	})
}

func (h *Handler) SessionDisconnect(w http.ResponseWriter, r *http.Request) {
	h.wa.Disconnect()
	writeSuccess(w, map[string]string{"message": "disconnect berhasil"})
}

type sendTextRequest struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

func (h *Handler) SendText(w http.ResponseWriter, r *http.Request) {
	var req sendTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}
	if req.To == "" || req.Text == "" {
		writeError(w, http.StatusBadRequest, "field 'to' dan 'text' wajib diisi")
		return
	}

	result, err := h.wa.SendText(r.Context(), req.To, req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, result)
}

func (h *Handler) SendImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}

	to := r.FormValue("to")
	caption := r.FormValue("caption")
	if to == "" {
		writeError(w, http.StatusBadRequest, "field 'to' wajib diisi")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file wajib diupload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}

	result, err := h.wa.SendImage(r.Context(), to, data, caption)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, result)
}

func (h *Handler) SendDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}

	to := r.FormValue("to")
	filename := r.FormValue("filename")
	mimetype := r.FormValue("mimetype")
	if to == "" {
		writeError(w, http.StatusBadRequest, "field 'to' wajib diisi")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file wajib diupload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}

	result, err := h.wa.SendDocument(r.Context(), to, data, filename, mimetype)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, result)
}

type webhookRequest struct {
	URL string `json:"url"`
}

func (h *Handler) SetWebhook(w http.ResponseWriter, r *http.Request) {
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}

	if err := h.sessions.SetWebhook(h.wa.ID(), req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"webhook_url": req.URL})
}

func (h *Handler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]string{"webhook_url": h.wa.GetWebhookURL()})
}
