package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/whatsmeow/gateway/internal/wa"
)

type Handler struct {
	wa     *wa.Manager
	apiKey string
}

func NewHandler(manager *wa.Manager, apiKey string) *Handler {
	return &Handler{wa: manager, apiKey: apiKey}
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

	h.wa.SetWebhookURL(req.URL)
	writeSuccess(w, map[string]string{"webhook_url": req.URL})
}

func (h *Handler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]string{"webhook_url": h.wa.GetWebhookURL()})
}
