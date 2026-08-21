package main

import (
	"log"
	"net/http"

	"github.com/Ribco/dinex/agent/internal/api"
	"github.com/Ribco/dinex/agent/internal/config"
	"github.com/Ribco/dinex/agent/internal/manager"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	mgr := manager.New(cfg)
	handler := api.New(cfg, mgr).Handler()

	log.Printf("🦖 Dinex Agent v%s", cfg.Version)
	log.Printf("Listening on %s", cfg.Listen)

	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
