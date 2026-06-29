package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/whatsmeow/gateway/internal/api"
	"github.com/whatsmeow/gateway/internal/config"
	"github.com/whatsmeow/gateway/internal/wa"
)

func main() {
	_ = godotenv.Overload()

	cfg := config.Load()
	ctx := context.Background()

	sessionsDir := filepath.Join(filepath.Dir(cfg.DBPath), "sessions")
	sessions, err := wa.NewSessionManager(sessionsDir, cfg.LogLevel)
	if err != nil {
		log.Fatalf("init sessions: %v", err)
	}

	if err := sessions.Load(ctx); err != nil {
		log.Printf("warning: load sessions gagal: %v", err)
	}

	// WEBHOOK_URL dari .env diterapkan ke session default (kompat single-tenant).
	if cfg.WebhookURL != "" {
		if err := sessions.SetWebhook(wa.DefaultSessionID, cfg.WebhookURL); err != nil {
			log.Printf("warning: set webhook default gagal: %v", err)
		}
	}

	router := api.NewRouter(sessions, cfg.APIKey)
	addr := fmt.Sprintf(":%d", cfg.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("WhatsApp Gateway berjalan di http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sessions.CloseAll()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
