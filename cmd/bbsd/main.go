// Commande bbsd : démon du serveur BBS Oric.
//
// Exemple :
//
//	bbsd                      # écoute 0.0.0.0:6502
//	bbsd -addr 0.0.0.0:6502 -max-conns 50 -max-conns-per-ip 3 -idle 5m
//
// Variables d'environnement équivalentes : BBS_ADDR.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bmarty/bbsoric/internal/bbs"
	"github.com/bmarty/bbsoric/internal/server"
)

func main() {
	defaultAddr := os.Getenv("BBS_ADDR")
	if defaultAddr == "" {
		defaultAddr = "0.0.0.0:6502" // port 6502 : clin d'œil au CPU de l'Oric
	}

	addr := flag.String("addr", defaultAddr, "adresse d'écoute host:port")
	maxConns := flag.Int("max-conns", 50, "connexions simultanées max (0 = illimité)")
	maxPerIP := flag.Int("max-conns-per-ip", 3, "connexions simultanées max par IP (0 = illimité)")
	idle := flag.Duration("idle", 5*time.Minute, "délai d'inactivité avant déconnexion (0 = aucun)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := server.Config{
		Addr:          *addr,
		MaxConns:      *maxConns,
		MaxConnsPerIP: *maxPerIP,
		IdleTimeout:   *idle,
	}
	srv := server.New(cfg, bbs.WelcomeHandler{}, log)

	// Arrêt propre sur SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.ListenAndServe(ctx); err != nil {
		log.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}
