package pm

import (
	"path/filepath"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(time.Minute); return t }
}

func TestSendAndInbox(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	if _, err := s.Send("alice", "Bob", "coucou"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := s.Send("carol", "bob", "salut"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// Insensible à la casse : « Bob », « bob » → même boîte.
	box := s.Inbox("BOB")
	if len(box) != 2 {
		t.Fatalf("Inbox = %d, veut 2", len(box))
	}
	// Antéchronologique : le plus récent (« salut ») en tête.
	if box[0].Text != "salut" || box[1].Text != "coucou" {
		t.Errorf("ordre inattendu : %+v", box)
	}
	if s.Unread("bob") != 2 {
		t.Errorf("Unread = %d, veut 2", s.Unread("bob"))
	}
}

func TestSendRejectsEmpty(t *testing.T) {
	s, _ := Open("")
	if _, err := s.Send("alice", "bob", "   "); err == nil {
		t.Error("corps vide accepté")
	}
	if _, err := s.Send("alice", "", "coucou"); err == nil {
		t.Error("destinataire vide accepté")
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, veut 0", s.Count())
	}
}

func TestMarkRead(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	m, _ := s.Send("alice", "bob", "coucou")
	if s.Unread("bob") != 1 {
		t.Fatalf("Unread avant lecture = %d, veut 1", s.Unread("bob"))
	}
	if err := s.MarkRead("bob", m.ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if s.Unread("bob") != 0 {
		t.Errorf("Unread après lecture = %d, veut 0", s.Unread("bob"))
	}
	// Re-marquer est un no-op sans erreur.
	if err := s.MarkRead("bob", m.ID); err != nil {
		t.Errorf("re-MarkRead: %v", err)
	}
	// Marquer pour un autre destinataire échoue (isolation des boîtes).
	if err := s.MarkRead("carol", m.ID); err == nil {
		t.Error("MarkRead a autorisé un autre destinataire")
	}
}

func TestInboxIsolation(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	s.Send("alice", "bob", "pour bob")
	s.Send("alice", "carol", "pour carol")
	if len(s.Inbox("bob")) != 1 || s.Inbox("bob")[0].Text != "pour bob" {
		t.Errorf("boîte de bob incorrecte : %+v", s.Inbox("bob"))
	}
	if len(s.Inbox("carol")) != 1 || s.Inbox("carol")[0].Text != "pour carol" {
		t.Errorf("boîte de carol incorrecte : %+v", s.Inbox("carol"))
	}
}

func TestPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pm.json")
	s, _ := Open(path)
	s.now = fixedClock()
	m, _ := s.Send("alice", "bob", "persistant")
	if err := s.MarkRead("bob", m.ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	box := s2.Inbox("bob")
	if len(box) != 1 || box[0].Text != "persistant" || !box[0].Read {
		t.Fatalf("rechargement inattendu : %+v", box)
	}
	// Le compteur d'ID survit.
	m2, _ := s2.Send("carol", "bob", "autre")
	if m2.ID == m.ID {
		t.Errorf("ID réutilisé après rechargement : %d", m2.ID)
	}
}

func TestGlobalCap(t *testing.T) {
	s, _ := Open("")
	s.now = fixedClock()
	for i := 0; i < MaxMessages+10; i++ {
		if _, err := s.Send("alice", "bob", "x"); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}
	if s.Count() != MaxMessages {
		t.Errorf("messages plafonnés à %d, obtenu %d", MaxMessages, s.Count())
	}
}
