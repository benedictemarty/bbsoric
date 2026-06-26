package content

import "testing"

// siteAvecDW construit un Site minimal valide avec une source et une page grille,
// puis applique la mutation mut pour produire les cas d'erreur.
func siteAvecDW(mut func(*Site)) *Site {
	s := &Site{
		Start: "g",
		SourcesDonnees: map[string]SourceDonnees{
			"rep": {
				Table: "rep",
				Colonnes: map[string]ColonneDef{
					"id":  {Type: "INTEGER", ClePrimaire: true, AutoIncrement: true},
					"nom": {Type: "TEXT", Requis: true},
				},
			},
		},
		Pages: map[string]*Page{
			"g": {Title: "GRILLE", DataWindow: &DataWindow{
				Source:            "rep",
				ColonnesAffichees: []string{"nom"},
				Largeurs:          []int{20},
			}},
		},
	}
	if mut != nil {
		mut(s)
	}
	return s
}

func TestValidateDataWindowOK(t *testing.T) {
	if err := siteAvecDW(nil).Validate(); err != nil {
		t.Fatalf("site valide refusé : %v", err)
	}
}

func TestValidateDataWindowErreurs(t *testing.T) {
	cas := []struct {
		nom string
		mut func(*Site)
	}{
		{"source inconnue", func(s *Site) { s.Pages["g"].DataWindow.Source = "absente" }},
		{"colonne inconnue", func(s *Site) { s.Pages["g"].DataWindow.ColonnesAffichees = []string{"xxx"} }},
		{"largeurs incohérentes", func(s *Site) { s.Pages["g"].DataWindow.Largeurs = []int{10, 10} }},
		{"trop large", func(s *Site) {
			s.Pages["g"].DataWindow.ColonnesAffichees = []string{"nom", "nom", "nom", "nom", "nom"}
			s.Pages["g"].DataWindow.Largeurs = []int{12, 12, 12, 12, 12}
		}},
		{"table invalide", func(s *Site) { src := s.SourcesDonnees["rep"]; src.Table = "a; DROP"; s.SourcesDonnees["rep"] = src }},
		{"type invalide", func(s *Site) {
			src := s.SourcesDonnees["rep"]
			src.Colonnes["nom"] = ColonneDef{Type: "VARCHAR(9)"}
			s.SourcesDonnees["rep"] = src
		}},
	}
	for _, c := range cas {
		if err := siteAvecDW(c.mut).Validate(); err == nil {
			t.Errorf("%s : erreur attendue, aucune", c.nom)
		}
	}
}
