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

	"github.com/benedictemarty/bbsoric/server/internal/news"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// newsSiteJSON : accueil -> (login | invité) -> menu (actualités).
const newsSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Connexion", "target": "login" },
      { "key": "3", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "login": { "applet": "login", "next": "main" },
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Actualites", "applet": "news", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

func startBBSNews(t *testing.T, users *user.Store, store *news.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, newsSiteJSON), Users: users, News: store},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// TestNewsGuestReadsNoPublish : un invité voit les annonces mais n'a PAS la
// touche de publication (N réservée à l'admin).
func TestNewsGuestReadsNoPublish(t *testing.T) {
	users, _ := user.Open("")
	store, _ := news.Open("")
	if _, err := store.Post("sysop", "Bienvenue", "Le BBS Oric est en ligne"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	addr, stop := startBBSNews(t, users, store)
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
	conn.Write([]byte("1")) // Actualites
	out, ok := readFor(t, r, conn, "Le BBS Oric est en ligne", time.Second)
	if !ok {
		t.Fatalf("annonce non affichée ; vu : %q", out)
	}
	if strings.Contains(out, "=publier") {
		t.Errorf("un invité ne doit pas voir la touche de publication ; vu : %q", out)
	}
}

// TestNewsAdminPublishes : le premier compte (admin) peut publier une annonce
// qui est persistée.
func TestNewsAdminPublishes(t *testing.T) {
	users, _ := user.Open("")
	if _, err := users.Register("Sysop", "pw1234"); err != nil { // 1er compte = admin
		t.Fatalf("register: %v", err)
	}
	store, _ := news.Open("")
	addr, stop := startBBSNews(t, users, store)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1")) // connexion
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("Sysop\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("pw1234\r"))
	readUntil(t, r, conn, "Bonjour")
	conn.Write([]byte(" "))
	readUntil(t, r, conn, "Votre choix")

	conn.Write([]byte("1")) // Actualites
	if out, ok := readFor(t, r, conn, "=publier", time.Second); !ok {
		t.Fatalf("admin doit voir la touche publier ; vu : %q", out)
	}
	conn.Write([]byte("N"))
	if _, ok := readFor(t, r, conn, "Titre", time.Second); !ok {
		t.Fatal("invite de titre non reçue")
	}
	conn.Write([]byte("Maintenance\r"))
	if _, ok := readFor(t, r, conn, "Texte", time.Second); !ok {
		t.Fatal("invite de texte non reçue")
	}
	conn.Write([]byte("Coupure prevue dimanche\r"))
	if out, ok := readFor(t, r, conn, "Coupure prevue dimanche", 2*time.Second); !ok {
		t.Fatalf("annonce non réaffichée ; vu : %q", out)
	}
	if store.Count() != 1 {
		t.Fatalf("Count = %d, veut 1", store.Count())
	}
	it := store.List(1)[0]
	if it.Title != "Maintenance" || it.Author != "Sysop" {
		t.Errorf("annonce persistée inattendue : %+v", it)
	}
}
