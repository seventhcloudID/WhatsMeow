package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	manager, err := wa.NewManager(cfg.DBPath, cfg.LogLevel)
	if err != nil {
		log.Fatalf("init whatsapp: %v", err)
	}

	if cfg.WebhookURL != "" {
		manager.SetWebhookURL(cfg.WebhookURL)
	}

	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Printf("warning: auto-connect gagal: %v", err)
	}

	router := api.NewRouter(manager, cfg.APIKey)
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

	manager.Disconnect()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
