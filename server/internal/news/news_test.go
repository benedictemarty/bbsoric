package news

import (
	"path/filepath"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(time.Minute); return t }
}

func TestPostAndListOrder(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	for _, txt := range []string{"un", "deux", "trois"} {
		if _, err := s.Post("sysop", txt, "corps "+txt); err != nil {
			t.Fatalf("Post(%q): %v", txt, err)
		}
	}
	list := s.List(0)
	if len(list) != 3 || list[0].Title != "trois" || list[2].Title != "un" {
		t.Fatalf("ordre antéchronologique KO : %+v", list)
	}
	if got := s.List(2); len(got) != 2 || got[0].Title != "trois" {
		t.Errorf("List(2) = %+v", got)
	}
}

func TestPostRejectsEmptyBody(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Post("sysop", "titre", "   "); err == nil {
		t.Error("corps vide accepté")
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, veut 0", s.Count())
	}
}

func TestPostDefaults(t *testing.T) {
	s, _ := Open("")
	it, err := s.Post("", "", "du contenu")
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if it.Title != "Annonce" {
		t.Errorf("titre par défaut = %q, veut Annonce", it.Title)
	}
	if it.Author != "sysop" {
		t.Errorf("auteur par défaut = %q, veut sysop", it.Author)
	}
}

func TestBounds(t *testing.T) {
	s, _ := Open("")
	longTitle := ""
	for i := 0; i < MaxTitle+20; i++ {
		longTitle += "T"
	}
	longBody := ""
	for i := 0; i < MaxBody+50; i++ {
		longBody += "b"
	}
	it, err := s.Post("sysop", longTitle, longBody)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if len(it.Title) != MaxTitle {
		t.Errorf("titre borné à %d, obtenu %d", MaxTitle, len(it.Title))
	}
	if len(it.Body) != MaxBody {
		t.Errorf("corps borné à %d, obtenu %d", MaxBody, len(it.Body))
	}
}

func TestCapEviction(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	for i := 0; i < MaxItems+10; i++ {
		if _, err := s.Post("sysop", "t", "corps"); err != nil {
			t.Fatalf("Post: %v", err)
		}
	}
	if s.Count() != MaxItems {
		t.Errorf("plafond = %d, obtenu %d", MaxItems, s.Count())
	}
}

func TestPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "news.json")
	s, _ := Open(path)
	s.now = fixedClock()
	if _, err := s.Post("sysop", "Ouverture", "Le BBS ouvre ses portes"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	list := s2.List(0)
	if len(list) != 1 || list[0].Title != "Ouverture" || list[0].Body != "Le BBS ouvre ses portes" {
		t.Fatalf("rechargement inattendu : %+v", list)
	}
}
