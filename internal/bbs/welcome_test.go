package bbs

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

	"github.com/bmarty/bbsoric/internal/server"
)

// startWelcomeServer démarre un vrai serveur BBS avec le WelcomeHandler sur un
// port éphémère et renvoie son adresse + une fonction d'arrêt.
func startWelcomeServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 2 * time.Second}
	srv := server.New(cfg, WelcomeHandler{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

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

func TestWelcomeBannerAndQuit(t *testing.T) {
	addr, stop := startWelcomeServer(t)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)

	// Lit la bannière (plusieurs lignes) jusqu'à l'invite "> ".
	var banner strings.Builder
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 8; i++ {
		line, err := r.ReadString('\n')
		banner.WriteString(line)
		if err != nil || strings.Contains(line, "QUIT pour quitter") {
			break
		}
	}
	if !strings.Contains(banner.String(), "B B S   O R I C") {
		t.Errorf("bannière absente:\n%s", banner.String())
	}

	// Envoie QUIT et vérifie la fin de session.
	if _, err := conn.Write([]byte("QUIT\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	rest, _ := io.ReadAll(r)
	if !strings.Contains(string(rest), "Au revoir") {
		t.Errorf("QUIT n'a pas terminé proprement, reçu:\n%s", string(rest))
	}
}

func TestCenter(t *testing.T) {
	got := center("abc")
	if strings.TrimSpace(got) != "abc" {
		t.Errorf("texte altéré: %q", got)
	}
	if len(got) <= 3 {
		t.Errorf("pas de centrage appliqué: %q", got)
	}
}
