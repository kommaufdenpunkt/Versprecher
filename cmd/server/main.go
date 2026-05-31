// Command server startet die Insider-API (§3).
package main

import (
	"context"
	"log"

	"github.com/kommaufdenpunkt/insider/internal/auth"
	"github.com/kommaufdenpunkt/insider/internal/config"
	"github.com/kommaufdenpunkt/insider/internal/db"
	"github.com/kommaufdenpunkt/insider/internal/httpx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Konfiguration: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Datenbank: %v", err)
	}
	defer pool.Close()

	jwt := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	repo := auth.NewPgRepository(pool)
	svc := auth.NewService(repo, jwt, cfg.RequireEmailVerification, cfg.EmailVerificationTTL)
	authH := auth.NewHandler(svc)

	router, err := httpx.NewRouter(cfg, jwt, authH)
	if err != nil {
		log.Fatalf("Router: %v", err)
	}

	log.Printf("Insider-API lauscht auf %s", cfg.Addr())
	if err := router.Run(cfg.Addr()); err != nil {
		log.Fatalf("Server beendet: %v", err)
	}
}
