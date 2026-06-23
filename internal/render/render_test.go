package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

func TestScreenMenu(t *testing.T) {
	p := &content.Page{Title: "MENU", Entries: []content.Entry{
		{Key: "1", Label: "Infos", Target: "info"},
	}}
	out := string(Screen(p))
	if !strings.Contains(out, "MENU") || !strings.Contains(out, "Infos") || !strings.Contains(out, "Votre choix") {
		t.Errorf("rendu menu incomplet:\n%q", out)
	}
}

func TestScreenContent(t *testing.T) {
	p := &content.Page{Title: "INFOS", Lines: []content.Line{{Text: "Bonjour"}}}
	out := string(Screen(p))
	if !strings.Contains(out, "Bonjour") || !strings.Contains(out, "Appuyez sur une touche") {
		t.Errorf("rendu contenu incomplet:\n%q", out)
	}
}

func TestScreenSegmentsMulticolor(t *testing.T) {
	p := &content.Page{Title: "T", Lines: []content.Line{
		{Segments: []content.Span{
			{Text: "A", Style: content.Style{Ink: "yellow"}},
			{Text: "B", Style: content.Style{Ink: "white"}},
		}},
	}}
	out := Screen(p)
	if !bytes.Contains(out, []byte{oascii.InkAttr(oascii.Yellow), 'A'}) {
		t.Errorf("segment jaune 'A' attendu")
	}
	if !bytes.Contains(out, []byte{oascii.InkAttr(oascii.White), 'B'}) {
		t.Errorf("segment blanc 'B' attendu")
	}
}

func TestRawScreen(t *testing.T) {
	p := &content.Page{Raw: true, Lines: []content.Line{{Text: "AB"}, {Text: "CD"}}}
	out := string(RawScreen(p))
	// lignes telles quelles, pas de barre de titre ni d'invite, pas de NL final.
	if out != "AB\r\nCD" {
		t.Errorf("RawScreen = %q, attendu \"AB\\r\\nCD\"", out)
	}
	if strings.Contains(out, "=") || strings.Contains(out, "Appuyez") {
		t.Errorf("écran brut ne doit pas avoir de chrome:\n%q", out)
	}
}

func TestRawScreenBuffer(t *testing.T) {
	// buffer 40×28 : "Hi" en haut-gauche, le reste vide -> 1 ligne "Hi", élaguée.
	buf := make([]byte, 40*28)
	for i := range buf {
		buf[i] = 0x20
	}
	buf[0] = 'H'
	buf[1] = 'i'
	out := string(RawScreen(&content.Page{Raw: true, Screen: buf}))
	// la 1re ligne complète (40 cases) puis rien (lignes vides élaguées).
	if len(out) != 40 || out[0] != 'H' || out[1] != 'i' {
		t.Errorf("RawScreen(buffer) = %q (len %d)", out, len(out))
	}
	if strings.Contains(out, "\r\n") {
		t.Errorf("une seule ligne non vide : pas de CRLF attendu")
	}
}

func TestScreenInverseIsBit7(t *testing.T) {
	// inverse = bit 7 sur le caractère, PAS un attribut sériel.
	p := &content.Page{Title: "T", Lines: []content.Line{
		{Text: "X", Style: content.Style{Inverse: true}},
	}}
	out := Screen(p)
	if !bytes.Contains(out, []byte{'X' | 0x80}) {
		t.Errorf("inverse devrait poser le bit 7 sur 'X'")
	}
	if bytes.Contains(out, []byte{29}) {
		t.Errorf("aucun attribut inverse (octet 29) ne doit être émis")
	}
}

// TestWrapWidthAndColor : une ligne dépassant 40 colonnes est repliée (chaque
// ligne physique ≤ 40 cases) et la couleur/fond est ré-émise sur la ligne n+1.
func TestWrapWidthAndColor(t *testing.T) {
	ln := content.Line{
		Text:  strings.Repeat("mot ", 15), // 60 caractères
		Style: content.Style{Ink: "red", Paper: "blue"},
	}
	b := oascii.New()
	emitLineWrapped(b, ln)
	out := b.String()

	rows := strings.Split(out, "\r\n")
	if len(rows) < 2 {
		t.Fatalf("la ligne longue doit être repliée (>1 ligne physique):\n%q", out)
	}
	for i, r := range rows {
		if len(r) > oascii.Cols {
			t.Errorf("ligne physique %d > 40 cases (%d):\n%q", i, len(r), r)
		}
	}
	// paper bleu = 0x14, ink rouge = 0x01 : chaque ligne non vide doit commencer
	// par le fond ré-émis (puis l'encre).
	paper := byte(0x10 | int(oascii.Blue)) // 0x14
	for i, r := range rows {
		if r == "" {
			continue
		}
		if r[0] != paper {
			t.Errorf("ligne %d ne ré-émet pas le fond (0x%02x) : %q", i, paper, r)
		}
	}
}
