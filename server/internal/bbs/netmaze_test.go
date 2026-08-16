package bbs

import (
	"strings"
	"testing"

	"github.com/benedictemarty/netmaze/arena"
	"github.com/benedictemarty/netmaze/engine"
)

// TestNetmazeEnregistre : l'applet est bien enregistré sous le nom "netmaze"
// (condition pour que l'entrée de menu du site passe la validation au démarrage).
func TestNetmazeEnregistre(t *testing.T) {
	if _, ok := lookupApplet("netmaze"); !ok {
		t.Fatal(`l'applet "netmaze" devrait être enregistré via init()`)
	}
}

// viewAt fabrique un GameState solo et une StateView positionnant le joueur.
func viewAt(x, y int, facing engine.Dir, contacts []engine.Contact) (*engine.GameState, *arena.StateView) {
	g := engine.NewSoloGame(1, 16, 16, 0.3, engine.Normal, 3)
	v := &arena.StateView{
		Tick: 7,
		Self: arena.SelfView{
			NodeID: 1, X: x, Y: y, Facing: facing,
			Energy: 60, Frags: 2, Alive: true,
		},
		Contacts: contacts,
	}
	return g, v
}

// viewCenter est l'indice (ligne/colonne) du joueur au centre de la vue fenêtrée.
const viewCenter = 2*netmazeViewRadius + 1

// TestRenderNetmazeContenu : la trame contient l'en-tête, le glyphe du joueur au
// centre de la vue tactique et les rubriques du HUD, sans paniquer.
func TestRenderNetmazeContenu(t *testing.T) {
	g, v := viewAt(8, 8, engine.East, nil)
	out := renderNetmaze(g, v)
	for _, want := range []string{"NETMAZE", "Energie", "Radar", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("la trame devrait contenir %q", want)
		}
	}
	rows := mazeWindow(g, v)
	if got := rows[viewCenter][viewCenter]; got != '>' {
		t.Errorf("le joueur tourné vers l'Est devrait s'afficher '>' au centre, obtenu %q", string(got))
	}
}

// TestMazeWindowMurs : la vue tactique dessine des murs (au moins un '|' ou '-'),
// preuve que le labyrinthe est bien rendu et non une simple grille de positions.
func TestMazeWindowMurs(t *testing.T) {
	g, v := viewAt(8, 8, engine.North, nil)
	out := strings.Join(mazeWindow(g, v), "\n")
	if !strings.ContainsAny(out, "|-") {
		t.Error("la vue tactique devrait comporter des murs ('|' ou '-')")
	}
	if !strings.Contains(out, "+") {
		t.Error("la vue tactique devrait comporter des coins '+'")
	}
}

// TestMazeWindowContact : un contact radar voisin apparaît par son NodeID.
func TestMazeWindowContact(t *testing.T) {
	// Contact une cellule au nord du joueur (8,7) : dans la fenêtre centrée (8,8).
	g, v := viewAt(8, 8, engine.North, []engine.Contact{{NodeID: 3, X: 8, Y: 7, Dist: 1}})
	rows := mazeWindow(g, v)
	if got := rows[viewCenter-2][viewCenter]; got != '3' {
		t.Errorf("le contact #3 juste au nord devrait s'afficher '3', obtenu %q", string(got))
	}
}

// TestEnergyBar : la jauge respecte ses bornes.
func TestEnergyBar(t *testing.T) {
	if b := energyBar(0, 100); b != "[----------]" {
		t.Errorf("jauge vide inattendue : %q", b)
	}
	if b := energyBar(100, 100); b != "[##########]" {
		t.Errorf("jauge pleine inattendue : %q", b)
	}
	if b := energyBar(200, 100); b != "[##########]" {
		t.Errorf("jauge devrait saturer à plein : %q", b)
	}
}

// TestBearing : le cap cardinal d'une cible relative.
func TestBearing(t *testing.T) {
	cases := map[[2]int]string{
		{0, -1}: "N", {0, 1}: "S", {1, 0}: "E", {-1, 0}: "O",
		{1, -1}: "NE", {-1, 1}: "SO", {0, 0}: "ici",
	}
	for d, want := range cases {
		if got := bearing(d[0], d[1]); got != want {
			t.Errorf("bearing(%d,%d) = %q, attendu %q", d[0], d[1], got, want)
		}
	}
}
