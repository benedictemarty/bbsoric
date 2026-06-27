package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
)

const validSite = `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"Q","label":"Quitter","target":"__quit__"}]}}}`

func TestListLoadSave(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "site.json"), []byte(validSite), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	s := New(dir)

	names, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 1 || names[0] != "site.json" {
		t.Fatalf("List = %v, attendu [site.json]", names)
	}

	data, err := s.Load("site.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := content.Parse(data); err != nil {
		t.Fatalf("le site chargé doit être valide : %v", err)
	}
}

// dwSite : un site DataWindow (source + page grille) doit passer la validation
// du studio et round-tripper sans perte (régression : ne pas droper sources/datawindow).
const dwSite = `{"start":"g","sources_donnees":{"rep":{"table":"rep","colonnes":{"id":{"type":"INTEGER","cle_primaire":true,"auto_increment":true},"nom":{"type":"TEXT","requis":true}}}},"pages":{"g":{"title":"GRILLE","datawindow":{"source":"rep","colonnes_affichees":["nom"],"largeurs":[20]}}}}`

func TestSaveLoadDataWindowRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save("dw.json", []byte(dwSite)); err != nil {
		t.Fatalf("Save site DataWindow refusé : %v", err)
	}
	data, err := s.Load("dw.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	site, err := content.Parse(data)
	if err != nil {
		t.Fatalf("le site DataWindow rechargé doit être valide : %v", err)
	}
	if _, ok := site.SourcesDonnees["rep"]; !ok {
		t.Error("la source 'rep' a été perdue au round-trip")
	}
	if p := site.Pages["g"]; p == nil || p.DataWindow == nil || p.DataWindow.Source != "rep" {
		t.Errorf("le descripteur datawindow de la page 'g' a été perdu : %+v", site.Pages["g"])
	}
}

func TestSaveValidatesBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// JSON valide -> écrit et relisible.
	if err := s.Save("ok.json", []byte(validSite)); err != nil {
		t.Fatalf("Save valide: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ok.json")); err != nil {
		t.Errorf("le fichier doit exister: %v", err)
	}

	// JSON invalide (cible inexistante) -> refusé, pas de fichier.
	bad := `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"1","label":"x","target":"absent"}]}}}`
	if err := s.Save("bad.json", []byte(bad)); err == nil {
		t.Errorf("un site invalide doit être refusé")
	}
	if _, err := os.Stat(filepath.Join(dir, "bad.json")); !os.IsNotExist(err) {
		t.Errorf("aucun fichier ne doit être écrit pour un site invalide")
	}
}

func TestSafePathRejectsTraversal(t *testing.T) {
	s := New(t.TempDir())
	for _, bad := range []string{"../x.json", "a/b.json", "..", "site.txt", ""} {
		if _, err := s.Load(bad); err == nil {
			t.Errorf("Load(%q) devrait échouer", bad)
		}
	}
}
