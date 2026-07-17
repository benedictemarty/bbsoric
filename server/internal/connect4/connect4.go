// Package connect4 implémente la logique du jeu de Puissance 4 (Connect Four),
// indépendamment de tout affichage : plateau 7 colonnes × 6 lignes, dépôt d'un
// jeton, détection de victoire (4 alignés) et coup de l'ordinateur (heuristique
// déterministe). L'applet BBS habille cette logique en OASCII.
package connect4

// Dimensions du plateau.
const (
	Cols = 7
	Rows = 6
)

// Cell est le contenu d'une case.
type Cell uint8

const (
	Empty    Cell = iota
	Player        // jeton du joueur humain
	Computer      // jeton de l'ordinateur
)

// Game porte l'état d'une partie. La ligne 0 est le bas du plateau (les jetons
// « tombent » vers les lignes basses).
type Game struct {
	cells [Rows][Cols]Cell
}

// New crée une partie vide.
func New() *Game { return &Game{} }

// At renvoie le contenu de la case (row, col). row 0 = bas.
func (g *Game) At(row, col int) Cell { return g.cells[row][col] }

// ColumnFull indique si la colonne est pleine.
func (g *Game) ColumnFull(col int) bool {
	return col < 0 || col >= Cols || g.cells[Rows-1][col] != Empty
}

// Full indique si le plateau est plein (match nul si aucun gagnant).
func (g *Game) Full() bool {
	for c := 0; c < Cols; c++ {
		if !g.ColumnFull(c) {
			return false
		}
	}
	return true
}

// Drop dépose un jeton c dans la colonne col. Renvoie la ligne occupée et true,
// ou -1 et false si la colonne est pleine ou invalide.
func (g *Game) Drop(col int, c Cell) (int, bool) {
	if col < 0 || col >= Cols {
		return -1, false
	}
	for r := 0; r < Rows; r++ {
		if g.cells[r][col] == Empty {
			g.cells[r][col] = c
			return r, true
		}
	}
	return -1, false
}

// undrop retire le jeton le plus haut de la colonne (pour la simulation de l'IA).
func (g *Game) undrop(col int) {
	for r := Rows - 1; r >= 0; r-- {
		if g.cells[r][col] != Empty {
			g.cells[r][col] = Empty
			return
		}
	}
}

// Winner renvoie le jeton gagnant (4 alignés) ou Empty. Parcourt toutes les
// lignes de 4 dans les 4 directions.
func (g *Game) Winner() Cell {
	dirs := [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} // →, ↑, ↗, ↖
	for r := 0; r < Rows; r++ {
		for c := 0; c < Cols; c++ {
			who := g.cells[r][c]
			if who == Empty {
				continue
			}
			for _, d := range dirs {
				if g.four(r, c, d[0], d[1], who) {
					return who
				}
			}
		}
	}
	return Empty
}

// four teste 4 cases identiques à partir de (r,c) dans la direction (dr,dc).
func (g *Game) four(r, c, dr, dc int, who Cell) bool {
	for i := 0; i < 4; i++ {
		rr, cc := r+dr*i, c+dc*i
		if rr < 0 || rr >= Rows || cc < 0 || cc >= Cols || g.cells[rr][cc] != who {
			return false
		}
	}
	return true
}

// winsWith teste si jouer col donne immédiatement la victoire à who.
func (g *Game) winsWith(col int, who Cell) bool {
	r, ok := g.Drop(col, who)
	if !ok {
		return false
	}
	win := g.Winner() == who
	g.cells[r][col] = Empty
	return win
}

// ComputerMove choisit une colonne pour l'ordinateur : 1) gagner si possible,
// 2) sinon bloquer une victoire adverse imminente, 3) sinon préférer le centre.
// Déterministe. Renvoie -1 si le plateau est plein.
func (g *Game) ComputerMove() int {
	// 1) coup gagnant.
	for c := 0; c < Cols; c++ {
		if !g.ColumnFull(c) && g.winsWith(c, Computer) {
			return c
		}
	}
	// 2) blocage d'un coup gagnant adverse.
	for c := 0; c < Cols; c++ {
		if !g.ColumnFull(c) && g.winsWith(c, Player) {
			return c
		}
	}
	// 3) préférence centrale.
	for _, c := range []int{3, 2, 4, 1, 5, 0, 6} {
		if !g.ColumnFull(c) {
			return c
		}
	}
	return -1
}
