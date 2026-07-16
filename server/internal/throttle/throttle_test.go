package throttle

import (
	"testing"
	"time"
)

func TestAllowsUntilMaxThenBlocks(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	l := New(3, 5*time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allowed("ip1") {
			t.Fatalf("tentative %d : devrait être autorisée avant d'atteindre le max", i+1)
		}
		l.Fail("ip1")
	}
	if l.Allowed("ip1") {
		t.Errorf("après 3 échecs, l'IP doit être bloquée")
	}
	// Une autre clé n'est pas affectée.
	if !l.Allowed("ip2") {
		t.Errorf("une IP distincte ne doit pas être bloquée")
	}
}

func TestWindowSlidesAndFrees(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	l := New(3, 5*time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		l.Fail("ip1")
	}
	if l.Allowed("ip1") {
		t.Fatalf("bloquée après 3 échecs")
	}
	// Au-delà de la fenêtre, les échecs expirent et l'IP redevient autorisée.
	now = now.Add(5*time.Minute + time.Second)
	if !l.Allowed("ip1") {
		t.Errorf("après expiration de la fenêtre, l'IP doit être réautorisée")
	}
}

func TestResetClearsFailures(t *testing.T) {
	l := New(3, 5*time.Minute)
	for i := 0; i < 3; i++ {
		l.Fail("ip1")
	}
	if l.Allowed("ip1") {
		t.Fatalf("bloquée après 3 échecs")
	}
	l.Reset("ip1") // ex. connexion réussie
	if !l.Allowed("ip1") {
		t.Errorf("après Reset, l'IP doit être réautorisée")
	}
}

func TestNilAndZeroAreNoop(t *testing.T) {
	var l *Limiter // nil
	if !l.Allowed("x") {
		t.Errorf("un limiteur nil doit tout autoriser")
	}
	l.Fail("x") // ne doit pas paniquer
	l.Reset("x")

	z := New(0, time.Minute)
	z.Fail("x")
	if !z.Allowed("x") {
		t.Errorf("un limiteur de capacité 0 doit tout autoriser")
	}
}
