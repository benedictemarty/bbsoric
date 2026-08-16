package bbs

import (
	"strings"
	"testing"
	"time"
)

// netmazeSiteJSON : accueil -> invité -> menu (NetMaze en touche 1).
const netmazeSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Invite", "applet": "guest", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]},
    "main": { "title": "MENU", "entries": [
      { "key": "1", "label": "NetMaze", "applet": "netmaze", "next": "main" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// TestNetmazeIntegration vérifie l'applet de bout en bout sur un vrai serveur
// TCP : lancement depuis le menu, boucle temps réel qui diffuse des trames
// successives (les ticks avancent tout seuls), puis abandon propre par la touche
// '0' avec retour au menu. C'est le smoke-test « la logique tourne » — le rendu
// curseur OASCII n'est pas interprété par ce client de test (comme un telnet PC),
// mais le texte des trames (« NETMAZE », « tick N ») reste lisible.
func TestNetmazeIntegration(t *testing.T) {
	addr, _, stop := startBBSPresence(t, netmazeSiteJSON)
	defer stop()

	conn, r := enterAsGuest(t, addr)
	defer conn.Close()

	conn.Write([]byte("1")) // lance NetMaze

	// Première trame (écran d'accueil de l'applet).
	if out, ok := readFor(t, r, conn, "NETMAZE", time.Second); !ok {
		t.Fatalf("trame NetMaze non reçue ; vu : %q", out)
	}

	// La boucle temps réel doit produire des trames successives sans intervention :
	// on attend l'apparition d'un tick avancé, preuve que l'arène tourne toute seule.
	if out, ok := readFor(t, r, conn, "tick 3", 3*time.Second); !ok {
		t.Fatalf("les ticks n'avancent pas (boucle temps réel) ; vu :\n%s", out)
	}

	// Abandon : la touche '0' rend la main au menu (page Next = main).
	conn.Write([]byte("0"))
	if out, ok := readFor(t, r, conn, "Votre choix", 3*time.Second); !ok {
		t.Fatalf("retour au menu attendu après '0' ; vu :\n%s", out)
	} else if strings.Count(out, "tick") > 40 {
		t.Errorf("l'abandon n'a pas stoppé la boucle (trames toujours diffusées)")
	}
}
