package content

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestStoreLoadAndHotReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "site.json")

	v1 := `{"start":"m","pages":{"m":{"title":"V1","type":"menu","entries":[{"key":"Q","target":"__quit__"}]}}}`
	if err := os.WriteFile(path, []byte(v1), 0o644); err != nil {
		t.Fatal(err)
	}

	old := PollInterval
	PollInterval = 50 * time.Millisecond
	defer func() { PollInterval = old }()

	st := NewStore(path, discardLog())
	if got := st.Site().Pages["m"].Title; got != "V1" {
		t.Fatalf("chargement initial: titre = %q, want V1", got)
	}

	// Modifie le fichier (mtime forcé dans le futur pour fiabilité).
	v2 := `{"start":"m","pages":{"m":{"title":"V2","type":"menu","entries":[{"key":"Q","target":"__quit__"}]}}}`
	if err := os.WriteFile(path, []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)

	// Attend le rechargement à chaud.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st.Site().Pages["m"].Title == "V2" {
			return // succès
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("rechargement à chaud non détecté: titre = %q", st.Site().Pages["m"].Title)
}

func TestStoreBadFileKeepsOld(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "site.json")
	good := `{"start":"m","pages":{"m":{"title":"BON","type":"menu","entries":[{"key":"Q","target":"__quit__"}]}}}`
	os.WriteFile(path, []byte(good), 0o644)

	old := PollInterval
	PollInterval = 50 * time.Millisecond
	defer func() { PollInterval = old }()

	st := NewStore(path, discardLog())
	if st.Site().Pages["m"].Title != "BON" {
		t.Fatal("chargement initial échoué")
	}

	// Écrit un JSON invalide : l'ancienne version doit être conservée.
	os.WriteFile(path, []byte("{ casse"), 0o644)
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)

	time.Sleep(300 * time.Millisecond)
	if st.Site().Pages["m"].Title != "BON" {
		t.Fatalf("fichier invalide a écrasé le contenu valide")
	}
}

func TestStoreNilPathUsesDefault(t *testing.T) {
	st := NewStore("", discardLog())
	if st.Site().Start != "main" {
		t.Errorf("store sans fichier devrait servir le contenu par défaut")
	}
}
