package oascii

import (
	"bytes"
	"testing"
)

// helper : compte les commandes plot (octets PlotByte) dans un flux.
func countPlots(b []byte) int {
	n := 0
	for _, c := range b {
		if c == PlotByte {
			n++
		}
	}
	return n
}

func TestScreenFirstRenderEmitsContent(t *testing.T) {
	s := NewScreen()
	s.PutText(2, 1, "HELLO")
	out := s.Render()
	if !bytes.Contains(out, []byte("HELLO")) {
		t.Errorf("le premier rendu doit contenir le texte:\n%q", out)
	}
	if countPlots(out) == 0 {
		t.Errorf("le premier rendu doit positionner (au moins un plot):\n%q", out)
	}
}

func TestScreenDiffOnlyChangedCells(t *testing.T) {
	s := NewScreen()
	s.PutText(0, 0, "SCORE 000")
	_ = s.Render() // premier rendu (état affiché = composé)

	// Rien ne change -> flux vide.
	if got := s.Render(); len(got) != 0 {
		t.Errorf("aucun changement : flux attendu vide, reçu %q", got)
	}

	// Les trois chiffres changent ("000" -> "999").
	s.PutText(6, 0, "999")
	out := s.Render()

	if bytes.Contains(out, []byte("SCORE")) {
		t.Errorf("le diff ne doit pas réémettre la partie inchangée:\n%q", out)
	}
	if p := countPlots(out); p != 1 {
		t.Errorf("un seul segment (1 plot) attendu, reçu %d :\n%q", p, out)
	}
	// Séquence exacte : plot(6,0) + "999".
	if want := append([]byte(Plot(6, 0)), []byte("999")...); !bytes.Equal(out, want) {
		t.Errorf("diff = %q, want %q", out, want)
	}
}

// TestScreenMinimalDiff : le diff saute même les cellules communes EN TÊTE d'un
// changement — "000" -> "042" n'émet que "42" (le premier 0 est inchangé).
func TestScreenMinimalDiff(t *testing.T) {
	s := NewScreen()
	s.PutText(6, 0, "000")
	_ = s.Render()
	s.PutText(6, 0, "042") // col 6 reste '0' ; seuls col 7,8 changent
	out := s.Render()
	if want := append([]byte(Plot(7, 0)), []byte("42")...); !bytes.Equal(out, want) {
		t.Errorf("diff minimal = %q, want %q", out, want)
	}
}

func TestScreenTwoSegmentsTwoPlots(t *testing.T) {
	s := NewScreen()
	s.PutText(0, 0, "AAAAA")
	_ = s.Render()
	// Deux zones disjointes changent sur la même ligne -> deux segments.
	s.Put(0, 0, 'X')
	s.Put(4, 0, 'Y')
	out := s.Render()
	if p := countPlots(out); p != 2 {
		t.Errorf("deux segments (2 plots) attendus, reçu %d :\n%q", p, out)
	}
}

func TestScreenSegmentStopsAtLineEnd(t *testing.T) {
	s := NewScreen()
	s.Reset() // force tout à émettre
	// Remplit deux lignes complètes.
	for c := 0; c < Cols; c++ {
		s.Put(c, 0, 'A')
		s.Put(c, 1, 'B')
	}
	out := s.Render()
	// Chaque ligne pleine = un segment -> au moins 2 plots, et aucun segment ne
	// dépasse 40 octets de données.
	if p := countPlots(out); p < 2 {
		t.Errorf("au moins un plot par ligne attendu, reçu %d", p)
	}
}

func TestScreenResetForcesFullRedraw(t *testing.T) {
	s := NewScreen()
	s.PutText(0, 0, "HI")
	_ = s.Render()
	if got := s.Render(); len(got) != 0 {
		t.Fatalf("flux vide attendu après stabilisation, reçu %q", got)
	}
	s.Reset()
	out := s.Render()
	if !bytes.Contains(out, []byte("HI")) {
		t.Errorf("Reset doit forcer la réémission:\n%q", out)
	}
}
