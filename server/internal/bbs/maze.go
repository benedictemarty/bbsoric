package bbs

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/maze"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre l'applet du labyrinthe HIRES.
func init() { Register("maze", mazeApplet) }

// Disposition du labyrinthe dans la zone HIRES 240×200.
const (
	mazeCols = 15
	mazeRows = 11
	mazeCell = 14                              // pixels par cellule
	mazeX0   = (oascii.HiresPixW - mazeCols*mazeCell) / 2 // marge horizontale (15)
	mazeY0   = (oascii.HiresPixH - mazeRows*mazeCell) / 2 // marge verticale (23)
	mazePad  = 3                               // marge du sprite/sortie dans la cellule
)

// mazeApplet : un labyrinthe rendu en HIRES. Le joueur (carré plein) se déplace
// aux flèches jusqu'à la sortie (carré creux, coin opposé). Vitrine du buffer HIRES
// différentiel : le labyrinthe est fixe, seule la case du joueur change d'une image
// à l'autre → diff minuscule, écriture cadencée (writeHiresPaced). Tour par tour
// (ReadKey bloquant, conforme à ADR-0002).
func mazeApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	m := maze.Generate(mazeCols, mazeRows, rand.Intn)
	hs := oascii.NewHiresScreen()
	moves := 0

	// Image initiale : bascule HIRES + labyrinthe + joueur au départ.
	drawMaze(hs, m)
	if err := writeHiresPaced(s, hs.Render()); err != nil {
		return Outcome{Quit: true}
	}

	for {
		select {
		case <-ctx.Done():
			return Outcome{Quit: true}
		default:
		}
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		if key == 'Q' || key == 'q' {
			return mazeQuit(s) // abandon
		}
		dir := mazeDir(key)
		if dir == 0 || !m.Move(dir) {
			continue // touche non pertinente ou mur : rien à redessiner
		}
		moves++
		drawMaze(hs, m)
		if err := writeHiresPaced(s, hs.Render()); err != nil { // diff = case du joueur
			return Outcome{Quit: true}
		}
		if m.Solved() {
			return mazeWin(s, moves)
		}
	}
}

// mazeDir convertit une touche en direction (flèches Oric, ou ZQSD/WASD, ou pavé
// numérique 8/4/6/2). 0 = touche non directionnelle.
func mazeDir(k byte) uint8 {
	switch k {
	case keyUp, 'w', 'W', 'z', 'Z', '8':
		return maze.North
	case keyDown, 's', 'S', '2':
		return maze.South
	case keyLeft, 'a', 'A', 'q', '4': // 'q' minuscule = gauche (AZERTY) ; 'Q' majuscule = quitter
		return maze.West
	case keyRight, 'd', 'D', '6':
		return maze.East
	}
	return 0
}

// drawMaze (re)compose le labyrinthe dans le buffer différentiel : murs (fixes),
// sortie (carré creux), joueur (carré plein). Appelé à chaque image ; le diff
// n'émet que ce qui a changé (la case du joueur).
func drawMaze(hs *oascii.HiresScreen, m *maze.Maze) {
	hs.Clear()
	cols, rows := m.Size()
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			left := mazeX0 + cx*mazeCell
			top := mazeY0 + cy*mazeCell
			right := left + mazeCell
			bottom := top + mazeCell
			if m.Wall(cx, cy, maze.North) {
				hs.Line(left, top, right, top)
			}
			if m.Wall(cx, cy, maze.West) {
				hs.Line(left, top, left, bottom)
			}
			if m.Wall(cx, cy, maze.South) {
				hs.Line(left, bottom, right, bottom)
			}
			if m.Wall(cx, cy, maze.East) {
				hs.Line(right, top, right, bottom)
			}
		}
	}
	// Sortie : carré creux dans le coin opposé.
	ex, ey := m.Exit()
	el, et := mazeX0+ex*mazeCell, mazeY0+ey*mazeCell
	hs.Box(el+mazePad, et+mazePad, el+mazeCell-mazePad, et+mazeCell-mazePad)
	// Joueur : carré plein.
	px, py := m.Player()
	pl, pt := mazeX0+px*mazeCell, mazeY0+py*mazeCell
	hs.FillBox(pl+mazePad, pt+mazePad, pl+mazeCell-mazePad, pt+mazeCell-mazePad)
}

// mazeWin repasse en TEXT et affiche l'écran de victoire.
func mazeWin(s *server.Session, moves int) Outcome {
	_ = s.Write(oascii.HiresOff())
	header(s, "LABYRINTHE")
	msg := oascii.New()
	msg.Newline().Ink(oascii.Green).Text("GAGNE !").Newline().Newline()
	msg.Ink(oascii.White).Text(fmt.Sprintf("Sortie atteinte en %d coups.", moves)).Newline().Newline()
	msg.Text("Appuyez sur une touche...").Newline()
	_ = s.Write(msg.String())
	anyKey(s)
	return Outcome{Done: true}
}

// mazeQuit repasse en TEXT sur un abandon.
func mazeQuit(s *server.Session) Outcome {
	_ = s.Write(oascii.HiresOff())
	header(s, "LABYRINTHE")
	msg := oascii.New()
	msg.Newline().Ink(oascii.Yellow).Text("Abandon.").Newline().Newline()
	msg.Ink(oascii.White).Text("Appuyez sur une touche...").Newline()
	_ = s.Write(msg.String())
	anyKey(s)
	return Outcome{}
}
