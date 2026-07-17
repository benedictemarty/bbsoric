package bbs

import (
	"strings"
	"testing"
	"time"
)

// connect4SiteJSON : accueil -> invité -> menu (Puissance 4).
const connect4SiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "Jeu", "applet": "connect4", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// TestConnect4PlayAndComputerResponds : le joueur dépose un jeton en colonne 1
// (O en bas à gauche) ; sur plateau quasi vide l'ordinateur joue le centre
// (colonne 4, X), résultat déterministe. On quitte ensuite proprement.
func TestConnect4PlayAndComputerResponds(t *testing.T) {
	addr, _, stop := startBBSPresence(t, connect4SiteJSON)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("1")) // lance Puissance 4
	if out, ok := readFor(t, r, conn, "1 2 3 4 5 6 7", time.Second); !ok {
		t.Fatalf("plateau non affiché ; vu : %q", out)
	}
	conn.Write([]byte("1")) // dépose en colonne 1 (index 0)
	// Sur plateau quasi vide, l'ordinateur joue le centre (colonne 4).
	out, ok := readFor(t, r, conn, "Ordi a joue colonne 4", 2*time.Second)
	if !ok {
		t.Fatalf("réponse de l'ordinateur (centre) non observée ; vu :\n%s", out)
	}
	if strings.Contains(out, "GAGNE") || strings.Contains(out, "PERDU") {
		t.Errorf("partie ne devrait pas être finie après 1 coup ; vu : %q", out)
	}
	conn.Write([]byte("Q")) // quitte le jeu -> retour menu
	if _, ok := readFor(t, r, conn, "Votre choix", time.Second); !ok {
		t.Fatal("retour au menu attendu après Q")
	}
}
