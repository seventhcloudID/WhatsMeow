package api

import (
	"net/http"

	"github.com/whatsmeow/gateway/internal/wa"
)

func NewRouter(sessions *wa.SessionManager, apiKey string) http.Handler {
	h := NewHandler(sessions, apiKey)
	mux := http.NewServeMux()

	// ws membungkus handler agar ter-scope ke session (X-Session-ID).
	ws := h.withSession

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/endpoints", h.APIIndex)

	mux.HandleFunc("GET /docs", h.Docs)
	mux.HandleFunc("GET /docs/", h.Docs)
	mux.HandleFunc("POST /docs/login", h.DocsLogin)
	mux.HandleFunc("POST /docs/logout", h.DocsLogout)

	mux.HandleFunc("GET /test", h.TestPage)

	// Sessions (manajemen tenant; tidak ber-scope)
	mux.HandleFunc("GET /sessions", h.ListSessions)
	mux.HandleFunc("POST /sessions", h.CreateSession)
	mux.HandleFunc("DELETE /sessions/{id}", h.DeleteSession)

	// Session (ber-scope per tenant)
	mux.HandleFunc("GET /session/status", ws((*Handler).SessionStatus))
	mux.HandleFunc("POST /session/connect", ws((*Handler).SessionConnect))
	mux.HandleFunc("GET /session/qr", ws((*Handler).SessionQR))
	mux.HandleFunc("POST /session/pair", ws((*Handler).SessionPair))
	mux.HandleFunc("POST /session/logout", ws((*Handler).SessionLogout))
	mux.HandleFunc("POST /session/reset", ws((*Handler).SessionReset))
	mux.HandleFunc("POST /session/disconnect", ws((*Handler).SessionDisconnect))

	// Messages
	mux.HandleFunc("POST /message/text", ws((*Handler).SendText))
	mux.HandleFunc("POST /message/image", ws((*Handler).SendImage))
	mux.HandleFunc("POST /message/video", ws((*Handler).SendVideo))
	mux.HandleFunc("POST /message/audio", ws((*Handler).SendAudio))
	mux.HandleFunc("POST /message/document", ws((*Handler).SendDocument))
	mux.HandleFunc("POST /message/sticker", ws((*Handler).SendSticker))
	mux.HandleFunc("POST /message/location", ws((*Handler).SendLocation))
	mux.HandleFunc("POST /message/contact", ws((*Handler).SendContact))
	mux.HandleFunc("POST /message/poll", ws((*Handler).SendPoll))
	mux.HandleFunc("POST /message/poll/vote", ws((*Handler).SendPollVote))
	mux.HandleFunc("POST /message/reaction", ws((*Handler).SendReaction))
	mux.HandleFunc("POST /message/revoke", ws((*Handler).SendRevoke))
	mux.HandleFunc("POST /message/edit", ws((*Handler).SendEdit))
	mux.HandleFunc("POST /message/disappearing", ws((*Handler).SetDisappearing))

	// Groups
	mux.HandleFunc("GET /groups", ws((*Handler).ListGroups))
	mux.HandleFunc("GET /groups/info", ws((*Handler).GetGroupInfo))
	mux.HandleFunc("POST /groups", ws((*Handler).CreateGroup))
	mux.HandleFunc("POST /groups/leave", ws((*Handler).LeaveGroup))
	mux.HandleFunc("GET /groups/invite", ws((*Handler).GroupInviteLink))
	mux.HandleFunc("GET /groups/preview", ws((*Handler).GroupInfoFromLink))
	mux.HandleFunc("POST /groups/join", ws((*Handler).JoinGroup))
	mux.HandleFunc("POST /groups/participants", ws((*Handler).UpdateGroupParticipants))
	mux.HandleFunc("PUT /groups/name", ws((*Handler).SetGroupName))
	mux.HandleFunc("PUT /groups/description", ws((*Handler).SetGroupDescription))
	mux.HandleFunc("POST /groups/photo", ws((*Handler).SetGroupPhoto))
	mux.HandleFunc("PUT /groups/locked", ws((*Handler).SetGroupLocked))
	mux.HandleFunc("PUT /groups/announce", ws((*Handler).SetGroupAnnounce))

	// Users
	mux.HandleFunc("POST /users/check", ws((*Handler).CheckWhatsApp))
	mux.HandleFunc("POST /users/info", ws((*Handler).GetUserInfo))
	mux.HandleFunc("GET /users/profile-picture", ws((*Handler).GetProfilePicture))
	mux.HandleFunc("PUT /users/status", ws((*Handler).SetAboutStatus))
	mux.HandleFunc("GET /users/business", ws((*Handler).GetBusinessProfile))
	mux.HandleFunc("GET /users/blocklist", ws((*Handler).GetBlocklist))
	mux.HandleFunc("POST /users/blocklist", ws((*Handler).UpdateBlocklist))
	mux.HandleFunc("GET /users/privacy", ws((*Handler).GetPrivacySettings))
	mux.HandleFunc("POST /users/devices", ws((*Handler).GetUserDevices))

	// Presence
	mux.HandleFunc("POST /presence", ws((*Handler).SendGlobalPresence))
	mux.HandleFunc("POST /presence/typing", ws((*Handler).SendTyping))
	mux.HandleFunc("POST /presence/subscribe", ws((*Handler).SubscribePresence))

	// Chats
	mux.HandleFunc("POST /chats/read", ws((*Handler).MarkRead))
	mux.HandleFunc("POST /chats/action", ws((*Handler).ChatAction))

	// Media
	mux.HandleFunc("POST /media/download", ws((*Handler).DownloadMedia))

	// Newsletter
	mux.HandleFunc("GET /newsletters", ws((*Handler).ListNewsletters))
	mux.HandleFunc("GET /newsletters/info", ws((*Handler).GetNewsletterInfo))
	mux.HandleFunc("GET /newsletters/invite", ws((*Handler).GetNewsletterByInvite))
	mux.HandleFunc("POST /newsletters/follow", ws((*Handler).FollowNewsletter))
	mux.HandleFunc("POST /newsletters/unfollow", ws((*Handler).UnfollowNewsletter))
	mux.HandleFunc("POST /newsletters", ws((*Handler).CreateNewsletter))
	mux.HandleFunc("GET /newsletters/messages", ws((*Handler).NewsletterMessages))
	mux.HandleFunc("PUT /newsletters/mute", ws((*Handler).NewsletterMute))

	// Webhook (per session)
	mux.HandleFunc("GET /webhook", ws((*Handler).GetWebhook))
	mux.HandleFunc("PUT /webhook", ws((*Handler).SetWebhook))

	var handler http.Handler = mux
	handler = APIKeyMiddleware(apiKey)(handler)
	handler = CORSMiddleware(handler)

	return handler
}
