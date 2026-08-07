// Command server startet die Insider-API (§3).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/kommaufdenpunkt/insider/internal/auth"
	"github.com/kommaufdenpunkt/insider/internal/config"
	"github.com/kommaufdenpunkt/insider/internal/db"
	"github.com/kommaufdenpunkt/insider/internal/fahrstunden"
	"github.com/kommaufdenpunkt/insider/internal/fidolin"
	"github.com/kommaufdenpunkt/insider/internal/groups"
	"github.com/kommaufdenpunkt/insider/internal/httpx"
	"github.com/kommaufdenpunkt/insider/internal/moderation"
	"github.com/kommaufdenpunkt/insider/internal/posts"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Konfiguration: %v", err)
	}

	// Beendet sauber bei STRG-C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Datenbank: %v", err)
	}
	defer pool.Close()

	jwt := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)

	// Gruppen (Phase 2) — auch GroupJoiner und Membership-Prüfer.
	groupsSvc := groups.NewService(groups.NewPgRepository(pool))
	groupsH := groups.NewHandler(groupsSvc)

	// Auth (Phase 1).
	authSvc := auth.NewService(auth.NewPgRepository(pool), jwt, groupsSvc, cfg.RequireEmailVerification, cfg.EmailVerificationTTL)
	authH := auth.NewHandler(authSvc)

	// Beiträge + Feed (Phase 3).
	postsSvc := posts.NewService(posts.NewPgRepository(pool), groupsSvc)
	postsH := posts.NewHandler(postsSvc)

	// Fahrstunden-Nachweis: das Nebenbuch zum FS Manager (gefahren am /
	// eingetragen am, Notiz, Unterschrift, PDF).
	fahrSvc := fahrstunden.NewService(fahrstunden.NewPgRepository(pool), cfg.FahrstundenTageslimit)
	fahrH := fahrstunden.NewHandler(fahrSvc, authSvc, cfg.FahrstundenFahrschule)
	fahrWeb := fahrstunden.NewWebHandler(cfg.FahrstundenAppName, "/v1/fahrstunden", "/v1/auth",
		cfg.FahrstundenTageslimit, cfg.FahrstundenFahrschule)

	// Fidolin-Worker (Moderation) — läuft im Hintergrund, stoppt mit ctx.
	blocklist := cfg.ModerationBlocklist
	if len(blocklist) == 0 {
		blocklist = fidolin.DefaultBlocklist
	}
	fid := fidolin.New(
		fidolin.NewPgStore(pool),
		moderation.NewPgRepository(pool),
		fidolin.NewHeuristicAnalyzer(blocklist),
		fidolin.Config{
			Workers:      cfg.FidolinWorkers,
			BatchSize:    cfg.FidolinBatchSize,
			PollInterval: cfg.FidolinPollInterval,
			StaleAfter:   cfg.FidolinStaleAfter,
		},
		log.Default(),
	)
	go fid.Run(ctx)

	router, err := httpx.NewRouter(cfg, jwt, authH, groupsH, postsH, fahrH, fahrWeb)
	if err != nil {
		log.Fatalf("Router: %v", err)
	}

	srv := &http.Server{Addr: cfg.Addr(), Handler: router}
	go func() {
		log.Printf("Insider-API lauscht auf %s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server beendet: %v", err)
		}
	}()

	// Auf Stopp-Signal warten, dann HTTP-Server geordnet herunterfahren.
	<-ctx.Done()
	log.Printf("Fahre herunter ...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown-Fehler: %v", err)
	}
}
