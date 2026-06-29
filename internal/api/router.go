package api

import (
	"net/http"

	"github.com/whatsmeow/gateway/internal/wa"
)

func NewRouter(manager *wa.Manager, apiKey string) http.Handler {
	h := NewHandler(manager, apiKey)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/endpoints", h.APIIndex)

	mux.HandleFunc("GET /docs", h.Docs)
	mux.HandleFunc("GET /docs/", h.Docs)
	mux.HandleFunc("POST /docs/login", h.DocsLogin)
	mux.HandleFunc("POST /docs/logout", h.DocsLogout)

	// Session
	mux.HandleFunc("GET /session/status", h.SessionStatus)
	mux.HandleFunc("POST /session/connect", h.SessionConnect)
	mux.HandleFunc("GET /session/qr", h.SessionQR)
	mux.HandleFunc("POST /session/pair", h.SessionPair)
	mux.HandleFunc("POST /session/logout", h.SessionLogout)
	mux.HandleFunc("POST /session/reset", h.SessionReset)
	mux.HandleFunc("POST /session/disconnect", h.SessionDisconnect)

	// Messages
	mux.HandleFunc("POST /message/text", h.SendText)
	mux.HandleFunc("POST /message/image", h.SendImage)
	mux.HandleFunc("POST /message/video", h.SendVideo)
	mux.HandleFunc("POST /message/audio", h.SendAudio)
	mux.HandleFunc("POST /message/document", h.SendDocument)
	mux.HandleFunc("POST /message/sticker", h.SendSticker)
	mux.HandleFunc("POST /message/location", h.SendLocation)
	mux.HandleFunc("POST /message/contact", h.SendContact)
	mux.HandleFunc("POST /message/poll", h.SendPoll)
	mux.HandleFunc("POST /message/poll/vote", h.SendPollVote)
	mux.HandleFunc("POST /message/reaction", h.SendReaction)
	mux.HandleFunc("POST /message/revoke", h.SendRevoke)
	mux.HandleFunc("POST /message/edit", h.SendEdit)
	mux.HandleFunc("POST /message/disappearing", h.SetDisappearing)

	// Groups
	mux.HandleFunc("GET /groups", h.ListGroups)
	mux.HandleFunc("GET /groups/info", h.GetGroupInfo)
	mux.HandleFunc("POST /groups", h.CreateGroup)
	mux.HandleFunc("POST /groups/leave", h.LeaveGroup)
	mux.HandleFunc("GET /groups/invite", h.GroupInviteLink)
	mux.HandleFunc("GET /groups/preview", h.GroupInfoFromLink)
	mux.HandleFunc("POST /groups/join", h.JoinGroup)
	mux.HandleFunc("POST /groups/participants", h.UpdateGroupParticipants)
	mux.HandleFunc("PUT /groups/name", h.SetGroupName)
	mux.HandleFunc("PUT /groups/description", h.SetGroupDescription)
	mux.HandleFunc("POST /groups/photo", h.SetGroupPhoto)
	mux.HandleFunc("PUT /groups/locked", h.SetGroupLocked)
	mux.HandleFunc("PUT /groups/announce", h.SetGroupAnnounce)

	// Users
	mux.HandleFunc("POST /users/check", h.CheckWhatsApp)
	mux.HandleFunc("POST /users/info", h.GetUserInfo)
	mux.HandleFunc("GET /users/profile-picture", h.GetProfilePicture)
	mux.HandleFunc("PUT /users/status", h.SetAboutStatus)
	mux.HandleFunc("GET /users/business", h.GetBusinessProfile)
	mux.HandleFunc("GET /users/blocklist", h.GetBlocklist)
	mux.HandleFunc("POST /users/blocklist", h.UpdateBlocklist)
	mux.HandleFunc("GET /users/privacy", h.GetPrivacySettings)
	mux.HandleFunc("POST /users/devices", h.GetUserDevices)

	// Presence
	mux.HandleFunc("POST /presence", h.SendGlobalPresence)
	mux.HandleFunc("POST /presence/typing", h.SendTyping)
	mux.HandleFunc("POST /presence/subscribe", h.SubscribePresence)

	// Chats
	mux.HandleFunc("POST /chats/read", h.MarkRead)
	mux.HandleFunc("POST /chats/action", h.ChatAction)

	// Media
	mux.HandleFunc("POST /media/download", h.DownloadMedia)

	// Newsletter
	mux.HandleFunc("GET /newsletters", h.ListNewsletters)
	mux.HandleFunc("GET /newsletters/info", h.GetNewsletterInfo)
	mux.HandleFunc("GET /newsletters/invite", h.GetNewsletterByInvite)
	mux.HandleFunc("POST /newsletters/follow", h.FollowNewsletter)
	mux.HandleFunc("POST /newsletters/unfollow", h.UnfollowNewsletter)
	mux.HandleFunc("POST /newsletters", h.CreateNewsletter)
	mux.HandleFunc("GET /newsletters/messages", h.NewsletterMessages)
	mux.HandleFunc("PUT /newsletters/mute", h.NewsletterMute)

	// Webhook
	mux.HandleFunc("GET /webhook", h.GetWebhook)
	mux.HandleFunc("PUT /webhook", h.SetWebhook)

	var handler http.Handler = mux
	handler = APIKeyMiddleware(apiKey)(handler)
	handler = CORSMiddleware(handler)

	return handler
}
