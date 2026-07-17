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

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/xmodem"
	"github.com/benedictemarty/bbsoric/server/internal/datawindow"
	"github.com/benedictemarty/bbsoric/server/internal/files"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
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

// dwAuthSiteJSON : comme dwSiteJSON, plus une entrée « Connexion » (3) — pour
// tester l'écriture DataWindow réservée aux comptes admin (S11.5).
const dwAuthSiteJSON = `{
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
      {"key":"3","label":"Connexion","applet":"login","next":"accueil"},
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

// startBBSDataAuth démarre un BBS DataWindow AVEC un magasin de comptes, pour
// piloter les tests d'écriture (admin). Renvoie le store pour préinscrire un compte.
func startBBSDataAuth(t *testing.T, json string) (addr string, users *user.Store, stop func()) {
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
	users, _ = user.Open("")
	cfg := server.Config{Addr: ln.Addr().String(), IdleTimeout: 30 * time.Second}
	srv := server.New(cfg, WelcomeHandler{Store: store, Data: eng, Users: users},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), users, func() { cancel(); _ = ln.Close(); eng.Close(); wg.Wait() }
}

// catSiteJSON : un catalogue (source "cat") dont une colonne porte le nom du
// fichier téléchargeable ; la grille l'expose via fichier_colonne (touche X).
const catSiteJSON = `{
  "start": "accueil",
  "sources_donnees": {
    "cat": {
      "table": "cat",
      "tri_defaut": "titre ASC",
      "lignes_par_page": 10,
      "colonnes": {
        "id":      {"type":"INTEGER","libelle":"ID","cle_primaire":true,"auto_increment":true},
        "titre":   {"type":"TEXT","libelle":"Titre","longueur_max":20},
        "fichier": {"type":"TEXT","libelle":"Fichier","longueur_max":16}
      },
      "donnees": [ {"titre":"Demo","fichier":"demo.tap"} ]
    }
  },
  "pages": {
    "accueil": {"title":"BIENVENUE","entries":[
      {"key":"1","label":"Invite","applet":"guest","next":"accueil"},
      {"key":"2","label":"Catalogue","target":"grille"},
      {"key":"Q","label":"Quitter","target":"__quit__"}
    ]},
    "grille": {"title":"CATALOGUE","datawindow":{
      "source":"cat",
      "colonnes_affichees":["titre"],
      "largeurs":[20],
      "fichier_colonne":"fichier"
    }}
  }
}`

// startBBSDataCatalogue démarre un BBS DataWindow AVEC une bibliothèque de fichiers.
func startBBSDataCatalogue(t *testing.T, json string, lib *files.Library) (addr string, stop func()) {
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
	srv := server.New(cfg, WelcomeHandler{Store: store, Data: eng, Files: lib},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); srv.Serve(ctx, ln) }()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close(); eng.Close(); wg.Wait() }
}

// TestDataWindowDownloadFromRow : la touche X d'une grille catalogue télécharge le
// fichier nommé dans fichier_colonne de la ligne sélectionnée, via XMODEM.
func TestDataWindowDownloadFromRow(t *testing.T) {
	lib, _ := files.Open(t.TempDir(), 0)
	want := []byte("Programme Oric de demonstration (petit .tap).")
	if err := lib.Write("demo.tap", want); err != nil {
		t.Fatal(err)
	}
	addr, stop := startBBSDataCatalogue(t, catSiteJSON, lib)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2")) // -> catalogue
	if out, ok := readFor(t, r, conn, "F/T Q", time.Second); !ok || !contains(out, "X=DL") {
		t.Fatalf("légende de téléchargement absente ; vu : %q", out)
	}
	conn.Write([]byte("X")) // télécharge la ligne sélectionnée (Demo -> demo.tap)
	if out, ok := readFor(t, r, conn, "terminal.", 2*time.Second); !ok {
		t.Fatalf("invite de réception absente ; vu : %q", out)
	}
	got, err := xmodem.Receive(&clientConn{r: r, c: conn})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("téléchargé %q, attendu %q", got, want)
	}
}

// TestWrapValeur : la fiche détail replie une valeur longue (au lieu de tronquer)
// et marque la troncature au-delà de maxLines (J4).
func TestWrapValeur(t *testing.T) {
	if got := wrapValeur("", 22, 4); len(got) != 1 || got[0] != "" {
		t.Errorf("vide -> une ligne vide, got %v", got)
	}
	if got := wrapValeur("court", 22, 4); len(got) != 1 || got[0] != "court" {
		t.Errorf("court -> une ligne, got %v", got)
	}
	long := "AAAAAAAAAA"    // 10
	v := long + long + long // 30 -> 2 lignes de 22 + reste
	got := wrapValeur(v, 22, 4)
	if len(got) != 2 || len(got[0]) != 22 {
		t.Errorf("repli: got %v (len0=%d)", got, len(got[0]))
	}
	// Troncature : beaucoup de contenu, cap 2 lignes -> dernière finit par "..."
	g2 := wrapValeur(bytesRepeat('X', 200), 22, 2)
	if len(g2) != 2 || g2[1][len(g2[1])-3:] != "..." {
		t.Errorf("troncature attendue avec ... ; got %v", g2)
	}
}

func bytesRepeat(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

// TestRenderGridPerPageNumbering : la colonne « No » numérote PAR PAGE (1..parPage),
// pas en absolu — sinon, sur une grande table, un index ≥ 100 collerait le titre
// (colonne de 3) et ≥ 1000 serait tronqué.
func TestRenderGridPerPageNumbering(t *testing.T) {
	scr := oascii.NewScreen()
	dw := &content.DataWindow{ColonnesAffichees: []string{"nom"}, Largeurs: []int{20}}
	src := content.SourceDonnees{Colonnes: map[string]content.ColonneDef{"nom": {Type: "TEXT"}}}
	rows := []map[string]string{{"nom": "X"}, {"nom": "Y"}, {"nom": "Z"}}
	// Page 2, 10 par page : un index absolu vaudrait 11,12,13 ; par page = 1,2,3.
	renderGrid(scr, dw, src, rows, -1, 2, 10, 23, "", "", false, false)
	c1 := scr.At(gridContentX, gridDataTop) & 0x7F   // 1er caractère du « No »
	c2 := scr.At(gridContentX+1, gridDataTop) & 0x7F // 2e (espace si « 1  », chiffre si « 11 »)
	if c1 != '1' || c2 != ' ' {
		t.Errorf("numérotation par page attendue (« 1  »), lu %q%q (index absolu ?)", string(rune(c1)), string(rune(c2)))
	}
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
	if !contains(out, "V=fiche") {
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
	addr, users, stop := startBBSDataAuth(t, dwAuthSiteJSON)
	defer stop()
	// 1er compte enregistré => admin (S11.5) ; seul un admin peut écrire.
	if _, err := users.Register("Sysop", "pw1234"); err != nil {
		t.Fatalf("register admin: %v", err)
	}

	r, conn := dialAuth(t, addr)
	defer conn.Close()

	// Connexion en admin depuis le menu d'accueil.
	readUntil(t, r, conn, "Votre choix")
	conn.Write([]byte("3"))
	readUntil(t, r, conn, "Pseudo")
	conn.Write([]byte("Sysop\r"))
	readUntil(t, r, conn, "Mot de passe")
	conn.Write([]byte("pw1234\r"))
	readUntil(t, r, conn, "Bonjour")
	conn.Write([]byte(" ")) // pause -> retour accueil (login next=accueil)
	readUntil(t, r, conn, "Votre choix")

	conn.Write([]byte("2")) // grille
	readFor(t, r, conn, "Page 1/1", time.Second)

	conn.Write([]byte("N")) // nouveau (autorisé : admin)
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

// TestDataWindowGuestCannotCreate : un invité (non admin) n'a pas les touches
// d'écriture et « N » ne crée rien (régression S11.5 — l'écriture DataWindow
// était ouverte à tout utilisateur connecté, invité compris).
func TestDataWindowGuestCannotCreate(t *testing.T) {
	addr, stop := startBBSData(t, dwSiteJSON)
	defer stop()
	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2")) // grille
	// Lire tout l'écran jusqu'à la fin de la légende ("F/T Q").
	out, ok := readFor(t, r, conn, "F/T Q", time.Second)
	if !ok {
		t.Fatalf("légende non reçue ; vu : %q", out)
	}
	if !contains(out, "Page 1/1") {
		t.Errorf("grille non affichée ; vu : %q", out)
	}
	if contains(out, "N/E/D") {
		t.Errorf("un invité ne doit pas voir les touches d'écriture N/E/D ; vu : %q", out)
	}
	// Presser N ne doit rien créer : aucune confirmation « cree » ne doit arriver.
	conn.Write([]byte("N"))
	if out, ok := readFor(t, r, conn, "cree", 500*time.Millisecond); ok {
		t.Errorf("la création doit être interdite à un invité ; vu : %q", out)
	}
}

func TestDataWindowTri(t *testing.T) {
	addr, stop := startBBSData(t, dwSiteJSON)
	defer stop()
	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("2"))
	readFor(t, r, conn, "Page 1/1", time.Second)
	// 1er T : Nom ASC -> libellé "tri Nom+".
	conn.Write([]byte("T"))
	if out, ok := readFor(t, r, conn, "tri Nom+", time.Second); !ok {
		t.Fatalf("tri ASC non affiché ; vu : %q", out)
	}
	// 2e T : Nom DESC -> "tri Nom-" et Bob (Paris) passe avant Alice.
	conn.Write([]byte("T"))
	out, ok := readFor(t, r, conn, "tri Nom-", time.Second)
	if !ok {
		t.Fatalf("tri DESC non affiché ; vu : %q", out)
	}
	// En DESC, Bob est sélectionné (inverse) et Alice redevient lisible.
	if !contains(out, "Alice") {
		t.Errorf("Alice devrait être lisible en tri DESC ; vu : %q", out)
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
