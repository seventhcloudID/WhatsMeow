package api

import (
	"net/http"

	"github.com/whatsmeow/gateway/internal/wa"
)

func NewRouter(manager *wa.Manager, apiKey string) http.Handler {
	h := NewHandler(manager, apiKey)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)

	mux.HandleFunc("GET /docs", h.Docs)
	mux.HandleFunc("GET /docs/", h.Docs)
	mux.HandleFunc("POST /docs/login", h.DocsLogin)
	mux.HandleFunc("POST /docs/logout", h.DocsLogout)

	mux.HandleFunc("GET /session/status", h.SessionStatus)
	mux.HandleFunc("POST /session/connect", h.SessionConnect)
	mux.HandleFunc("GET /session/qr", h.SessionQR)
	mux.HandleFunc("POST /session/pair", h.SessionPair)
	mux.HandleFunc("POST /session/logout", h.SessionLogout)
	mux.HandleFunc("POST /session/reset", h.SessionReset)
	mux.HandleFunc("POST /session/disconnect", h.SessionDisconnect)

	mux.HandleFunc("POST /message/text", h.SendText)
	mux.HandleFunc("POST /message/image", h.SendImage)
	mux.HandleFunc("POST /message/document", h.SendDocument)

	mux.HandleFunc("GET /webhook", h.GetWebhook)
	mux.HandleFunc("PUT /webhook", h.SetWebhook)

	var handler http.Handler = mux
	handler = APIKeyMiddleware(apiKey)(handler)
	handler = CORSMiddleware(handler)

	return handler
}
