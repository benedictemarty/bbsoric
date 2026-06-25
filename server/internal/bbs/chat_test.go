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

	"github.com/benedictemarty/bbsoric/server/internal/presence"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// startBBSPresence démarre un BBS avec un registre de présence partagé.
func startBBSPresence(t *testing.T, json string) (addr string, reg *presence.Registry, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	reg = presence.New()
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, json), Presence: reg},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), reg, func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// chatSiteJSON : accueil -> invité -> menu (qui / chat).
const chatSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Qui est en ligne", "applet": "who", "next": "main" },
      { "key": "2", "label": "Chat", "applet": "chat", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// readFor lit jusqu'au marqueur (ou timeout), en posant une échéance fraîche.
func readFor(t *testing.T, r *bufio.Reader, conn net.Conn, marker string, d time.Duration) (string, bool) {
	t.Helper()
	var acc strings.Builder
	buf := make([]byte, 256)
	deadline := time.Now().Add(d)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		n, err := r.Read(buf)
		if n > 0 {
			acc.Write(buf[:n])
			if strings.Contains(acc.String(), marker) {
				return acc.String(), true
			}
		}
		if err != nil {
			break
		}
	}
	return acc.String(), false
}

// enterAsGuest connecte un client et le mène jusqu'au menu principal en invité.
func enterAsGuest(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	r := bufio.NewReader(conn)
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1")) // accès invité
	readUntil(t, r, conn, "touche")
	conn.Write([]byte(" ")) // appuyez sur une touche
	readUntil(t, r, conn, "Votre choix")
	return conn, r
}

func TestWhoApplet(t *testing.T) {
	addr, _, stop := startBBSPresence(t, chatSiteJSON)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("1")) // « Qui est en ligne »
	out, ok := readFor(t, r, conn, "appelant", time.Second)
	if !ok {
		t.Fatalf("liste des connectés non reçue ; vu : %q", out)
	}
	if !strings.Contains(out, "Invite-") {
		t.Errorf("pseudo invité absent de la liste ; vu : %q", out)
	}
	if !strings.Contains(out, "(vous)") {
		t.Errorf("marqueur (vous) absent ; vu : %q", out)
	}
}

func TestChatRelayBetweenTwoClients(t *testing.T) {
	addr, _, stop := startBBSPresence(t, chatSiteJSON)
	defer stop()

	// Client A entre dans le chat en premier (donc abonné en premier).
	connA, rA := enterAsGuest(t, addr)
	defer connA.Close()
	connA.Write([]byte("2")) // Chat
	if _, ok := readFor(t, rA, connA, "quitter", time.Second); !ok {
		t.Fatal("A : entête de chat non reçu")
	}

	// Client B entre à son tour et envoie un message.
	connB, rB := enterAsGuest(t, addr)
	defer connB.Close()
	connB.Write([]byte("2"))
	if _, ok := readFor(t, rB, connB, "quitter", time.Second); !ok {
		t.Fatal("B : entête de chat non reçu")
	}

	// A doit voir l'arrivée de B (message système).
	if _, ok := readFor(t, rA, connA, "rejoint", 2*time.Second); !ok {
		t.Fatal("A n'a pas vu l'arrivée de B")
	}

	// B envoie une ligne ; A doit la recevoir.
	connB.Write([]byte("bonjour les Oriciens\r"))
	if out, ok := readFor(t, rA, connA, "bonjour les Oriciens", 2*time.Second); !ok {
		t.Fatalf("A n'a pas reçu le message de B ; vu : %q", out)
	}

	// Sortie propre des deux côtés.
	connA.Write([]byte("/q\r"))
	connB.Write([]byte("/q\r"))
}
