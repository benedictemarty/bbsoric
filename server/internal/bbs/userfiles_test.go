package bbs

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/userfiles"
)

// startBBSUserFiles démarre un serveur BBS avec un espace fichiers personnel.
func startBBSUserFiles(t *testing.T, json string, uf *userfiles.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, json), UserFiles: uf},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

const mesFichiersSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "ACCUEIL", "entries": [
      { "key": "1", "label": "Mes fichiers", "applet": "mesfichiers", "next": "accueil" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// readUntilContains lit la connexion jusqu'à voir sub (ou timeout).
func readUntilContains(t *testing.T, c net.Conn, sub string) bool {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var acc []byte
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			if bytes.Contains(acc, []byte(sub)) {
				return true
			}
		}
		if err != nil {
			return bytes.Contains(acc, []byte(sub))
		}
	}
}

// Un utilisateur NON identifié (invité) qui ouvre « Mes fichiers » est refusé :
// l'espace personnel est réservé aux comptes (cf. ADR-0006).
func TestMesFichiersRefuseNonIdentifie(t *testing.T) {
	uf, err := userfiles.Open(t.TempDir(), 20, 512*1024, 64*1024)
	if err != nil {
		t.Fatalf("userfiles.Open : %v", err)
	}
	addr, stop := startBBSUserFiles(t, mesFichiersSiteJSON, uf)
	defer stop()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial : %v", err)
	}
	defer c.Close()

	// Bannière + menu accueil affichés ; on choisit « 1 » (Mes fichiers).
	if !readUntilContains(t, c, "ACCUEIL") {
		t.Fatal("menu d'accueil non reçu")
	}
	if _, err := c.Write([]byte("1")); err != nil {
		t.Fatalf("write : %v", err)
	}
	if !readUntilContains(t, c, "Reserve aux membres identifies") {
		t.Fatal("l'espace personnel aurait dû refuser un invité")
	}
}
