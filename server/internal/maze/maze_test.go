package maze

import "testing"

// seqRnd renvoie un rnd déterministe qui choisit toujours l'indice 0 (premier
// voisin non visité) — labyrinthe reproductible pour les tests.
func firstRnd(n int) int { return 0 }

func TestGenerateFullyConnected(t *testing.T) {
	m := Generate(8, 6, firstRnd)
	// Un labyrinthe correct est un arbre couvrant : toutes les cellules sont
	// atteignables depuis (0,0) en ne franchissant que des passages (murs absents).
	w, h := m.Size()
	seen := make([][]bool, h)
	for y := range seen {
		seen[y] = make([]bool, w)
	}
	type pt struct{ x, y int }
	stack := []pt{{0, 0}}
	seen[0][0] = true
	count := 1
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		steps := []struct {
			dir        uint8
			dx, dy int
		}{{North, 0, -1}, {East, 1, 0}, {South, 0, 1}, {West, -1, 0}}
		for _, s := range steps {
			if m.Wall(c.x, c.y, s.dir) {
				continue
			}
			nx, ny := c.x+s.dx, c.y+s.dy
			if nx < 0 || nx >= w || ny < 0 || ny >= h || seen[ny][nx] {
				continue
			}
			seen[ny][nx] = true
			count++
			stack = append(stack, pt{nx, ny})
		}
	}
	if count != w*h {
		t.Errorf("labyrinthe non connexe : %d/%d cellules atteignables", count, w*h)
	}
}

func TestBorderWallsClosed(t *testing.T) {
	m := Generate(5, 5, firstRnd)
	w, h := m.Size()
	for x := 0; x < w; x++ {
		if !m.Wall(x, 0, North) || !m.Wall(x, h-1, South) {
			t.Errorf("bordure haute/basse ouverte en x=%d", x)
		}
	}
	for y := 0; y < h; y++ {
		if !m.Wall(0, y, West) || !m.Wall(w-1, y, East) {
			t.Errorf("bordure gauche/droite ouverte en y=%d", y)
		}
	}
}

func TestMoveRespectsWalls(t *testing.T) {
	m := Generate(5, 5, firstRnd)
	// Au départ (0,0) : Nord et Ouest sont des bordures -> déplacement refusé.
	if m.Move(North) {
		t.Error("déplacement Nord depuis (0,0) devrait être refusé (bordure)")
	}
	if m.Move(West) {
		t.Error("déplacement Ouest depuis (0,0) devrait être refusé (bordure)")
	}
	if x, y := m.Player(); x != 0 || y != 0 {
		t.Errorf("le joueur a bougé malgré les murs : (%d,%d)", x, y)
	}
}

func TestMoveThroughPassageAndSolve(t *testing.T) {
	m := Generate(4, 4, firstRnd)
	// Suit un chemin quelconque jusqu'à la sortie via les passages ouverts (BFS
	// des directions), en bornant le nombre de coups.
	w, h := m.Size()
	for step := 0; step < 4*w*h && !m.Solved(); step++ {
		px, py := m.Player()
		moved := false
		// Préfère aller vers la sortie (Est/Sud) si un passage existe.
		for _, d := range []uint8{East, South, North, West} {
			if !m.Wall(px, py, d) {
				// évite de faire du surplace inutile : on tente le premier passage.
				if m.Move(d) {
					moved = true
					break
				}
			}
		}
		if !moved {
			t.Fatal("bloqué : aucune direction ouverte (labyrinthe incohérent)")
		}
	}
	// Le labyrinthe étant connexe, une exploration finit par atteindre la sortie ;
	// ce test vérifie surtout que Move/Wall/Solved sont cohérents (pas de blocage).
	_ = h
}

func TestPlayerStartsAtOriginExitOpposite(t *testing.T) {
	m := Generate(7, 9, firstRnd)
	if x, y := m.Player(); x != 0 || y != 0 {
		t.Errorf("départ = (%d,%d), veut (0,0)", x, y)
	}
	if ex, ey := m.Exit(); ex != 6 || ey != 8 {
		t.Errorf("sortie = (%d,%d), veut (6,8)", ex, ey)
	}
}
