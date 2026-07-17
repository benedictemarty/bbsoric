package forum

import (
	"path/filepath"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(time.Minute); return t }
}

func TestNewThreadAndList(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	th, err := s.NewThread("alice", "Bonjour Oric", "Premier message")
	if err != nil {
		t.Fatalf("NewThread: %v", err)
	}
	if th.ID == 0 || len(th.Posts) != 1 {
		t.Fatalf("fil mal formé : %+v", th)
	}
	list := s.List()
	if len(list) != 1 || list[0].Title != "Bonjour Oric" || list[0].Posts != 1 {
		t.Fatalf("List inattendu : %+v", list)
	}
	if list[0].Author != "alice" {
		t.Errorf("auteur = %q", list[0].Author)
	}
}

func TestNewThreadRejectsEmpty(t *testing.T) {
	s, _ := Open("")
	if _, err := s.NewThread("bob", "", "corps"); err == nil {
		t.Error("titre vide accepté")
	}
	if _, err := s.NewThread("bob", "titre", "   "); err == nil {
		t.Error("corps vide accepté")
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, veut 0", s.Count())
	}
}

func TestReplyAndOrder(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	t1, _ := s.NewThread("alice", "Sujet A", "a1")
	t2, _ := s.NewThread("bob", "Sujet B", "b1")
	// Répond à t1 : il devient le plus récemment actif → en tête de List.
	if _, err := s.Reply(t1.ID, "carol", "a2"); err != nil {
		t.Fatalf("Reply: %v", err)
	}
	list := s.List()
	if list[0].ID != t1.ID {
		t.Errorf("tri par activité KO : tête = %d, veut %d", list[0].ID, t1.ID)
	}
	if list[0].Posts != 2 {
		t.Errorf("t1 devrait avoir 2 messages, a %d", list[0].Posts)
	}
	full, ok := s.Thread(t1.ID)
	if !ok || len(full.Posts) != 2 || full.Posts[1].Text != "a2" || full.Posts[1].Author != "carol" {
		t.Errorf("Thread(t1) inattendu : %+v", full)
	}
	_ = t2
}

func TestReplyUnknownThread(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Reply(999, "bob", "coucou"); err == nil {
		t.Error("réponse à un fil inexistant acceptée")
	}
}

func TestThreadCopyIsolated(t *testing.T) {
	s, _ := Open("")
	th, _ := s.NewThread("alice", "titre", "corps")
	cp, _ := s.Thread(th.ID)
	cp.Posts[0].Text = "modifié dehors"
	again, _ := s.Thread(th.ID)
	if again.Posts[0].Text != "corps" {
		t.Error("Thread ne renvoie pas une copie isolée")
	}
}

func TestSanitizeBounds(t *testing.T) {
	s, _ := Open("")
	longTitle := ""
	for i := 0; i < MaxTitle+20; i++ {
		longTitle += "T"
	}
	longBody := ""
	for i := 0; i < MaxText+50; i++ {
		longBody += "b"
	}
	th, err := s.NewThread("bob", longTitle, longBody)
	if err != nil {
		t.Fatalf("NewThread: %v", err)
	}
	if len(th.Title) != MaxTitle {
		t.Errorf("titre borné à %d, obtenu %d", MaxTitle, len(th.Title))
	}
	if len(th.Posts[0].Text) != MaxText {
		t.Errorf("corps borné à %d, obtenu %d", MaxText, len(th.Posts[0].Text))
	}
}

func TestPostsCapPerThread(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	th, _ := s.NewThread("alice", "sujet", "p0")
	for i := 0; i < MaxPostsPerThread+5; i++ {
		if _, err := s.Reply(th.ID, "bob", "x"); err != nil {
			t.Fatalf("Reply: %v", err)
		}
	}
	full, _ := s.Thread(th.ID)
	if len(full.Posts) != MaxPostsPerThread {
		t.Errorf("messages plafonnés à %d, obtenu %d", MaxPostsPerThread, len(full.Posts))
	}
}

func TestPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forum.json")
	s, _ := Open(path)
	s.now = fixedClock()
	th, _ := s.NewThread("alice", "Persistant", "corps")
	if _, err := s.Reply(th.ID, "bob", "reponse"); err != nil {
		t.Fatalf("Reply: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	full, ok := s2.Thread(th.ID)
	if !ok || full.Title != "Persistant" || len(full.Posts) != 2 {
		t.Fatalf("rechargement inattendu : %+v", full)
	}
	// Le compteur d'ID doit survivre : un nouveau fil ne réutilise pas l'ID.
	th2, _ := s2.NewThread("carol", "Autre", "x")
	if th2.ID == th.ID {
		t.Errorf("ID réutilisé après rechargement : %d", th2.ID)
	}
}

func TestThreadsCapEviction(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	for i := 0; i < MaxThreads+10; i++ {
		if _, err := s.NewThread("bob", "sujet", "corps"); err != nil {
			t.Fatalf("NewThread #%d: %v", i, err)
		}
	}
	if s.Count() != MaxThreads {
		t.Errorf("fils plafonnés à %d, obtenu %d", MaxThreads, s.Count())
	}
}
