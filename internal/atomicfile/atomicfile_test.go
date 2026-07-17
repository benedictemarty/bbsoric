package atomicfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesAndOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := Write(path, []byte("premier")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "premier" {
		t.Fatalf("contenu = %q", b)
	}
	// Ré-écriture : remplace le contenu, pas de fichier temporaire résiduel.
	if err := Write(path, []byte("second")); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "second" {
		t.Fatalf("contenu après ré-écriture = %q", b)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("fichier temporaire résiduel : %v", entries)
	}
}

func TestWriteJSONRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.json")
	type rec struct {
		A int      `json:"a"`
		B []string `json:"b"`
	}
	in := rec{A: 7, B: []string{"x", "y"}}
	if err := WriteJSON(path, in); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	b, _ := os.ReadFile(path)
	var out rec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if out.A != 7 || len(out.B) != 2 || out.B[0] != "x" {
		t.Fatalf("round-trip KO : %+v", out)
	}
}

func TestWriteFailsOnBadDir(t *testing.T) {
	if err := Write(filepath.Join(t.TempDir(), "absent", "f.txt"), []byte("x")); err == nil {
		t.Error("écriture dans un répertoire inexistant devrait échouer")
	}
}
