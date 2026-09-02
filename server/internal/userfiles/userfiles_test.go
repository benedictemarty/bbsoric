package userfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func openTmp(t *testing.T, maxFiles int, maxBytes int64) *Store {
	t.Helper()
	st, err := Open(t.TempDir(), maxFiles, maxBytes, 64*1024)
	if err != nil {
		t.Fatalf("Open : %v", err)
	}
	return st
}

// L'espace d'un utilisateur est isolé et persistant ; deux comptes ne se voient pas.
func TestPerUserIsolation(t *testing.T) {
	st := openTmp(t, 0, 0)
	if err := st.Write("alice", "A.TAP", []byte("aaa")); err != nil {
		t.Fatalf("write alice : %v", err)
	}
	if err := st.Write("bob", "B.TAP", []byte("bbbb")); err != nil {
		t.Fatalf("write bob : %v", err)
	}
	la, _ := st.For("alice")
	lb, _ := st.For("bob")
	fa, _ := la.List()
	fb, _ := lb.List()
	if len(fa) != 1 || fa[0].Name != "A.TAP" {
		t.Errorf("espace alice = %v, attendu [A.TAP]", fa)
	}
	if len(fb) != 1 || fb[0].Name != "B.TAP" {
		t.Errorf("espace bob = %v, attendu [B.TAP]", fb)
	}
	if data, err := la.Read("A.TAP"); err != nil || string(data) != "aaa" {
		t.Errorf("relecture alice : %q, %v", data, err)
	}
}

// Le pseudo est normalisé : "Bob" et "bob" partagent le même espace.
func TestHandleCaseInsensitive(t *testing.T) {
	st := openTmp(t, 0, 0)
	if err := st.Write("Bob", "X.TAP", []byte("x")); err != nil {
		t.Fatalf("write : %v", err)
	}
	n, _, err := st.Usage("bob")
	if err != nil || n != 1 {
		t.Errorf("usage(bob) = %d, %v ; attendu 1 (même espace que Bob)", n, err)
	}
}

// Quota en NOMBRE de fichiers : le (maxFiles+1)-ième est refusé sans écriture.
func TestQuotaFileCount(t *testing.T) {
	st := openTmp(t, 2, 0)
	must := func(name string) {
		if err := st.Write("u", name, []byte("z")); err != nil {
			t.Fatalf("write %s : %v", name, err)
		}
	}
	must("F1")
	must("F2")
	if err := st.Write("u", "F3", []byte("z")); err == nil {
		t.Fatal("F3 aurait dû être refusé (quota 2 fichiers)")
	}
	// Remplacer un fichier existant reste permis (ne crée pas de 3e entrée).
	if err := st.Write("u", "F2", []byte("zz")); err != nil {
		t.Errorf("remplacement F2 refusé à tort : %v", err)
	}
	n, _, _ := st.Usage("u")
	if n != 2 {
		t.Errorf("nb fichiers = %d, attendu 2", n)
	}
}

// Quota en OCTETS : le dépassement du total est refusé ; un remplacement recompte
// correctement (l'ancienne taille est déduite).
func TestQuotaBytes(t *testing.T) {
	st := openTmp(t, 0, 10) // 10 octets au total
	if err := st.Write("u", "A", []byte("12345")); err != nil {
		t.Fatalf("A(5) : %v", err)
	}
	if err := st.Write("u", "B", []byte("12345")); err != nil {
		t.Fatalf("B(5) -> total 10 : %v", err)
	}
	if err := st.Write("u", "C", []byte("1")); err == nil {
		t.Fatal("C aurait dû être refusé (10/10 déjà utilisés)")
	}
	// Remplacer A(5) par A(3) libère 2 octets -> C(1) passe ensuite.
	if err := st.Write("u", "A", []byte("123")); err != nil {
		t.Errorf("réduction de A refusée à tort : %v", err)
	}
	if err := st.Write("u", "C", []byte("1")); err != nil {
		t.Errorf("C(1) refusé alors que le total le permet (3+5+1=9) : %v", err)
	}
}

// Un pseudo qui ne se réduit pas à [a-z0-9_-] est refusé (défense path-traversal).
func TestUnsafeHandleRejected(t *testing.T) {
	st := openTmp(t, 0, 0)
	for _, bad := range []string{"../etc", "a/b", "..", "a.b", ""} {
		if _, err := st.For(bad); err == nil {
			t.Errorf("pseudo %q aurait dû être refusé", bad)
		}
	}
	// Le fichier ne doit pas non plus s'écrire hors de la racine.
	if err := st.Write("../evil", "X", []byte("z")); err == nil {
		t.Error("write avec pseudo hors périmètre aurait dû échouer")
	}
}

// La racine du store contient bien un sous-répertoire nommé d'après le pseudo.
func TestPerUserDirName(t *testing.T) {
	root := t.TempDir()
	st, _ := Open(root, 0, 0, 0)
	if err := st.Write("Alice", "F", []byte("z")); err != nil {
		t.Fatalf("write : %v", err)
	}
	if _, err := st.For("alice"); err != nil {
		t.Fatalf("For : %v", err)
	}
	// Le dossier doit être <root>/alice (pseudo normalisé).
	if fi, err := os.Stat(filepath.Join(root, "alice")); err != nil || !fi.IsDir() {
		t.Errorf("sous-répertoire <root>/alice absent : %v", err)
	}
}
