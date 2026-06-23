package bbs

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/xmodem"
	"github.com/benedictemarty/bbsoric/server/internal/files"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// clientConn adapte (bufio.Reader + net.Conn) à xmodem.Conn côté client de test :
// lecture via le tampon (préserve les octets déjà lus par readUntil), écriture et
// échéance via la connexion.
type clientConn struct {
	r *bufio.Reader
	c net.Conn
}

func (cc *clientConn) Read(p []byte) (int, error)        { return cc.r.Read(p) }
func (cc *clientConn) Write(p []byte) (int, error)       { return cc.c.Write(p) }
func (cc *clientConn) SetReadDeadline(t time.Time) error { return cc.c.SetReadDeadline(t) }

// startBBSFiles démarre un serveur BBS (avec bibliothèque) sur un port éphémère.
func startBBSFiles(t *testing.T, json string, lib *files.Library) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: storeFromJSON(t, json), Files: lib},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

const xferSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Telecharger", "applet": "download", "next": "accueil" },
      { "key": "2", "label": "Televerser", "applet": "upload", "next": "accueil" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

func TestDownloadApplet(t *testing.T) {
	lib, _ := files.Open(t.TempDir(), 0)
	want := []byte("Bonjour Oric, ceci vient du BBS via XMODEM.")
	if err := lib.Write("hello.txt", want); err != nil {
		t.Fatal(err)
	}
	addr, stop := startBBSFiles(t, xferSiteJSON, lib)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1")) // -> applet download
	readUntil(t, r, conn, "annuler")
	conn.Write([]byte("1")) // choisit hello.txt
	readUntil(t, r, conn, "terminal.")

	got, err := xmodem.Receive(&clientConn{r: r, c: conn})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("téléchargé %q, attendu %q", got, want)
	}
}

func TestUploadApplet(t *testing.T) {
	lib, _ := files.Open(t.TempDir(), 0)
	addr, stop := startBBSFiles(t, xferSiteJSON, lib)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("2")) // -> applet upload
	readUntil(t, r, conn, "Nom du fichier")
	conn.Write([]byte("recu.bin\r"))
	readUntil(t, r, conn, "terminal.")

	data := []byte("Donnees televersees depuis le terminal.")
	if err := xmodem.Send(&clientConn{r: r, c: conn}, data); err != nil {
		t.Fatalf("Send: %v", err)
	}
	readUntil(t, r, conn, "Recu")

	got, err := lib.Read("recu.bin")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("stocké %q, attendu %q", got, data)
	}
}
