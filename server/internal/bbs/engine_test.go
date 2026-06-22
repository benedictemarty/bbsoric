package bbs

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// startServerWithStore démarre un BBS avec un Store de contenu donné.
func startServerWithStore(t *testing.T, store *content.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: store}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// storeFromJSON écrit le JSON dans un fichier temporaire et construit un Store.
func storeFromJSON(t *testing.T, json string) *content.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "site.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatalf("write site.json: %v", err)
	}
	return content.NewStore(path, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

const appletSiteJSON = `{
  "start": "start",
  "pages": {
    "start": { "type": "menu", "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Se connecter", "target": "login" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "login": { "type": "applet", "applet": "testlogin", "next": "main" },
    "main": { "type": "menu", "title": "MENU PRINCIPAL", "entries": [
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// TestAppletDispatchAndNext : une page de type applet déclenche l'applet
// enregistré et, en cas de succès, navigue vers sa page "next".
func TestAppletDispatchAndNext(t *testing.T) {
	ran := make(chan *AppContext, 1)
	Register("testlogin", func(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
		ran <- ac
		_ = s.Write("APPLET-OK ")
		ac.State.Guest = true
		return Outcome{Done: true}
	})

	addr, stop := startServerWithStore(t, storeFromJSON(t, appletSiteJSON))
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix") // menu d'accueil
	if _, err := conn.Write([]byte("1")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// L'applet doit avoir été appelé avec un AppContext utilisable.
	select {
	case ac := <-ran:
		if ac == nil || ac.State == nil || ac.Page == nil {
			t.Fatalf("AppContext incomplet: %+v", ac)
		}
		if ac.Page.Applet != "testlogin" {
			t.Errorf("page applet inattendue: %q", ac.Page.Applet)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("l'applet n'a pas été appelé")
	}

	// Sortie de l'applet puis navigation vers "next" (MENU PRINCIPAL).
	out := readUntil(t, r, conn, "MENU PRINCIPAL")
	if !strings.Contains(out, "APPLET-OK") {
		t.Errorf("sortie de l'applet absente:\n%s", out)
	}
	if !strings.Contains(out, "MENU PRINCIPAL") {
		t.Errorf("navigation vers 'next' échouée:\n%s", out)
	}
}

// TestMenuEntryApplet : une ENTRÉE de menu peut lancer un applet (au lieu de
// naviguer) ; un même menu peut donc proposer plusieurs applets au choix.
func TestMenuEntryApplet(t *testing.T) {
	Register("entrylogin", func(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
		_ = s.Write("ENTRY-OK ")
		return Outcome{Done: true}
	})
	const json = `{
      "start": "accueil",
      "pages": {
        "accueil": { "type": "menu", "title": "BIENVENUE", "entries": [
          { "key": "1", "label": "Se connecter", "applet": "entrylogin", "next": "main" },
          { "key": "Q", "label": "Quitter", "target": "__quit__" }
        ]},
        "main": { "type": "menu", "title": "MENU PRINCIPAL", "entries": [
          { "key": "Q", "label": "Quitter", "target": "__quit__" }
        ]}
      }
    }`
	addr, stop := startServerWithStore(t, storeFromJSON(t, json))
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	if _, err := conn.Write([]byte("1")); err != nil { // lance l'applet de l'entrée
		t.Fatalf("write: %v", err)
	}
	out := readUntil(t, r, conn, "MENU PRINCIPAL")
	if !strings.Contains(out, "ENTRY-OK") {
		t.Errorf("l'applet de l'entrée n'a pas tourné:\n%s", out)
	}
	if !strings.Contains(out, "MENU PRINCIPAL") {
		t.Errorf("navigation vers 'next' après l'applet échouée:\n%s", out)
	}
}

// TestUnknownAppletIsGraceful : un applet non enregistré ne casse pas la session.
func TestUnknownAppletIsGraceful(t *testing.T) {
	const json = `{
      "start": "start",
      "pages": {
        "start": { "type": "menu", "title": "ACCUEIL", "entries": [
          { "key": "1", "label": "Jeu", "target": "game" }
        ]},
        "game": { "type": "applet", "applet": "inexistant", "next": "start" }
      }
    }`
	addr, stop := startServerWithStore(t, storeFromJSON(t, json))
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	if _, err := conn.Write([]byte("1")); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := readUntil(t, r, conn, "indisponible")
	if !strings.Contains(out, "indisponible") {
		t.Errorf("message d'applet manquant attendu, reçu:\n%s", out)
	}
}
