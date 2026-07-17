package bbs

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/wall"
)

// wallSiteJSON : accueil -> invité -> menu (mur de messages).
const wallSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Mur", "target": "guestbook" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "guestbook": { "title": "MUR DE MESSAGES", "applet": "wall" }
  }
}`

// startBBSWall démarre un BBS avec un store de mur partagé (en mémoire).
func startBBSWall(t *testing.T, w *wall.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, wallSiteJSON), Wall: w},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// TestWallPostAndPersist : un invité poste un message ; le store le conserve et
// il réapparaît à l'écran (rebouclage de l'applet), pseudo invité inclus.
func TestWallPostAndPersist(t *testing.T) {
	w, err := wall.Open("") // mémoire
	if err != nil {
		t.Fatalf("wall.Open: %v", err)
	}
	addr, stop := startBBSWall(t, w)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("1")) // « Mur de messages »
	if _, ok := readFor(t, r, conn, "Mur vide", time.Second); !ok {
		t.Fatal("écran de mur vide non reçu")
	}
	// Poste un message.
	conn.Write([]byte("salut les Oriciens\r"))
	// L'applet reboucle et réaffiche le mur avec le message.
	if out, ok := readFor(t, r, conn, "salut les Oriciens", 2*time.Second); !ok {
		t.Fatalf("message non réaffiché ; vu : %q", out)
	}

	// Le store doit contenir exactement ce message, posté par un invité.
	if w.Count() != 1 {
		t.Fatalf("Count = %d, veut 1", w.Count())
	}
	m := w.List(1)[0]
	if m.Text != "salut les Oriciens" {
		t.Errorf("texte persisté = %q", m.Text)
	}
	if !strings.HasPrefix(m.Handle, "Invite-") {
		t.Errorf("handle persisté = %q, veut un pseudo invité", m.Handle)
	}
}

// TestWallEmptyReturns : un message vide (RETURN seul) quitte l'applet sans rien
// poster (retour au menu appelant).
func TestWallEmptyReturns(t *testing.T) {
	w, _ := wall.Open("")
	addr, stop := startBBSWall(t, w)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("1")) // Mur
	if _, ok := readFor(t, r, conn, "vide=retour", time.Second); !ok {
		t.Fatal("invite du mur non reçue")
	}
	conn.Write([]byte("\r")) // message vide -> retour au menu
	if _, ok := readFor(t, r, conn, "Votre choix", time.Second); !ok {
		t.Fatal("retour au menu attendu après message vide")
	}
	if w.Count() != 0 {
		t.Errorf("Count = %d, veut 0 (rien posté)", w.Count())
	}
}
