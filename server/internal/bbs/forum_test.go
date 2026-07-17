package bbs

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/forum"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// forumSiteJSON : accueil -> invité -> menu (forum).
const forumSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Forum", "applet": "forum", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// startBBSForum démarre un BBS avec un store de forum partagé (en mémoire).
func startBBSForum(t *testing.T, f *forum.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, forumSiteJSON), Forum: f},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

// TestForumCreateReadReply : un invité ouvre le forum vide, crée un fil, le lit,
// puis y répond. On vérifie l'écran à chaque étape et l'état persisté.
func TestForumCreateReadReply(t *testing.T) {
	f, _ := forum.Open("")
	addr, stop := startBBSForum(t, f)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	// Ouvre le forum (vide).
	conn.Write([]byte("1"))
	if _, ok := readFor(t, r, conn, "Aucun fil", time.Second); !ok {
		t.Fatal("écran de forum vide non reçu")
	}
	// Crée un fil : N puis titre puis message.
	conn.Write([]byte("N"))
	if _, ok := readFor(t, r, conn, "Titre", time.Second); !ok {
		t.Fatal("invite de titre non reçue")
	}
	conn.Write([]byte("Mon premier sujet\r"))
	if _, ok := readFor(t, r, conn, "Message", time.Second); !ok {
		t.Fatal("invite de message non reçue")
	}
	conn.Write([]byte("Salut le forum Oric\r"))
	// Retour à la liste : le fil apparaît.
	if out, ok := readFor(t, r, conn, "Mon premier sujet", 2*time.Second); !ok {
		t.Fatalf("fil absent de la liste ; vu : %q", out)
	}
	// Ouvre le fil (touche 1) : on voit le message.
	conn.Write([]byte("1"))
	if out, ok := readFor(t, r, conn, "Salut le forum Oric", time.Second); !ok {
		t.Fatalf("message du fil non affiché ; vu : %q", out)
	}
	// Répond.
	conn.Write([]byte("R"))
	if _, ok := readFor(t, r, conn, "Reponse", time.Second); !ok {
		t.Fatal("invite de réponse non reçue")
	}
	conn.Write([]byte("Bien recu !\r"))
	if out, ok := readFor(t, r, conn, "Bien recu", 2*time.Second); !ok {
		t.Fatalf("réponse non affichée ; vu : %q", out)
	}

	// État persisté : 1 fil, 2 messages, second posté par un invité.
	if f.Count() != 1 {
		t.Fatalf("Count = %d, veut 1", f.Count())
	}
	list := f.List()
	full, ok := f.Thread(list[0].ID)
	if !ok || len(full.Posts) != 2 {
		t.Fatalf("fil attendu avec 2 messages ; vu : %+v", full)
	}
	if full.Posts[0].Text != "Salut le forum Oric" || full.Posts[1].Text != "Bien recu !" {
		t.Errorf("contenu des messages inattendu : %+v", full.Posts)
	}
}
