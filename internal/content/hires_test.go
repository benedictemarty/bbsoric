package content

import (
	"strings"
	"testing"
)

// siteAvecHires construit un Site minimal avec une page HIRES, puis applique la
// mutation mut pour produire les cas d'erreur.
func siteAvecHires(mut func(*Site)) *Site {
	s := &Site{
		Start: "h",
		Pages: map[string]*Page{
			"h": {Title: "HIRES", Hires: &Hires{
				Draw: []HiresOp{
					{Op: "ink", C: 3},
					{Op: "curset", X: 10, Y: 20},
					{Op: "line", X: 230, Y: 190},
					{Op: "circle", R: 30},
					{Op: "char", X: 0, Y: 0, Ch: "A"},
				},
			}},
		},
	}
	if mut != nil {
		mut(s)
	}
	return s
}

func TestValidateHiresOK(t *testing.T) {
	if err := siteAvecHires(nil).Validate(); err != nil {
		t.Fatalf("page HIRES (primitives) valide refusée : %v", err)
	}
	// Fond bitmap de la bonne taille, sans primitives.
	bg := siteAvecHires(func(s *Site) {
		s.Pages["h"].Hires = &Hires{Background: make([]byte, HiresBitmapSize)}
	})
	if err := bg.Validate(); err != nil {
		t.Fatalf("page HIRES (bitmap) valide refusée : %v", err)
	}
	if HiresBitmapSize != 8000 {
		t.Fatalf("HiresBitmapSize attendu 8000, obtenu %d", HiresBitmapSize)
	}
}

func TestValidateHiresErreurs(t *testing.T) {
	cas := []struct {
		nom string
		mut func(*Site)
	}{
		{"vide (ni background ni draw)", func(s *Site) { s.Pages["h"].Hires = &Hires{} }},
		{"background mauvaise taille", func(s *Site) { s.Pages["h"].Hires = &Hires{Background: make([]byte, 100)} }},
		{"primitive inconnue", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "zigzag"}} }},
		{"couleur hors bornes", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "ink", C: 9}} }},
		{"point hors écran (x)", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "point", X: 240, Y: 0}} }},
		{"point hors écran (y)", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "line", X: 0, Y: 200}} }},
		{"rayon négatif", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "circle", R: -1}} }},
		{"char sans ch", func(s *Site) { s.Pages["h"].Hires.Draw = []HiresOp{{Op: "char", X: 0, Y: 0}} }},
	}
	for _, c := range cas {
		if err := siteAvecHires(c.mut).Validate(); err == nil {
			t.Errorf("%s : aucune erreur alors qu'une était attendue", c.nom)
		} else if !strings.Contains(err.Error(), "hires") && !strings.Contains(err.Error(), "page") {
			t.Errorf("%s : message inattendu : %v", c.nom, err)
		}
	}
}
