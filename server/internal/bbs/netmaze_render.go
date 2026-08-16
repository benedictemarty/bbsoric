package bbs

import (
	"fmt"
	"strings"

	"github.com/benedictemarty/netmaze/arena"
	"github.com/benedictemarty/netmaze/engine"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Rendu OASCII de NetMaze en mode TEXT 40×28. La trame est repeinte EN PLACE à
// chaque tick par positionnement du curseur (oascii.At = octet 0x1F), sans faire
// défiler l'écran : chaque ligne est réécrite pleine largeur pour effacer la
// précédente. Une seule encre est posée par ligne afin que les attributs sériels
// (qui occupent une case sur l'Oric) ne décalent pas l'alignement des colonnes.
//
// La vue est SUBJECTIVE : la StateView ne porte que l'état propre du joueur et
// les contacts radar. Le labyrinthe, lui, n'est pas secret (graine partagée) : on
// le lit dans le GameState local pour dessiner une mini-carte vue de dessus.

// facingGlyph rend l'orientation du joueur sur la carte.
func facingGlyph(d engine.Dir) byte {
	switch d {
	case engine.North:
		return '^'
	case engine.East:
		return '>'
	case engine.South:
		return 'v'
	default: // West
		return '<'
	}
}

// dirName rend une orientation en toutes lettres pour le HUD.
func dirName(d engine.Dir) string {
	switch d {
	case engine.North:
		return "Nord"
	case engine.East:
		return "Est"
	case engine.South:
		return "Sud"
	default:
		return "Ouest"
	}
}

// bearing donne le cap cardinal d'une cible relative (dx = est+, dy = sud+).
func bearing(dx, dy int) string {
	ns := ""
	if dy < 0 {
		ns = "N"
	} else if dy > 0 {
		ns = "S"
	}
	ew := ""
	if dx > 0 {
		ew = "E"
	} else if dx < 0 {
		ew = "O"
	}
	if ns == "" && ew == "" {
		return "ici"
	}
	return ns + ew
}

// energyBar rend une jauge d'énergie à segments, ex. "[####----]".
func energyBar(cur, max int) string {
	const width = 10
	if max <= 0 {
		max = 1
	}
	if cur < 0 {
		cur = 0
	}
	filled := cur * width / max
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

// netmazeViewRadius est le rayon (en cellules) de la vue tactique locale. Une
// carte globale 16×16 avec murs complets ferait 33 lignes (> 28) : on affiche
// donc une fenêtre murée centrée sur le joueur (les contacts lointains restent
// listés par la ligne radar). 9×9 cellules → 19×19 caractères, compatible 40×28.
const netmazeViewRadius = 4

// cellGlyph rend le contenu d'une cellule : joueur (prioritaire), contact radar
// par NodeID (2..9, sinon '*'), ou couloir '.'.
func cellGlyph(v *arena.StateView, x, y int) byte {
	c := byte('.')
	for _, ct := range v.Contacts {
		if ct.X == x && ct.Y == y {
			if ct.NodeID >= 2 && ct.NodeID <= 9 {
				c = byte('0' + ct.NodeID)
			} else {
				c = '*'
			}
		}
	}
	if v.Self.Alive && v.Self.X == x && v.Self.Y == y {
		c = facingGlyph(v.Self.Facing)
	}
	return c
}

// mazeWindow rend une vue tactique locale murée centrée sur le joueur, en ASCII
// classique de labyrinthe : '+' aux coins, '-'/'|' aux murs (bord de cellule où
// engine.Maze.HasWall est vrai), espace pour un passage ouvert, contenu de
// cellule au centre. Renvoie une ligne de texte par rangée de la trame.
func mazeWindow(g *engine.GameState, v *arena.StateView) []string {
	m := g.Maze
	cx, cy := v.Self.X, v.Self.Y
	n := 2*netmazeViewRadius + 1 // cellules par côté
	h, w := 2*n+1, 2*n+1         // caractères
	buf := make([][]byte, h)
	for i := range buf {
		buf[i] = make([]byte, w)
		for j := range buf[i] {
			buf[i][j] = ' '
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			mx, my := cx-netmazeViewRadius+j, cy-netmazeViewRadius+i
			if !m.In(mx, my) {
				continue // hors labyrinthe : laissé vide (pas de coins flottants)
			}
			r, c := 2*i, 2*j
			// Coins de la cellule (les cellules de bord dessinent ainsi le contour).
			buf[r][c], buf[r][c+2], buf[r+2][c], buf[r+2][c+2] = '+', '+', '+', '+'
			if m.HasWall(mx, my, engine.North) {
				buf[r][c+1] = '-'
			}
			if m.HasWall(mx, my, engine.South) {
				buf[r+2][c+1] = '-'
			}
			if m.HasWall(mx, my, engine.West) {
				buf[r+1][c] = '|'
			}
			if m.HasWall(mx, my, engine.East) {
				buf[r+1][c+2] = '|'
			}
			buf[r+1][c+1] = cellGlyph(v, mx, my)
		}
	}
	rows := make([]string, h)
	for i := range buf {
		rows[i] = string(buf[i])
	}
	return rows
}

// paint pose une ligne pleine largeur à la rangée row, encre unique, en effaçant
// le reste de la ligne (repaint en place sans défilement).
// L'octet de positionnement + l'attribut d'encre occupent la 1re case ; il reste
// donc 39 cases utiles sur la ligne de 40.
const paintWidth = 39

func paint(b *oascii.Builder, row int, ink oascii.Color, text string) {
	if len(text) > paintWidth {
		text = text[:paintWidth]
	}
	b.At(0, row).Ink(ink).Text(text + strings.Repeat(" ", paintWidth-len(text)))
}

// renderNetmaze produit la trame OASCII complète d'un tick.
func renderNetmaze(g *engine.GameState, v *arena.StateView) string {
	b := oascii.New()
	self := v.Self

	paint(b, 0, oascii.Cyan, fmt.Sprintf("NETMAZE   tick %d", v.Tick))

	// Vue tactique locale murée (déjà bordée par ses '+'/'-'/'|').
	grid := mazeWindow(g, v)
	for i, r := range grid {
		paint(b, 2+i, oascii.Green, r)
	}

	hud := 2 + len(grid) + 1
	paint(b, hud, oascii.White,
		fmt.Sprintf("Energie %s %3d/%d", energyBar(self.Energy, g.Cfg.MaxEnergy), self.Energy, g.Cfg.MaxEnergy))

	shield := "off"
	if self.Shield {
		shield = "ON"
	}
	state := "vivant"
	if !self.Alive {
		state = "DETRUIT"
	}
	paint(b, hud+1, oascii.White,
		fmt.Sprintf("Frags %d/%d  Regard %s", self.Frags, g.Cfg.FragLimit, dirName(self.Facing)))
	paint(b, hud+2, oascii.White,
		fmt.Sprintf("Bouclier %s  Etat %s", shield, state))

	// Radar : contacts relatifs (cap + distance).
	radar := "Radar: aucun contact"
	if len(v.Contacts) > 0 {
		parts := make([]string, 0, len(v.Contacts))
		for _, ct := range v.Contacts {
			parts = append(parts, fmt.Sprintf("#%d %s d%d", ct.NodeID, bearing(ct.X-self.X, ct.Y-self.Y), ct.Dist))
		}
		radar = "Radar: " + strings.Join(parts, "  ")
	}
	paint(b, hud+3, oascii.Yellow, radar)

	// Aide-mémoire des commandes (mapping identique au client Oric, cf. proto).
	paint(b, hud+5, oascii.Magenta, "w/x av/rec  q/d pivote  a/e cote")
	paint(b, hud+6, oascii.Magenta, "f tir  s bouclier  ESPACE scan  0=quit")

	return b.String()
}

// netmazeIntro est l'écran d'accueil, bref (la première trame de jeu le recouvre).
func netmazeIntro() string {
	b := oascii.New()
	paint(b, 0, oascii.Cyan, "NETMAZE - deathmatch de labyrinthe")
	paint(b, 2, oascii.White, "Vous affrontez des bots. Frags pour gagner.")
	paint(b, 3, oascii.White, "La partie demarre...")
	return b.String()
}

// netmazeEnd est l'écran de fin de partie.
func netmazeEnd(g *engine.GameState, winner *engine.Entity) string {
	b := oascii.New()
	paint(b, 0, oascii.Cyan, "NETMAZE - fin de partie")
	switch {
	case winner == nil:
		paint(b, 2, oascii.Yellow, "Match nul.")
	case winner.NodeID == 1:
		paint(b, 2, oascii.Green, "VICTOIRE ! Vous dominez le labyrinthe.")
	default:
		paint(b, 2, oascii.Red, fmt.Sprintf("DEFAITE. Le bot #%d l'emporte.", winner.NodeID))
	}
	paint(b, 4, oascii.White, "Appuyez sur une touche.")
	return b.String()
}
