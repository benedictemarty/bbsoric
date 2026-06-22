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
