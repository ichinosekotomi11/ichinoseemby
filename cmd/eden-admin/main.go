package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ichinosekotomi11/ichinoseemby/internal/admin"
)

func main() {
	cfg := admin.LoadConfigFromEnv()
	fieldCipher, err := admin.LoadFieldCipherFromEnv()
	if err != nil {
		log.Fatalf("load data encryption key: %v", err)
	}
	admin.SetDefaultFieldCipher(fieldCipher)

	store, err := admin.OpenStore(cfg.DataPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	service := admin.NewService(cfg, store, admin.NewEmbyClient(cfg.Emby))
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           service.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("eden admin backend listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
