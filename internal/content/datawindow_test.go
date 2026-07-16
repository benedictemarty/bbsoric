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

func TestValidateSourceAPI(t *testing.T) {
	// Source API valide (url présente, pas de table SQL exigée).
	ok := siteAvecDW(func(s *Site) {
		src := s.SourcesDonnees["rep"]
		src.Table = "" // pas de table pour une source API
		src.TypeSource = "api"
		src.API = &APIConfig{URL: "http://example/data.json"}
		s.SourcesDonnees["rep"] = src
	})
	if err := ok.Validate(); err != nil {
		t.Errorf("source API valide refusée : %v", err)
	}
	// Source API sans url -> erreur.
	ko := siteAvecDW(func(s *Site) {
		src := s.SourcesDonnees["rep"]
		src.TypeSource = "api"
		src.API = nil
		s.SourcesDonnees["rep"] = src
	})
	if err := ko.Validate(); err == nil {
		t.Error("source API sans url aurait dû échouer")
	}
}

// TestValidateColumnPattern : un motif de validation (regex) invalide est refusé
// au chargement, pas à l'exécution (S11.7).
func TestValidateColumnPattern(t *testing.T) {
	ok := siteAvecDW(func(s *Site) {
		src := s.SourcesDonnees["rep"]
		src.Colonnes["nom"] = ColonneDef{Type: "TEXT", Pattern: `^[A-Z]+$`}
		s.SourcesDonnees["rep"] = src
	})
	if err := ok.Validate(); err != nil {
		t.Errorf("un motif valide ne doit pas être refusé : %v", err)
	}
	ko := siteAvecDW(func(s *Site) {
		src := s.SourcesDonnees["rep"]
		src.Colonnes["nom"] = ColonneDef{Type: "TEXT", Pattern: `[A-Z`} // regex invalide
		s.SourcesDonnees["rep"] = src
	})
	if err := ko.Validate(); err == nil {
		t.Error("un motif regex invalide aurait dû échouer")
	}
}

// TestValidateFichierColonne : fichier_colonne doit désigner une colonne de la source.
func TestValidateFichierColonne(t *testing.T) {
	ok := siteAvecDW(func(s *Site) {
		s.Pages["g"].DataWindow.FichierColonne = "nom" // colonne existante
	})
	if err := ok.Validate(); err != nil {
		t.Errorf("fichier_colonne valide refusé : %v", err)
	}
	ko := siteAvecDW(func(s *Site) {
		s.Pages["g"].DataWindow.FichierColonne = "absente"
	})
	if err := ko.Validate(); err == nil {
		t.Error("fichier_colonne inconnue aurait dû échouer")
	}
}

// TestValidateFiltreFixe : filtre_fixe doit désigner une colonne existante (J3).
func TestValidateFiltreFixe(t *testing.T) {
	ok := siteAvecDW(func(s *Site) {
		s.Pages["g"].DataWindow.FiltreFixe = &FiltreFixe{Colonne: "nom", Valeur: "Alice"}
	})
	if err := ok.Validate(); err != nil {
		t.Errorf("filtre_fixe valide refusé : %v", err)
	}
	ko := siteAvecDW(func(s *Site) {
		s.Pages["g"].DataWindow.FiltreFixe = &FiltreFixe{Colonne: "absente", Valeur: "x"}
	})
	if err := ko.Validate(); err == nil {
		t.Error("filtre_fixe colonne inconnue aurait dû échouer")
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
