package user

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fixedClock renvoie une horloge déterministe pour les tests.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestRegisterAndAuthenticate(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Register("Bob", "hunter2"); err != nil {
		t.Fatalf("Register : %v", err)
	}
	if s.Count() != 1 {
		t.Fatalf("Count attendu 1, got %d", s.Count())
	}
	// Authentification insensible à la casse du pseudo.
	u, err := s.Authenticate("bob", "hunter2")
	if err != nil {
		t.Fatalf("Authenticate : %v", err)
	}
	if u.Handle != "Bob" {
		t.Errorf("la casse d'origine du pseudo doit etre conservee, got %q", u.Handle)
	}
	if u.Calls != 1 {
		t.Errorf("Calls attendu 1 apres une connexion, got %d", u.Calls)
	}
	if _, err := s.Authenticate("bob", "mauvais"); err == nil {
		t.Errorf("un mot de passe faux doit etre rejete")
	}
	if _, err := s.Authenticate("inconnu", "hunter2"); err == nil {
		t.Errorf("un pseudo inconnu doit etre rejete")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Register("Bob", "hunter2"); err != nil {
		t.Fatalf("Register : %v", err)
	}
	if _, err := s.Register("BOB", "autre1"); err == nil {
		t.Errorf("un pseudo deja pris (casse differente) doit etre refuse")
	}
	if s.Count() != 1 {
		t.Errorf("le doublon ne doit pas etre ajoute, Count=%d", s.Count())
	}
}

func TestRegisterValidates(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Register("a", "hunter2"); err == nil {
		t.Errorf("pseudo trop court doit etre refuse")
	}
	if _, err := s.Register("Bob", "12"); err == nil {
		t.Errorf("mot de passe trop court doit etre refuse")
	}
	if s.Count() != 0 {
		t.Errorf("aucun compte invalide ne doit etre cree, Count=%d", s.Count())
	}
}

func TestCallsIncrementAndLastLogin(t *testing.T) {
	s, _ := Open("")
	ts := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	s.now = fixedClock(ts)
	s.Register("Bob", "hunter2")
	for i := 1; i <= 3; i++ {
		u, err := s.Authenticate("bob", "hunter2")
		if err != nil {
			t.Fatalf("Authenticate #%d : %v", i, err)
		}
		if u.Calls != i {
			t.Errorf("Calls attendu %d, got %d", i, u.Calls)
		}
		if !u.LastLogin.Equal(ts) {
			t.Errorf("LastLogin non mis a jour")
		}
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open (1) : %v", err)
	}
	if _, err := s1.Register("Alice", "motdepasse"); err != nil {
		t.Fatalf("Register : %v", err)
	}
	if _, err := s1.Authenticate("alice", "motdepasse"); err != nil {
		t.Fatalf("Authenticate : %v", err)
	}

	// Réouverture : les comptes et l'état (Calls) doivent persister.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("Open (2) : %v", err)
	}
	if s2.Count() != 1 {
		t.Fatalf("apres reouverture, Count attendu 1, got %d", s2.Count())
	}
	u, ok := s2.Get("alice")
	if !ok {
		t.Fatalf("le compte doit persister")
	}
	if u.Calls != 1 {
		t.Errorf("Calls doit persister, got %d", u.Calls)
	}
	// Le mot de passe doit toujours se vérifier après rechargement.
	if _, err := s2.Authenticate("alice", "motdepasse"); err != nil {
		t.Errorf("le hachage doit rester verifiable apres rechargement : %v", err)
	}
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open d'un fichier absent doit reussir : %v", err)
	}
	if s.Count() != 0 {
		t.Errorf("un store sans fichier doit etre vide, got %d", s.Count())
	}
}

func TestConcurrentAccessIsSafe(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "users.json"))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			h := "user" + string(rune('A'+n))
			if _, err := s.Register(h, "password"); err != nil {
				t.Errorf("Register concurrent %q : %v", h, err)
				return
			}
			if _, err := s.Authenticate(h, "password"); err != nil {
				t.Errorf("Authenticate concurrent %q : %v", h, err)
			}
		}(i)
	}
	wg.Wait()
	if s.Count() != 20 {
		t.Errorf("20 comptes attendus, got %d", s.Count())
	}
}
