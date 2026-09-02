package main

import "testing"

func TestTranslateKeysArrows(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		out  []byte
	}{
		{"CSI haut", []byte("\x1b[A"), []byte{oricUp}},
		{"CSI bas", []byte("\x1b[B"), []byte{oricDown}},
		{"CSI droite", []byte("\x1b[C"), []byte{oricRight}},
		{"CSI gauche", []byte("\x1b[D"), []byte{oricLeft}},
		{"SS3 haut", []byte("\x1bOA"), []byte{oricUp}},
		{"SS3 gauche", []byte("\x1bOD"), []byte{oricLeft}},
		{"octets ordinaires inchangés", []byte("1X q"), []byte("1X q")},
		{"flèche entourée de touches", []byte("a\x1b[Bb"), []byte{'a', oricDown, 'b'}},
		{"deux flèches", []byte("\x1b[A\x1b[A"), []byte{oricUp, oricUp}},
		{"ESC suivi d'une touche : ESC transmis (quitter serveur)", []byte("\x1bQ"), []byte{0x1b, 'Q'}},
		{"CSI non-flèche ignorée (Suppr ESC[3~)", []byte("\x1b[3~x"), []byte{'x'}},
		{"quitKey préservé", []byte{quitKey}, []byte{quitKey}},
	}
	for _, c := range cases {
		out, rest := translateKeys(c.in)
		if len(rest) != 0 {
			t.Errorf("%s: rest inattendu %v", c.name, rest)
		}
		if string(out) != string(c.out) {
			t.Errorf("%s: out = %v, attendu %v", c.name, out, c.out)
		}
	}
}

// Un ESC final seul est ambigu (début possible de flèche) : mis en attente via rest,
// pas perdu — il sera transmis (ou traduit) dès la lecture suivante.
func TestTranslateKeysTrailingEsc(t *testing.T) {
	out, rest := translateKeys([]byte("\x1b"))
	if len(out) != 0 || string(rest) != "\x1b" {
		t.Fatalf("ESC final: out=%v rest=%v (attendu out vide, rest=ESC)", out, rest)
	}
}

// Une séquence de flèche scindée sur deux lectures stdin doit être recollée via rest.
func TestTranslateKeysSplitSequence(t *testing.T) {
	// 1er chunk : ESC seul.
	out, rest := translateKeys([]byte("\x1b"))
	if len(out) != 0 || string(rest) != "\x1b" {
		t.Fatalf("chunk1: out=%v rest=%v", out, rest)
	}
	// 2e chunk recollé : rest + "[A" -> flèche haut.
	out, rest = translateKeys(append(append([]byte(nil), rest...), []byte("[A")...))
	if string(out) != string([]byte{oricUp}) || len(rest) != 0 {
		t.Fatalf("chunk2: out=%v rest=%v", out, rest)
	}

	// Coupure entre '[' et 'A'.
	out, rest = translateKeys([]byte("\x1b["))
	if len(out) != 0 || string(rest) != "\x1b[" {
		t.Fatalf("csi partiel: out=%v rest=%v", out, rest)
	}
	out, _ = translateKeys(append(append([]byte(nil), rest...), 'C'))
	if string(out) != string([]byte{oricRight}) {
		t.Fatalf("csi recollé: out=%v", out)
	}
}
