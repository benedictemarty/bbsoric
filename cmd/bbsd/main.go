// Commande bbsd : démon du serveur BBS Oric.
//
// Exemple :
//
//	bbsd                      # écoute 0.0.0.0:6502 (telnet)
//	bbsd -addr 0.0.0.0:6502 -tls-addr 0.0.0.0:6992 -idle 5m
//
// Variables d'environnement équivalentes : BBS_ADDR.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benedictemarty/bbsoric/internal/bbs"
	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/server"
	"github.com/benedictemarty/bbsoric/internal/user"
)

func main() {
	defaultAddr := os.Getenv("BBS_ADDR")
	if defaultAddr == "" {
		defaultAddr = "0.0.0.0:6502" // port 6502 : clin d'œil au CPU de l'Oric
	}

	addr := flag.String("addr", defaultAddr, "adresse d'écoute telnet host:port")
	tlsAddr := flag.String("tls-addr", "", "adresse d'écoute TLS host:port (vide = désactivé)")
	tlsCert := flag.String("tls-cert", "", "fichier certificat TLS (vide = auto-signé)")
	tlsKey := flag.String("tls-key", "", "fichier clé TLS")
	maxConns := flag.Int("max-conns", 50, "connexions simultanées max (0 = illimité)")
	maxPerIP := flag.Int("max-conns-per-ip", 3, "connexions simultanées max par IP (0 = illimité)")
	idle := flag.Duration("idle", 5*time.Minute, "délai d'inactivité avant déconnexion (0 = aucun)")
	contentPath := flag.String("content", "", "fichier JSON du flux de pages (vide = contenu par défaut ; rechargé à chaud)")
	usersPath := flag.String("users", "", "fichier JSON des comptes (vide = comptes en mémoire, non persistés)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := server.Config{
		Addr:          *addr,
		MaxConns:      *maxConns,
		MaxConnsPerIP: *maxPerIP,
		IdleTimeout:   *idle,
	}
	store := content.NewStore(*contentPath, log)

	users, err := user.Open(*usersPath)
	if err != nil {
		log.Error("comptes : ouverture impossible", "path", *usersPath, "err", err)
		os.Exit(1)
	}
	if *usersPath != "" {
		log.Info("comptes chargés", "path", *usersPath, "comptes", users.Count())
	}

	srv := server.New(cfg, bbs.WelcomeHandler{Store: store, Users: users}, log)

	// Arrêt propre sur SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Écoute TLS optionnelle (même BBS, mêmes garde-fous), pour les modems WiFi
	// qui terminent le TLS (Pico W : ATDT#hôte:port).
	if *tlsAddr != "" {
		var cert tls.Certificate
		var err error
		if *tlsCert != "" && *tlsKey != "" {
			cert, err = tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		} else {
			cert, err = server.SelfSignedCert("bbsoric")
			log.Info("TLS : certificat auto-signé généré")
		}
		if err != nil {
			log.Error("TLS : certificat invalide", "err", err)
			os.Exit(1)
		}
		tlsLn, err := tls.Listen("tcp", *tlsAddr, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err != nil {
			log.Error("écoute TLS impossible", "err", err)
			os.Exit(1)
		}
		log.Info("serveur BBS Oric en écoute (TLS)", "addr", tlsLn.Addr().String())
		go srv.Serve(ctx, tlsLn)
	}

	if err := srv.ListenAndServe(ctx); err != nil {
		log.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}
