package bbs

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestPaceHiresChunksInOrder(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDE"), 100) // 500 octets
	var got bytes.Buffer
	var slept []time.Duration
	write := func(s string) error { got.WriteString(s); return nil }
	sleep := func(d time.Duration) { slept = append(slept, d) }

	if err := paceHires(write, data, 700, 240, sleep); err != nil {
		t.Fatalf("paceHires: %v", err)
	}
	// Tout est écrit, dans l'ordre.
	if !bytes.Equal(got.Bytes(), data) {
		t.Fatal("données écrites ≠ données d'entrée")
	}
	// 500 octets / 240 -> 3 tronçons (240, 240, 20) -> 3 attentes.
	if len(slept) != 3 {
		t.Fatalf("nb d'attentes = %d, veut 3", len(slept))
	}
	// Le dernier tronçon (20 o) attend moins que les pleins (240 o).
	if slept[2] >= slept[0] {
		t.Errorf("dernier tronçon devrait attendre moins : %v vs %v", slept[2], slept[0])
	}
	// Débit respecté : 240 o à 700 o/s ≈ 342 ms.
	want := time.Duration(240) * time.Second / 700
	if slept[0] != want {
		t.Errorf("attente d'un tronçon plein = %v, veut %v", slept[0], want)
	}
}

func TestPaceHiresNoRateNoSleep(t *testing.T) {
	var got bytes.Buffer
	slept := 0
	write := func(s string) error { got.WriteString(s); return nil }
	sleep := func(time.Duration) { slept++ }
	if err := paceHires(write, []byte("hello"), 0, 240, sleep); err != nil {
		t.Fatal(err)
	}
	if slept != 0 {
		t.Errorf("débit ≤ 0 ne doit pas dormir, a dormi %d fois", slept)
	}
	if got.String() != "hello" {
		t.Errorf("écrit %q, veut hello", got.String())
	}
}

func TestPaceHiresPropagatesWriteError(t *testing.T) {
	boom := errors.New("connexion coupée")
	calls := 0
	write := func(string) error { calls++; return boom }
	if err := paceHires(write, bytes.Repeat([]byte{1}, 500), 700, 240, func(time.Duration) {}); err != boom {
		t.Fatalf("erreur = %v, veut %v", err, boom)
	}
	if calls != 1 {
		t.Errorf("doit s'arrêter au 1ᵉʳ échec (appels=%d)", calls)
	}
}

func TestPaceHiresEmpty(t *testing.T) {
	calls := 0
	if err := paceHires(func(string) error { calls++; return nil }, nil, 700, 240, func(time.Duration) {}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Errorf("flux vide : aucune écriture attendue, a %d", calls)
	}
}
