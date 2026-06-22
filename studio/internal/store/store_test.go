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
