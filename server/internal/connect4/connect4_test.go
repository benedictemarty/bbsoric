package connect4

import "testing"

// fill dépose une suite de coups (col, jeton) et échoue si l'un est refusé.
func fill(t *testing.T, g *Game, moves []struct {
	col int
	c   Cell
}) {
	t.Helper()
	for i, m := range moves {
		if _, ok := g.Drop(m.col, m.c); !ok {
			t.Fatalf("coup #%d (col %d) refusé", i, m.col)
		}
	}
}

func TestDropStacks(t *testing.T) {
	g := New()
	r1, _ := g.Drop(3, Player)
	r2, _ := g.Drop(3, Computer)
	if r1 != 0 || r2 != 1 {
		t.Fatalf("empilement KO : r1=%d r2=%d", r1, r2)
	}
	if g.At(0, 3) != Player || g.At(1, 3) != Computer {
		t.Errorf("cases inattendues")
	}
}

func TestColumnFull(t *testing.T) {
	g := New()
	for i := 0; i < Rows; i++ {
		g.Drop(0, Player)
	}
	if !g.ColumnFull(0) {
		t.Fatal("colonne devrait être pleine")
	}
	if _, ok := g.Drop(0, Player); ok {
		t.Error("dépôt accepté dans une colonne pleine")
	}
}

func TestWinHorizontal(t *testing.T) {
	g := New()
	for c := 0; c < 4; c++ {
		g.Drop(c, Player)
	}
	if g.Winner() != Player {
		t.Fatalf("victoire horizontale non détectée : %v", g.Winner())
	}
}

func TestWinVertical(t *testing.T) {
	g := New()
	for i := 0; i < 4; i++ {
		g.Drop(2, Computer)
	}
	if g.Winner() != Computer {
		t.Fatalf("victoire verticale non détectée : %v", g.Winner())
	}
}

func TestWinDiagonal(t *testing.T) {
	g := New()
	// Construit une diagonale ↗ de Player en colonnes 0..3.
	// col0: P ; col1: C,P ; col2: C,C,P ; col3: C,C,C,P
	fill(t, g, []struct {
		col int
		c   Cell
	}{
		{0, Player},
		{1, Computer}, {1, Player},
		{2, Computer}, {2, Computer}, {2, Player},
		{3, Computer}, {3, Computer}, {3, Computer}, {3, Player},
	})
	if g.Winner() != Player {
		t.Fatalf("victoire diagonale non détectée : %v", g.Winner())
	}
}

func TestNoWinnerEmpty(t *testing.T) {
	if New().Winner() != Empty {
		t.Error("plateau vide ne doit pas avoir de gagnant")
	}
}

func TestComputerTakesWin(t *testing.T) {
	g := New()
	// Computer a 3 alignés en bas des colonnes 0,1,2 ; doit compléter en 3.
	g.Drop(0, Computer)
	g.Drop(1, Computer)
	g.Drop(2, Computer)
	if got := g.ComputerMove(); got != 3 {
		t.Fatalf("l'ordinateur devrait jouer 3 pour gagner, a joué %d", got)
	}
}

func TestComputerBlocksPlayer(t *testing.T) {
	g := New()
	// Player menace de gagner en 3 (3 alignés cols 0,1,2) ; pas de victoire
	// immédiate pour Computer → il doit bloquer en 3.
	g.Drop(0, Player)
	g.Drop(1, Player)
	g.Drop(2, Player)
	if got := g.ComputerMove(); got != 3 {
		t.Fatalf("l'ordinateur devrait bloquer en 3, a joué %d", got)
	}
}

func TestComputerPrefersCenter(t *testing.T) {
	g := New()
	if got := g.ComputerMove(); got != 3 {
		t.Fatalf("sur plateau vide, l'ordinateur devrait préférer le centre (3), a joué %d", got)
	}
}

func TestFull(t *testing.T) {
	g := New()
	if g.Full() {
		t.Fatal("plateau vide déclaré plein")
	}
	// Remplit sans gagnant en alternant des paires de colonnes (motif qui évite
	// 4 alignés) n'est pas trivial ; on remplit tout et on vérifie juste Full.
	for c := 0; c < Cols; c++ {
		for r := 0; r < Rows; r++ {
			g.Drop(c, Player)
		}
	}
	if !g.Full() {
		t.Error("plateau rempli non déclaré plein")
	}
}
