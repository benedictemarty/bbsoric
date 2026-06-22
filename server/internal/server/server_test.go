package server

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// echoHandler : handler de test qui renvoie chaque ligne reçue.
type echoHandler struct{}

func (echoHandler) Handle(ctx context.Context, s *Session) {
	for {
		line, err := s.ReadLine()
		if err != nil {
			return
		}
		if err := s.Println("echo:" + line); err != nil {
			return
		}
	}
}

// blockHandler : reste connecté jusqu'à un signal, pour tester les limites.
type blockHandler struct{ release chan struct{} }

func (h blockHandler) Handle(ctx context.Context, s *Session) {
	select {
	case <-h.release:
	case <-ctx.Done():
	}
}

func startServer(t *testing.T, cfg Config, h Handler) (addr string, stop func()) {
	t.Helper()
	cfg.Addr = "127.0.0.1:0"
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := New(cfg, h, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.Serve(ctx, ln)
	}()
	return ln.Addr().String(), func() {
		cancel()
		_ = ln.Close()
		wg.Wait()
	}
}

func TestEchoRoundTrip(t *testing.T) {
	addr, stop := startServer(t, Config{IdleTimeout: 2 * time.Second}, echoHandler{})
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("bonjour\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(line); got != "echo:bonjour" {
		t.Fatalf("réponse inattendue: %q", got)
	}
}

func TestCROnlyLineTermination(t *testing.T) {
	// L'Oric envoie CR ($0D) seul sur RETURN : la ligne doit se terminer.
	addr, stop := startServer(t, Config{IdleTimeout: 2 * time.Second}, echoHandler{})
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("salut\r")); err != nil { // CR seul, pas de LF
		t.Fatalf("write: %v", err)
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(line); got != "echo:salut" {
		t.Fatalf("CR seul non traité comme fin de ligne: %q", got)
	}
}

func TestTelnetIACStripped(t *testing.T) {
	addr, stop := startServer(t, Config{IdleTimeout: 2 * time.Second}, echoHandler{})
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// IAC WILL ECHO (0xFF 0xFB 0x01) inséré au milieu de la saisie.
	payload := append([]byte("ab"), iac, telnetWill, 1)
	payload = append(payload, []byte("cd\r\n")...)
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(line); got != "echo:abcd" {
		t.Fatalf("IAC non filtré, got %q", got)
	}
}

func TestMaxConnsPerIP(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	addr, stop := startServer(t, Config{MaxConnsPerIP: 1}, blockHandler{release: release})
	defer stop()

	// 1re connexion : acceptée et maintenue.
	c1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial1: %v", err)
	}
	defer c1.Close()
	// Laisse le serveur enregistrer la connexion.
	waitActive(t, addr)

	// 2e connexion depuis la même IP : doit être refusée avec un message.
	c2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial2: %v", err)
	}
	defer c2.Close()
	_ = c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf, _ := io.ReadAll(c2)
	if !strings.Contains(string(buf), "Trop de connexions") {
		t.Fatalf("2e connexion non refusée, reçu %q", string(buf))
	}
}

// waitActive ouvre puis ferme une sonde pour laisser le serveur traiter les
// connexions précédentes (évite une course sur le compteur par IP).
func waitActive(t *testing.T, addr string) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)
}
