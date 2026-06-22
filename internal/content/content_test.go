package content

import (
	"strings"
	"testing"
)

func TestDefaultSiteValid(t *testing.T) {
	if err := DefaultSite().Validate(); err != nil {
		t.Fatalf("site par défaut invalide: %v", err)
	}
}

func TestParseValid(t *testing.T) {
	js := `{
		"start": "main",
		"pages": {
			"main": {"title": "ACCUEIL", "type": "menu", "entries": [
				{"key": "1", "label": "Infos", "target": "info"},
				{"key": "Q", "label": "Quitter", "target": "__quit__"}
			]},
			"info": {"title": "INFOS", "type": "page", "lines": [
				{"text": "Bonjour", "ink": "yellow"}
			]}
		}
	}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Start != "main" || len(s.Pages) != 2 {
		t.Fatalf("site inattendu: %+v", s)
	}
	if s.Pages["main"].Entries[0].Target != "info" {
		t.Errorf("cible inattendue")
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]string{
		"start manquant":    `{"pages": {"a": {"type": "page"}}}`,
		"start introuvable": `{"start": "x", "pages": {"a": {"type": "page"}}}`,
		"cible introuvable": `{"start": "a", "pages": {"a": {"type": "menu", "entries": [{"key": "1", "target": "zzz"}]}}}`,
		"type inconnu":      `{"start": "a", "pages": {"a": {"type": "truc"}}}`,
		"aucune page":       `{"start": "a", "pages": {}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: erreur attendue, obtenu nil", name)
		}
	}
}

func TestSpecialTargetsAccepted(t *testing.T) {
	js := `{"start": "m", "pages": {"m": {"type": "menu", "entries": [
		{"key": "B", "target": "__back__"},
		{"key": "H", "target": "__home__"},
		{"key": "Q", "target": "__quit__"}
	]}}}`
	if _, err := Parse([]byte(js)); err != nil {
		t.Fatalf("cibles spéciales rejetées: %v", err)
	}
}

func TestInk(t *testing.T) {
	if Ink("yellow") != 3 { // Yellow = 3
		t.Errorf("Ink(yellow) inattendu")
	}
	if Ink("Cyan") != 6 { // insensible à la casse, Cyan = 6
		t.Errorf("Ink(Cyan) inattendu")
	}
	if Ink("inconnu") != 7 { // défaut White = 7
		t.Errorf("Ink défaut devrait être blanc")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse([]byte("pas du json")); err == nil {
		t.Error("JSON invalide accepté")
	} else if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("message d'erreur inattendu: %v", err)
	}
}
