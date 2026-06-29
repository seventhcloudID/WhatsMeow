package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/whatsmeow/gateway/internal/wa"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return false
	}
	return true
}

func readMultipartFile(w http.ResponseWriter, r *http.Request, maxMB int64) (to, caption, filename, mimetype string, data []byte, ok bool) {
	if err := r.ParseMultipartForm(maxMB << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}
	to = r.FormValue("to")
	caption = r.FormValue("caption")
	filename = r.FormValue("filename")
	mimetype = r.FormValue("mimetype")
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
	data, err = io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}
	ok = true
	return
}

func handleWA(w http.ResponseWriter, fn func() (any, error)) {
	result, err := fn()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, result)
}

func handleWAErr(w http.ResponseWriter, fn func() error) {
	if err := fn(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "ok"})
}

// --- Extended message handlers ---

func (h *Handler) SendVideo(w http.ResponseWriter, r *http.Request) {
	to, caption, _, _, data, ok := readMultipartFile(w, r, 64)
	if !ok {
		return
	}
	handleWA(w, func() (any, error) {
		return h.wa.SendVideo(r.Context(), to, data, caption)
	})
}

func (h *Handler) SendAudio(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}
	to := r.FormValue("to")
	ptt := r.FormValue("ptt") == "true" || r.FormValue("ptt") == "1"
	file, _, err := r.FormFile("file")
	if err != nil || to == "" {
		writeError(w, http.StatusBadRequest, "to dan file wajib diisi")
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	handleWA(w, func() (any, error) {
		return h.wa.SendAudio(r.Context(), to, data, ptt, 0)
	})
}

func (h *Handler) SendSticker(w http.ResponseWriter, r *http.Request) {
	to, _, _, _, data, ok := readMultipartFile(w, r, 16)
	if !ok {
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendSticker(r.Context(), to, data) })
}

func (h *Handler) SendLocation(w http.ResponseWriter, r *http.Request) {
	var req wa.LocationOpts
	if !decodeJSON(w, r, &req) || req.To == "" {
		if req.To == "" {
			writeError(w, http.StatusBadRequest, "field 'to' wajib diisi")
		}
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendLocation(r.Context(), req) })
}

func (h *Handler) SendContact(w http.ResponseWriter, r *http.Request) {
	var req wa.ContactOpts
	if !decodeJSON(w, r, &req) || req.To == "" || req.VCard == "" {
		writeError(w, http.StatusBadRequest, "to dan vcard wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendContact(r.Context(), req) })
}

func (h *Handler) SendPoll(w http.ResponseWriter, r *http.Request) {
	var req wa.PollOpts
	if !decodeJSON(w, r, &req) || req.To == "" || req.Name == "" || len(req.Options) < 2 {
		writeError(w, http.StatusBadRequest, "to, name, dan minimal 2 options wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendPoll(r.Context(), req) })
}

func (h *Handler) SendPollVote(w http.ResponseWriter, r *http.Request) {
	var req wa.PollVoteOpts
	if !decodeJSON(w, r, &req) || req.Chat == "" || req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "chat dan message_id wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendPollVote(r.Context(), req) })
}

func (h *Handler) SendReaction(w http.ResponseWriter, r *http.Request) {
	var req wa.ReactionOpts
	if !decodeJSON(w, r, &req) || req.To == "" || req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "to dan message_id wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendReaction(r.Context(), req) })
}

func (h *Handler) SendRevoke(w http.ResponseWriter, r *http.Request) {
	var req wa.RevokeOpts
	if !decodeJSON(w, r, &req) || req.To == "" || req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "to dan message_id wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendRevoke(r.Context(), req) })
}

func (h *Handler) SendEdit(w http.ResponseWriter, r *http.Request) {
	var req wa.EditOpts
	if !decodeJSON(w, r, &req) || req.To == "" || req.MessageID == "" || req.Text == "" {
		writeError(w, http.StatusBadRequest, "to, message_id, dan text wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.SendEdit(r.Context(), req) })
}

func (h *Handler) SetDisappearing(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To      string `json:"to"`
		Seconds int    `json:"seconds"`
	}
	if !decodeJSON(w, r, &req) || req.To == "" {
		writeError(w, http.StatusBadRequest, "to wajib diisi")
		return
	}
	handleWAErr(w, func() error {
		return h.wa.SetDisappearing(r.Context(), wa.DisappearingOpts{
			To:    req.To,
			Timer: time.Duration(req.Seconds) * time.Second,
		})
	})
}

// --- Groups ---

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	handleWA(w, func() (any, error) { return h.wa.ListGroups(r.Context()) })
}

func (h *Handler) GetGroupInfo(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	if jid == "" {
		writeError(w, http.StatusBadRequest, "query jid wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetGroupInfo(r.Context(), jid) })
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req wa.CreateGroupOpts
	if !decodeJSON(w, r, &req) || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.CreateGroup(r.Context(), req) })
}

func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req struct{ JID string `json:"jid"` }
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.LeaveGroup(r.Context(), req.JID) })
}

func (h *Handler) GroupInviteLink(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	reset := r.URL.Query().Get("reset") == "true"
	if jid == "" {
		writeError(w, http.StatusBadRequest, "query jid wajib diisi")
		return
	}
	handleWA(w, func() (any, error) {
		link, err := h.wa.GetGroupInviteLink(r.Context(), jid, reset)
		return map[string]string{"invite_link": link}, err
	})
}

func (h *Handler) GroupInfoFromLink(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "query code wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetGroupInfoFromLink(r.Context(), code) })
}

func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req struct{ Code string `json:"code"` }
	if !decodeJSON(w, r, &req) || req.Code == "" {
		writeError(w, http.StatusBadRequest, "code wajib diisi")
		return
	}
	handleWA(w, func() (any, error) {
		jid, err := h.wa.JoinGroupWithLink(r.Context(), req.Code)
		return map[string]string{"group_jid": jid.String()}, err
	})
}

func (h *Handler) UpdateGroupParticipants(w http.ResponseWriter, r *http.Request) {
	var req wa.GroupParticipantsOpts
	if !decodeJSON(w, r, &req) || req.GroupJID == "" || req.Action == "" {
		writeError(w, http.StatusBadRequest, "group_jid dan action wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.UpdateGroupParticipants(r.Context(), req) })
}

func (h *Handler) SetGroupName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID  string `json:"jid"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SetGroupName(r.Context(), req.JID, req.Name) })
}

func (h *Handler) SetGroupDescription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID         string `json:"jid"`
		Description string `json:"description"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SetGroupDescription(r.Context(), req.JID, req.Description) })
}

func (h *Handler) SetGroupPhoto(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}
	jid := r.FormValue("jid")
	file, _, err := r.FormFile("file")
	if jid == "" || err != nil {
		writeError(w, http.StatusBadRequest, "jid dan file wajib diisi")
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	handleWA(w, func() (any, error) {
		id, err := h.wa.SetGroupPhoto(r.Context(), jid, data)
		return map[string]string{"photo_id": id}, err
	})
}

func (h *Handler) SetGroupLocked(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID    string `json:"jid"`
		Locked bool   `json:"locked"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SetGroupLocked(r.Context(), req.JID, req.Locked) })
}

func (h *Handler) SetGroupAnnounce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID      string `json:"jid"`
		Announce bool   `json:"announce"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SetGroupAnnounce(r.Context(), req.JID, req.Announce) })
}

// --- Users ---

func (h *Handler) CheckWhatsApp(w http.ResponseWriter, r *http.Request) {
	var req struct{ Phones []string `json:"phones"` }
	if !decodeJSON(w, r, &req) || len(req.Phones) == 0 {
		writeError(w, http.StatusBadRequest, "phones wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.CheckWhatsApp(r.Context(), req.Phones) })
}

func (h *Handler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct{ JIDs []string `json:"jids"` }
	if !decodeJSON(w, r, &req) || len(req.JIDs) == 0 {
		writeError(w, http.StatusBadRequest, "jids wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetUserInfo(r.Context(), req.JIDs) })
}

func (h *Handler) GetProfilePicture(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	if jid == "" {
		writeError(w, http.StatusBadRequest, "query jid wajib diisi")
		return
	}
	preview := r.URL.Query().Get("preview") == "true"
	handleWA(w, func() (any, error) { return h.wa.GetProfilePicture(r.Context(), jid, preview) })
}

func (h *Handler) SetAboutStatus(w http.ResponseWriter, r *http.Request) {
	var req struct{ Status string `json:"status"` }
	if !decodeJSON(w, r, &req) {
		return
	}
	handleWAErr(w, func() error { return h.wa.SetAboutStatus(r.Context(), req.Status) })
}

func (h *Handler) GetBusinessProfile(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	if jid == "" {
		writeError(w, http.StatusBadRequest, "query jid wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetBusinessProfile(r.Context(), jid) })
}

func (h *Handler) GetBlocklist(w http.ResponseWriter, r *http.Request) {
	handleWA(w, func() (any, error) { return h.wa.GetBlocklist(r.Context()) })
}

func (h *Handler) UpdateBlocklist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID    string `json:"jid"`
		Action string `json:"action"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.UpdateBlocklist(r.Context(), req.JID, req.Action) })
}

func (h *Handler) GetPrivacySettings(w http.ResponseWriter, r *http.Request) {
	handleWA(w, func() (any, error) { return h.wa.GetPrivacySettings(r.Context()) })
}

func (h *Handler) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	var req struct{ JIDs []string `json:"jids"` }
	if !decodeJSON(w, r, &req) || len(req.JIDs) == 0 {
		writeError(w, http.StatusBadRequest, "jids wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetUserDevices(r.Context(), req.JIDs) })
}

// --- Presence ---

func (h *Handler) SendGlobalPresence(w http.ResponseWriter, r *http.Request) {
	var req struct{ State string `json:"state"` }
	if !decodeJSON(w, r, &req) {
		return
	}
	handleWAErr(w, func() error { return h.wa.SendGlobalPresence(r.Context(), req.State) })
}

func (h *Handler) SendTyping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To    string `json:"to"`
		State string `json:"state"`
		Media string `json:"media"`
	}
	if !decodeJSON(w, r, &req) || req.To == "" {
		writeError(w, http.StatusBadRequest, "to wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SendTyping(r.Context(), req.To, req.State, req.Media) })
}

func (h *Handler) SubscribePresence(w http.ResponseWriter, r *http.Request) {
	var req struct{ JID string `json:"jid"` }
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.SubscribePresence(r.Context(), req.JID) })
}

// --- Chats ---

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req wa.MarkReadOpts
	if !decodeJSON(w, r, &req) || req.Chat == "" || len(req.MessageIDs) == 0 {
		writeError(w, http.StatusBadRequest, "chat dan message_ids wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.MarkRead(r.Context(), req) })
}

func (h *Handler) ChatAction(w http.ResponseWriter, r *http.Request) {
	var req wa.ChatActionOpts
	if !decodeJSON(w, r, &req) || req.Chat == "" {
		writeError(w, http.StatusBadRequest, "chat wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.ChatAction(r.Context(), req) })
}

// --- Media ---

func (h *Handler) DownloadMedia(w http.ResponseWriter, r *http.Request) {
	var req wa.DownloadMediaOpts
	if !decodeJSON(w, r, &req) || req.URL == "" || req.MediaKey == "" {
		writeError(w, http.StatusBadRequest, "url dan media_key wajib diisi")
		return
	}
	data, mime, err := h.wa.DownloadMedia(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "attachment")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// --- Newsletter ---

func (h *Handler) ListNewsletters(w http.ResponseWriter, r *http.Request) {
	handleWA(w, func() (any, error) { return h.wa.ListNewsletters(r.Context()) })
}

func (h *Handler) GetNewsletterInfo(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	if jid == "" {
		writeError(w, http.StatusBadRequest, "query jid wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetNewsletterInfo(r.Context(), jid) })
}

func (h *Handler) GetNewsletterByInvite(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("invite")
	if key == "" {
		writeError(w, http.StatusBadRequest, "query invite wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.GetNewsletterByInvite(r.Context(), key) })
}

func (h *Handler) FollowNewsletter(w http.ResponseWriter, r *http.Request) {
	var req struct{ JID string `json:"jid"` }
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.FollowNewsletter(r.Context(), req.JID) })
}

func (h *Handler) UnfollowNewsletter(w http.ResponseWriter, r *http.Request) {
	var req struct{ JID string `json:"jid"` }
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.UnfollowNewsletter(r.Context(), req.JID) })
}

func (h *Handler) CreateNewsletter(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid")
		return
	}
	name := r.FormValue("name")
	desc := r.FormValue("description")
	var picture []byte
	if file, _, err := r.FormFile("file"); err == nil {
		defer file.Close()
		picture, _ = io.ReadAll(file)
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return
	}
	handleWA(w, func() (any, error) { return h.wa.CreateNewsletter(r.Context(), name, desc, picture) })
}

func (h *Handler) NewsletterMessages(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	count := 10
	handleWA(w, func() (any, error) { return h.wa.NewsletterMessages(r.Context(), jid, count) })
}

func (h *Handler) NewsletterMute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID  string `json:"jid"`
		Mute bool   `json:"mute"`
	}
	if !decodeJSON(w, r, &req) || req.JID == "" {
		writeError(w, http.StatusBadRequest, "jid wajib diisi")
		return
	}
	handleWAErr(w, func() error { return h.wa.NewsletterToggleMute(r.Context(), req.JID, req.Mute) })
}

// API index for docs
func (h *Handler) APIIndex(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, apiEndpoints())
}
