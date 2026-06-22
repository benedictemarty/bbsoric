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

	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// startWelcomeServer démarre un vrai serveur BBS avec le WelcomeHandler sur un
// port éphémère et renvoie son adresse + une fonction d'arrêt.
func startWelcomeServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
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

// readUntil lit (par octets, car les invites n'ont pas de \n) jusqu'à voir le
// marqueur ou expiration, et renvoie le cumul.
func readUntil(t *testing.T, r *bufio.Reader, conn net.Conn, marker string) string {
	t.Helper()
	var acc strings.Builder
	buf := make([]byte, 256)
	deadline := time.Now().Add(2 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		n, err := r.Read(buf)
		if n > 0 {
			acc.Write(buf[:n])
			if strings.Contains(acc.String(), marker) {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return acc.String()
}

func TestBannerAndMenu(t *testing.T) {
	addr, stop := startWelcomeServer(t)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	// Bannière + menu principal apparaissent à la connexion.
	out := readUntil(t, r, conn, "Votre choix")
	if !strings.Contains(out, "B B S   O R I C") {
		t.Errorf("bannière absente:\n%s", out)
	}
	if !strings.Contains(out, "MENU PRINCIPAL") {
		t.Errorf("menu absent:\n%s", out)
	}
}

func TestMenuNavigationAndQuit(t *testing.T) {
	addr, stop := startWelcomeServer(t)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")

	// Touche unique '1' (sans RETURN) -> écran Informations systeme
	if _, err := conn.Write([]byte("1")); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := readUntil(t, r, conn, "une touche")
	if !strings.Contains(out, "INFORMATIONS SYSTEME") {
		t.Errorf("écran info attendu, reçu:\n%s", out)
	}

	// Une touche quelconque (espace) -> revient au menu
	if _, err := conn.Write([]byte(" ")); err != nil {
		t.Fatalf("write: %v", err)
	}
	readUntil(t, r, conn, "Votre choix")

	// Q -> quitte
	if _, err := conn.Write([]byte("Q")); err != nil {
		t.Fatalf("write: %v", err)
	}
	rest, _ := io.ReadAll(r)
	if !strings.Contains(string(rest), "A bientot") {
		t.Errorf("Q n'a pas quitté proprement, reçu:\n%s", string(rest))
	}
}

func TestFirstKey(t *testing.T) {
	cases := map[string]byte{"": 0, "1": '1', " q ": 'Q', "Quit": 'Q', "  ": 0, "abc": 'A'}
	for in, want := range cases {
		if got := firstKey(in); got != want {
			t.Errorf("firstKey(%q) = %q, want %q", in, got, want)
		}
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
