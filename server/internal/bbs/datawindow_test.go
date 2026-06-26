package bbs

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/datawindow"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// dwSiteJSON : accueil (invité + grille) → grille DataWindow éditable.
const dwSiteJSON = `{
  "start": "accueil",
  "sources_donnees": {
    "rep": {
      "table": "rep",
      "tri_defaut": "nom ASC",
      "lignes_par_page": 10,
      "colonnes": {
        "id":    {"type":"INTEGER","libelle":"ID","cle_primaire":true,"auto_increment":true},
        "nom":   {"type":"TEXT","libelle":"Nom","requis":true,"longueur_max":16},
        "ville": {"type":"TEXT","libelle":"Ville","longueur_max":10},
        "note":  {"type":"INTEGER","libelle":"Note"}
      },
      "donnees": [
        {"nom":"Alice","ville":"Lyon","note":5},
        {"nom":"Bob","ville":"Paris","note":3}
      ]
    }
  },
  "pages": {
    "accueil": {"title":"BIENVENUE","entries":[
      {"key":"1","label":"Invite","applet":"guest","next":"accueil"},
      {"key":"2","label":"Repertoire","target":"grille"},
      {"key":"Q","label":"Quitter","target":"__quit__"}
    ]},
    "grille": {"title":"REPERTOIRE","datawindow":{
      "source":"rep",
      "colonnes_affichees":["nom","ville","note"],
      "largeurs":[16,10,3],
      "editable":true
    }}
  }
}`

// startBBSData démarre un BBS avec un moteur DataWindow (base SQLite temporaire)
// et les sources du site initialisées.
func startBBSData(t *testing.T, json string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	store := storeFromJSON(t, json)
	eng := datawindow.NewEngine(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, src := range store.Site().SourcesDonnees {
		if err := eng.InitialiserSource(src); err != nil {
			t.Fatalf("init source: %v", err)
		}
	}
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: store, Data: eng},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); eng.Close(); wg.Wait() }
}

func TestDataWindowGrille(t *testing.T) {
	addr, stop := startBBSData(t, dwSiteJSON)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2")) // -> grille
	out, ok := readFor(t, r, conn, "Page 1/1", time.Second)
	if !ok {
		t.Fatalf("grille non affichée ; vu : %q", out)
	}
	if !contains(out, "Nom") || !contains(out, "Ville") {
		t.Errorf("entête absente ; vu : %q", out)
	}
	if !contains(out, "2 enreg") {
		t.Errorf("total attendu 2 ; vu : %q", out)
	}
	if !contains(out, "F=filtre") {
		t.Errorf("légende absente ; vu : %q", out)
	}
}

func TestDataWindowFiltre(t *testing.T) {
	addr, stop := startBBSData(t, dwSiteJSON)
	defer stop()
	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2"))
	readFor(t, r, conn, "Page 1/1", time.Second)
	conn.Write([]byte("F")) // filtre
	readFor(t, r, conn, "Filtre", time.Second)
	conn.Write([]byte("Lyon\r"))
	out, ok := readFor(t, r, conn, "1 enreg", time.Second)
	if !ok {
		t.Fatalf("filtre non appliqué ; vu : %q", out)
	}
	// Alice (Lyon) reste ; mais elle est sélectionnée → inverse, donc on vérifie
	// plutôt que Bob (Paris) a disparu et que le total est tombé à 1.
	if contains(out, "Bob") {
		t.Errorf("Bob (Paris) aurait dû être filtré ; vu : %q", out)
	}
}

func TestDataWindowCreer(t *testing.T) {
	addr, stop := startBBSData(t, dwSiteJSON)
	defer stop()
	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2"))
	readFor(t, r, conn, "Page 1/1", time.Second)

	conn.Write([]byte("N")) // nouveau
	readFor(t, r, conn, "Nom", time.Second)
	conn.Write([]byte("Zoe\r"))
	readFor(t, r, conn, "Ville", time.Second)
	conn.Write([]byte("Tours\r"))
	readFor(t, r, conn, "Note", time.Second)
	conn.Write([]byte("9\r"))
	out, ok := readFor(t, r, conn, "cree", time.Second)
	if !ok {
		t.Fatalf("création non confirmée ; vu : %q", out)
	}
	conn.Write([]byte(" ")) // touche pour revenir à la grille
	out, ok = readFor(t, r, conn, "3 enreg", time.Second)
	if !ok {
		t.Fatalf("total attendu 3 après création ; vu : %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
