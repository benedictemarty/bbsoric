package wall

import (
	"path/filepath"
	"testing"
	"time"
)

// fixedClock renvoie une horloge déterministe pour des dates stables en test.
func fixedClock() func() time.Time {
	t := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(time.Minute); return t }
}

func TestPostAndListOrder(t *testing.T) {
	s, err := Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.now = fixedClock()
	for _, txt := range []string{"premier", "deuxieme", "troisieme"} {
		if _, err := s.Post("bob", txt); err != nil {
			t.Fatalf("Post(%q): %v", txt, err)
		}
	}
	if s.Count() != 3 {
		t.Fatalf("Count = %d, veut 3", s.Count())
	}
	list := s.List(0) // tout, antéchronologique
	if len(list) != 3 || list[0].Text != "troisieme" || list[2].Text != "premier" {
		t.Fatalf("ordre inattendu : %+v", list)
	}
	if list[0].Handle != "bob" {
		t.Errorf("handle = %q, veut bob", list[0].Handle)
	}
	if got := s.List(2); len(got) != 2 || got[0].Text != "troisieme" {
		t.Errorf("List(2) = %+v, veut les 2 plus récents", got)
	}
}

func TestPostRejectsEmpty(t *testing.T) {
	s, _ := Open("")
	for _, txt := range []string{"", "   ", "\t\n", string([]byte{1, 2, 3})} {
		if _, err := s.Post("bob", txt); err == nil {
			t.Errorf("Post(%q) accepté, veut refus (message vide)", txt)
		}
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, veut 0", s.Count())
	}
}

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"  bonjour  ":        "bonjour",
		"a\tb\nc":            "a b c", // blancs de contrôle → espace unique
		"trop     d'espaces": "trop d'espaces",
		"abcédef":            "abcdef", // non-ASCII écarté sans laisser de trou
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, veut %q", in, got, want)
		}
	}
	// Longueur bornée.
	long := ""
	for i := 0; i < MaxText+50; i++ {
		long += "x"
	}
	if got := Sanitize(long); len(got) != MaxText {
		t.Errorf("Sanitize borne à %d, obtenu %d", MaxText, len(got))
	}
}

func TestPostHandleFallback(t *testing.T) {
	s, _ := Open("")
	m, err := s.Post("", "coucou")
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if m.Handle != "Anonyme" {
		t.Errorf("handle vide -> %q, veut Anonyme", m.Handle)
	}
}

func TestCapEviction(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	for i := 0; i < MaxMessages+10; i++ {
		if _, err := s.Post("bob", "msg"); err != nil {
			t.Fatalf("Post #%d: %v", i, err)
		}
	}
	if s.Count() != MaxMessages {
		t.Fatalf("Count = %d, veut plafond %d", s.Count(), MaxMessages)
	}
}

func TestPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wall.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.Post("alice", "salut le mur"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	// Recharge depuis le disque : le message doit persister.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	list := s2.List(0)
	if len(list) != 1 || list[0].Text != "salut le mur" || list[0].Handle != "alice" {
		t.Fatalf("rechargement inattendu : %+v", list)
	}
}

func TestReloadAppliesCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wall.json")
	s, _ := Open(path)
	s.now = fixedClock()
	for i := 0; i < MaxMessages+5; i++ {
		if _, err := s.Post("bob", "x"); err != nil {
			t.Fatalf("Post: %v", err)
		}
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	if s2.Count() != MaxMessages {
		t.Errorf("Count après rechargement = %d, veut %d", s2.Count(), MaxMessages)
	}
}
