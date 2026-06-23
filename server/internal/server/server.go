// Package server fournit la couche réseau du BBS Oric : écoute TCP, gestion
// d'une tâche par connexion et garde-fous d'exposition Internet (limite de
// connexions globale et par IP, timeout d'inactivité, journalisation).
//
// La logique applicative est fournie via un Handler, ce qui garde la couche
// réseau agnostique du contenu du BBS (moteur de menus, couche OASCII…).
package server

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Config regroupe les paramètres d'exécution du serveur.
type Config struct {
	Addr          string        // adresse d'écoute, ex. "0.0.0.0:6502"
	MaxConns      int           // connexions simultanées max (0 = illimité)
	MaxConnsPerIP int           // connexions simultanées max par IP (0 = illimité)
	IdleTimeout   time.Duration // délai d'inactivité avant déconnexion (0 = aucun)
}

// Handler traite une session cliente. Handle doit rendre la main lorsque la
// session se termine ; la connexion est alors fermée par le serveur.
type Handler interface {
	Handle(ctx context.Context, s *Session)
}

// Server est un serveur BBS TCP.
type Server struct {
	cfg     Config
	handler Handler
	log     *slog.Logger
	start   time.Time // instant de création, base du calcul d'uptime

	mu    sync.Mutex
	perIP map[string]int // nombre de connexions actives par IP
	slots chan struct{}  // sémaphore global de connexions

	// Compteurs de supervision (cf. Stats / MetricsHandler).
	connTotal    atomic.Int64 // connexions TCP acceptées (cumul)
	connActive   atomic.Int64 // sessions en cours (ayant passé les garde-fous)
	connRejected atomic.Int64 // connexions refusées par un garde-fou (cumul)
}

// New construit un serveur. Si log est nil, le logger par défaut est utilisé.
func New(cfg Config, handler Handler, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		cfg:     cfg,
		handler: handler,
		log:     log,
		start:   time.Now(),
		perIP:   make(map[string]int),
	}
	if cfg.MaxConns > 0 {
		s.slots = make(chan struct{}, cfg.MaxConns)
	}
	return s
}

// Stats est un instantané des métriques de supervision du serveur.
type Stats struct {
	Uptime       time.Duration
	ConnTotal    int64 // connexions TCP acceptées (cumul)
	ConnActive   int64 // sessions en cours
	ConnRejected int64 // connexions refusées par un garde-fou (cumul)
}

// Stats renvoie un instantané cohérent des compteurs (lecture atomique).
func (s *Server) Stats() Stats {
	return Stats{
		Uptime:       time.Since(s.start),
		ConnTotal:    s.connTotal.Load(),
		ConnActive:   s.connActive.Load(),
		ConnRejected: s.connRejected.Load(),
	}
}

// ListenAndServe écoute sur l'adresse configurée et sert jusqu'à l'annulation
// du contexte.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	s.log.Info("serveur BBS Oric en écoute", "addr", ln.Addr().String())
	s.Serve(ctx, ln)
	return nil
}

// Serve accepte les connexions sur le listener fourni jusqu'à l'annulation du
// contexte, puis attend la fin des sessions en cours. Utile pour fournir son
// propre listener (tests, activation par socket systemd).
func (s *Server) Serve(ctx context.Context, ln net.Listener) {
	// Ferme le listener à l'annulation pour débloquer Accept.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var wg sync.WaitGroup
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break // arrêt demandé
			}
			s.log.Warn("échec Accept", "err", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.serveConn(ctx, conn)
		}()
	}
	wg.Wait()
	s.log.Info("serveur arrêté")
}

// serveConn applique les garde-fous puis délègue au Handler.
func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	s.connTotal.Add(1)
	ip := remoteIP(conn)

	// Garde-fou 1 : limite globale de connexions simultanées.
	if s.slots != nil {
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			s.connRejected.Add(1)
			s.log.Warn("connexion refusée : limite globale atteinte", "ip", ip)
			_, _ = conn.Write([]byte("\r\nServeur sature, reessayez plus tard.\r\n"))
			return
		}
	}

	// Garde-fou 2 : limite de connexions simultanées par IP.
	if !s.acquireIP(ip) {
		s.connRejected.Add(1)
		s.log.Warn("connexion refusée : limite par IP atteinte", "ip", ip)
		_, _ = conn.Write([]byte("\r\nTrop de connexions depuis votre adresse.\r\n"))
		return
	}
	defer s.releaseIP(ip)

	s.connActive.Add(1)
	defer s.connActive.Add(-1)

	s.log.Info("connexion", "ip", ip)
	start := time.Now()

	sess := newSession(conn, s.cfg.IdleTimeout)
	s.handler.Handle(ctx, sess)

	s.log.Info("déconnexion", "ip", ip, "duree", time.Since(start).Round(time.Second).String())
}

// acquireIP tente de réserver un emplacement pour l'IP donnée.
func (s *Server) acquireIP(ip string) bool {
	if s.cfg.MaxConnsPerIP <= 0 {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip] >= s.cfg.MaxConnsPerIP {
		return false
	}
	s.perIP[ip]++
	return true
}

func (s *Server) releaseIP(ip string) {
	if s.cfg.MaxConnsPerIP <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip]--; s.perIP[ip] <= 0 {
		delete(s.perIP, ip)
	}
}

func remoteIP(conn net.Conn) string {
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return conn.RemoteAddr().String()
	}
	return host
}
