// Command server startet die Insider-API (§3).
package main

import (
	"context"
	"log"

	"github.com/kommaufdenpunkt/insider/internal/auth"
	"github.com/kommaufdenpunkt/insider/internal/config"
	"github.com/kommaufdenpunkt/insider/internal/db"
	"github.com/kommaufdenpunkt/insider/internal/groups"
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

	// Gruppen (Phase 2) — wird auch als GroupJoiner an den Auth-Service gereicht,
	// damit die Registrierung mit Gruppen-Einladung direkt beitritt.
	groupsRepo := groups.NewPgRepository(pool)
	groupsSvc := groups.NewService(groupsRepo)
	groupsH := groups.NewHandler(groupsSvc)

	authRepo := auth.NewPgRepository(pool)
	authSvc := auth.NewService(authRepo, jwt, groupsSvc, cfg.RequireEmailVerification, cfg.EmailVerificationTTL)
	authH := auth.NewHandler(authSvc)

	router, err := httpx.NewRouter(cfg, jwt, authH, groupsH)
	if err != nil {
		log.Fatalf("Router: %v", err)
	}

	log.Printf("Insider-API lauscht auf %s", cfg.Addr())
	if err := router.Run(cfg.Addr()); err != nil {
		log.Fatalf("Server beendet: %v", err)
	}
}
