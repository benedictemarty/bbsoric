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

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// startServerFull démarre un BBS avec un Store de contenu et un Store de comptes.
func startServerFull(t *testing.T, store *content.Store, users *user.Store) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: store, Users: users}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); wg.Wait() }
}

const authSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "type": "menu", "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Se connecter", "target": "login" },
      { "key": "2", "label": "Creer un compte", "target": "register" },
      { "key": "3", "label": "Invite", "target": "guest" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "login": { "type": "applet", "applet": "login", "next": "main" },
    "register": { "type": "applet", "applet": "register", "next": "main" },
    "guest": { "type": "applet", "applet": "guest", "next": "main" },
    "main": { "type": "menu", "title": "MENU PRINCIPAL", "entries": [
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// dialAuth ouvre une connexion. L'appelant doit `defer conn.Close()` APRÈS son
// `defer stop()` : l'ordre LIFO ferme la connexion d'abord, ce qui débloque la
// session côté serveur et évite que stop() attende le timeout d'inactivité.
func dialAuth(t *testing.T, addr string) (*bufio.Reader, net.Conn) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return bufio.NewReader(conn), conn
}

// TestFormPageLogin : une page de saisie déclarative (content.Form, action
// "login") tient lieu d'écran de connexion — sans applet Go dédié.
func TestFormPageLogin(t *testing.T) {
	users, _ := user.Open("")
	if _, err := users.Register("Bob", "pw1234"); err != nil {
		t.Fatalf("register fixture: %v", err)
	}
	const json = `{
      "start": "login",
      "pages": {
        "login": { "title": "CONNEXION",
          "form": { "action": "login", "next": "main", "fields": [
            { "key": "login", "label": "Pseudo" },
            { "key": "password", "label": "Mot de passe", "secret": true }
          ] } },
        "main": { "title": "MENU PRINCIPAL", "entries": [
          { "key": "Q", "label": "Quitter", "target": "__quit__" }
        ] }
      }
    }`
	addr, stop := startServerFull(t, storeFromJSON(t, json), users)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("bob\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("pw1234\r"))
	out := readUntil(t, r, conn, "Bonjour")
	if !strings.Contains(out, "Bonjour Bob") {
		t.Errorf("accueil personnalisé attendu:\n%s", out)
	}
	conn.Write([]byte(" ")) // pause -> next
	main := readUntil(t, r, conn, "MENU PRINCIPAL")
	if !strings.Contains(main, "MENU PRINCIPAL") {
		t.Errorf("navigation vers form.next échouée:\n%s", main)
	}
}

// TestFormPageRegister : page de saisie déclarative, action "register".
func TestFormPageRegister(t *testing.T) {
	users, _ := user.Open("")
	const json = `{
      "start": "signup",
      "pages": {
        "signup": { "title": "INSCRIPTION",
          "form": { "action": "register", "next": "main", "fields": [
            { "key": "login", "label": "Pseudo" },
            { "key": "password", "label": "Mot de passe", "secret": true },
            { "key": "confirm", "label": "Confirmer", "secret": true }
          ] } },
        "main": { "title": "MENU PRINCIPAL", "entries": [
          { "key": "Q", "label": "Quitter", "target": "__quit__" }
        ] }
      }
    }`
	addr, stop := startServerFull(t, storeFromJSON(t, json), users)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("Alice\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("secret1\r"))
	readUntil(t, r, conn, "Confirmer")
	conn.Write([]byte("secret1\r"))
	out := readUntil(t, r, conn, "Bienvenue")
	if !strings.Contains(out, "Compte cree") {
		t.Errorf("confirmation de création attendue:\n%s", out)
	}
	if _, ok := users.Get("alice"); !ok {
		t.Errorf("le compte Alice doit être persisté")
	}
}

func TestLoginAppletSuccess(t *testing.T) {
	users, _ := user.Open("")
	if _, err := users.Register("Bob", "pw1234"); err != nil {
		t.Fatalf("register fixture: %v", err)
	}
	addr, stop := startServerFull(t, storeFromJSON(t, authSiteJSON), users)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Votre choix") // menu BIENVENUE
	conn.Write([]byte("1"))              // -> applet login
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("bob\r")) // pseudo (insensible a la casse)
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("pw1234\r"))
	out := readUntil(t, r, conn, "Bonjour")
	if !strings.Contains(out, "Bonjour Bob") {
		t.Errorf("accueil personnalise attendu, recu:\n%s", out)
	}
	if !strings.Contains(out, "Appel n1") {
		t.Errorf("numero d'appel attendu, recu:\n%s", out)
	}
	conn.Write([]byte(" ")) // pause -> navigue vers main
	main := readUntil(t, r, conn, "MENU PRINCIPAL")
	if !strings.Contains(main, "MENU PRINCIPAL") {
		t.Errorf("navigation vers main echouee:\n%s", main)
	}
}

func TestLoginAppletWrongPassword(t *testing.T) {
	users, _ := user.Open("")
	users.Register("Bob", "pw1234")
	addr, stop := startServerFull(t, storeFromJSON(t, authSiteJSON), users)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("1"))
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("bob\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("mauvais\r"))
	out := readUntil(t, r, conn, "Echec")
	if !strings.Contains(out, "Echec") {
		t.Errorf("message d'echec attendu, recu:\n%s", out)
	}
}

func TestGuestApplet(t *testing.T) {
	addr, stop := startServerFull(t, storeFromJSON(t, authSiteJSON), nil)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("3")) // invite
	out := readUntil(t, r, conn, "lecture seule")
	if !strings.Contains(out, "ACCES INVITE") {
		t.Errorf("ecran invite attendu, recu:\n%s", out)
	}
	conn.Write([]byte(" ")) // pause -> main
	main := readUntil(t, r, conn, "MENU PRINCIPAL")
	if !strings.Contains(main, "MENU PRINCIPAL") {
		t.Errorf("invite non dirige vers main:\n%s", main)
	}
}

func TestRegisterApplet(t *testing.T) {
	users, _ := user.Open("")
	addr, stop := startServerFull(t, storeFromJSON(t, authSiteJSON), users)
	defer stop()

	r, conn := dialAuth(t, addr)
	defer conn.Close()
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("2")) // creation de compte
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("Alice\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("secret1\r"))
	readUntil(t, r, conn, "Confirmer")
	conn.Write([]byte("secret1\r"))
	out := readUntil(t, r, conn, "Bienvenue")
	if !strings.Contains(out, "Compte cree") {
		t.Errorf("confirmation de creation attendue, recu:\n%s", out)
	}
	if _, ok := users.Get("alice"); !ok {
		t.Errorf("le compte Alice doit etre persiste dans le store")
	}
}
