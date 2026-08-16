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

// TestRenderNetmazeContenu : la trame contient l'en-tête, le glyphe du joueur à
// sa position et les rubriques du HUD, sans paniquer.
func TestRenderNetmazeContenu(t *testing.T) {
	g, v := viewAt(3, 4, engine.East, nil)
	out := renderNetmaze(g, v)
	for _, want := range []string{"NETMAZE", "Energie", "Radar", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("la trame devrait contenir %q", want)
		}
	}
	rows := mazeGrid(g, v)
	if got := rows[4][3]; got != '>' {
		t.Errorf("le joueur tourné vers l'Est devrait s'afficher '>' en (3,4), obtenu %q", string(got))
	}
}

// TestMazeGridContact : un contact radar apparaît par son NodeID sur la carte.
func TestMazeGridContact(t *testing.T) {
	g, v := viewAt(0, 0, engine.North, []engine.Contact{{NodeID: 3, X: 5, Y: 6, Dist: 4}})
	rows := mazeGrid(g, v)
	if got := rows[6][5]; got != '3' {
		t.Errorf("le contact #3 devrait s'afficher '3' en (5,6), obtenu %q", string(got))
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
