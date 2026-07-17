package bbs

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/pm"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// pmSiteJSON : accueil -> (login | invité) -> menu (messagerie).
const pmSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Connexion", "target": "login" },
      { "key": "3", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "login": { "applet": "login", "next": "main" },
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Messagerie", "applet": "pm", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// startBBSPM démarre un BBS avec comptes ET messagerie privée.
func startBBSPM(t *testing.T, users *user.Store, mailbox *pm.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, pmSiteJSON), Users: users, PM: mailbox},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// TestPMReadAndReply : Bob se connecte, lit un message reçu d'Alice (qui passe
// « lu »), puis y répond. On vérifie l'écran et l'état persisté.
func TestPMReadAndReply(t *testing.T) {
	users, _ := user.Open("")
	if _, err := users.Register("Bob", "pw1234"); err != nil {
		t.Fatalf("register Bob: %v", err)
	}
	if _, err := users.Register("Alice", "pw1234"); err != nil {
		t.Fatalf("register Alice: %v", err)
	}
	mailbox, _ := pm.Open("")
	if _, err := mailbox.Send("Alice", "Bob", "coucou Bob"); err != nil {
		t.Fatalf("seed message: %v", err)
	}

	addr, stop := startBBSPM(t, users, mailbox)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	// Connexion de Bob.
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1")) // -> page login (applet)
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("Bob\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("pw1234\r"))
	readUntil(t, r, conn, "Bonjour")
	conn.Write([]byte(" ")) // pause -> menu principal
	readUntil(t, r, conn, "Votre choix")

	// Messagerie : 1 message non lu.
	conn.Write([]byte("1"))
	if out, ok := readFor(t, r, conn, "non lu", time.Second); !ok {
		t.Fatalf("compteur de non-lus non affiché ; vu : %q", out)
	}
	// Ouvre le message (touche 1) : contenu affiché, expéditeur Alice.
	conn.Write([]byte("1"))
	if out, ok := readFor(t, r, conn, "coucou Bob", time.Second); !ok {
		t.Fatalf("corps du message non affiché ; vu : %q", out)
	}
	// Répond à Alice.
	conn.Write([]byte("R"))
	if _, ok := readFor(t, r, conn, "Message", time.Second); !ok {
		t.Fatal("invite de message non reçue")
	}
	conn.Write([]byte("merci Alice\r"))
	if out, ok := readFor(t, r, conn, "envoye a Alice", 2*time.Second); !ok {
		t.Fatalf("confirmation d'envoi absente ; vu : %q", out)
	}

	// État persisté : le message d'Alice est lu ; Alice a reçu la réponse de Bob.
	if mailbox.Unread("Bob") != 0 {
		t.Errorf("le message de Bob devrait être lu ; non-lus = %d", mailbox.Unread("Bob"))
	}
	aliceBox := mailbox.Inbox("Alice")
	if len(aliceBox) != 1 || aliceBox[0].Text != "merci Alice" || aliceBox[0].From != "Bob" {
		t.Fatalf("réponse non persistée pour Alice : %+v", aliceBox)
	}
}

// TestPMRequiresAccount : un invité ne peut pas accéder à la messagerie.
func TestPMRequiresAccount(t *testing.T) {
	users, _ := user.Open("")
	mailbox, _ := pm.Open("")
	addr, stop := startBBSPM(t, users, mailbox)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("3")) // invité
	readUntil(t, r, conn, "touche")
	conn.Write([]byte(" "))
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1")) // messagerie
	if out, ok := readFor(t, r, conn, "Reserve aux membres", time.Second); !ok {
		t.Fatalf("gating membre attendu ; vu : %q", out)
	}
}
