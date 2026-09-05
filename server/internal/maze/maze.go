// Package maze porte la LOGIQUE PURE du jeu de labyrinthe (génération, murs,
// déplacement, résolution), indépendante de tout rendu ou entrée — comme
// server/internal/connect4. L'applet bbs `maze` s'occupe du rendu HIRES et de la
// saisie ; ce package se teste sans réseau.
//
// Le labyrinthe est une grille w×h de cellules. Chaque cellule porte un masque de
// murs (N/E/S/O). La génération (backtracker récursif) creuse un ARBRE couvrant :
// toutes les cellules sont accessibles et il existe un unique chemin entre deux
// cellules. Le joueur part en (0,0), la sortie est en (w-1,h-1).
package maze

// Murs d'une cellule (masque de bits). Deux cellules adjacentes partagent un mur :
// enlever le mur Est de l'une enlève le mur Ouest de l'autre.
const (
	North uint8 = 1 << iota
	East
	South
	West
)

// opposite renvoie le mur opposé (Nord<->Sud, Est<->Ouest).
func opposite(d uint8) uint8 {
	switch d {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	case West:
		return East
	}
	return 0
}

// Maze est une grille de cellules avec la position du joueur.
type Maze struct {
	w, h   int
	cells  [][]uint8 // cells[y][x] = masque des murs présents
	px, py int       // position du joueur
}

type neighbor struct {
	x, y int
	dir  uint8
}

// Generate construit un labyrinthe w×h par backtracker récursif (itératif). rnd(n)
// doit renvoyer un entier dans [0,n) — injectable pour des tests déterministes
// (math/rand.Intn en jeu réel). w,h ≥ 1.
func Generate(w, h int, rnd func(n int) int) *Maze {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m := &Maze{w: w, h: h, cells: make([][]uint8, h)}
	for y := 0; y < h; y++ {
		m.cells[y] = make([]uint8, w)
		for x := 0; x < w; x++ {
			m.cells[y][x] = North | East | South | West // tous murs fermés
		}
	}
	visited := make([][]bool, h)
	for y := range visited {
		visited[y] = make([]bool, w)
	}

	type pt struct{ x, y int }
	stack := []pt{{0, 0}}
	visited[0][0] = true
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		nb := unvisitedNeighbors(cur.x, cur.y, w, h, visited)
		if len(nb) == 0 {
			stack = stack[:len(stack)-1] // cul-de-sac : on revient
			continue
		}
		n := nb[rnd(len(nb))]
		// Creuse le passage : enlève le mur des deux côtés.
		m.cells[cur.y][cur.x] &^= n.dir
		m.cells[n.y][n.x] &^= opposite(n.dir)
		visited[n.y][n.x] = true
		stack = append(stack, pt{n.x, n.y})
	}
	return m
}

func unvisitedNeighbors(x, y, w, h int, visited [][]bool) []neighbor {
	var nb []neighbor
	if y > 0 && !visited[y-1][x] {
		nb = append(nb, neighbor{x, y - 1, North})
	}
	if x < w-1 && !visited[y][x+1] {
		nb = append(nb, neighbor{x + 1, y, East})
	}
	if y < h-1 && !visited[y+1][x] {
		nb = append(nb, neighbor{x, y + 1, South})
	}
	if x > 0 && !visited[y][x-1] {
		nb = append(nb, neighbor{x - 1, y, West})
	}
	return nb
}

// Size renvoie les dimensions (w,h) en cellules.
func (m *Maze) Size() (int, int) { return m.w, m.h }

// Player renvoie la position courante du joueur.
func (m *Maze) Player() (int, int) { return m.px, m.py }

// Exit renvoie la cellule de sortie (coin opposé).
func (m *Maze) Exit() (int, int) { return m.w - 1, m.h - 1 }

// Wall indique s'il y a un mur du côté dir de la cellule (x,y).
func (m *Maze) Wall(x, y int, dir uint8) bool {
	if x < 0 || x >= m.w || y < 0 || y >= m.h {
		return true
	}
	return m.cells[y][x]&dir != 0
}

// Move tente de déplacer le joueur dans la direction dir. Renvoie true s'il a
// bougé (pas de mur), false sinon. Les murs de bordure empêchent de sortir.
func (m *Maze) Move(dir uint8) bool {
	if m.cells[m.py][m.px]&dir != 0 {
		return false // mur
	}
	switch dir {
	case North:
		m.py--
	case South:
		m.py++
	case East:
		m.px++
	case West:
		m.px--
	}
	return true
}

// Solved indique si le joueur a atteint la sortie.
func (m *Maze) Solved() bool { return m.px == m.w-1 && m.py == m.h-1 }
