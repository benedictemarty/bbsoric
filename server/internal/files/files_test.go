package files

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestWriteReadList(t *testing.T) {
	lib, err := Open(t.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.Write("hello.txt", []byte("salut")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := lib.Read("hello.txt")
	if err != nil || !bytes.Equal(got, []byte("salut")) {
		t.Fatalf("Read = %q, %v", got, err)
	}
	list, err := lib.List()
	if err != nil || len(list) != 1 || list[0].Name != "hello.txt" || list[0].Size != 5 {
		t.Fatalf("List = %+v, %v", list, err)
	}
}

func TestRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	lib, _ := Open(dir, 0)
	for _, bad := range []string{"../evil", "a/b", `..\x`, "", "."} {
		if err := lib.Write(bad, []byte("x")); err == nil {
			t.Errorf("Write(%q) aurait dû échouer", bad)
		}
		if _, err := lib.Read(bad); err == nil {
			t.Errorf("Read(%q) aurait dû échouer", bad)
		}
	}
	// Rien n'a fui hors du répertoire.
	if matches, _ := filepath.Glob(filepath.Join(dir, "..", "evil")); len(matches) > 0 {
		t.Errorf("fichier écrit hors du répertoire: %v", matches)
	}
}

func TestMaxSize(t *testing.T) {
	lib, _ := Open(t.TempDir(), 4)
	if err := lib.Write("ok", []byte("abcd")); err != nil {
		t.Errorf("4 octets devraient passer: %v", err)
	}
	if err := lib.Write("trop", []byte("abcde")); err == nil {
		t.Errorf("5 octets devraient être refusés")
	}
}
