package oascii

import (
	"bytes"
	"testing"
)

// Les valeurs attendues proviennent directement du décodeur ULA de l'émulateur
// de référence (Oric1/oric1-emu, src/video/video.c : decode_attr).

func TestInkAttr(t *testing.T) {
	cases := map[Color]byte{Black: 0x00, Red: 0x01, Green: 0x02, Yellow: 0x03,
		Blue: 0x04, Magenta: 0x05, Cyan: 0x06, White: 0x07}
	for c, want := range cases {
		if got := InkAttr(c); got != want {
			t.Errorf("InkAttr(%d) = %#02x, want %#02x", c, got, want)
		}
	}
}

func TestPaperAttr(t *testing.T) {
	// 16–23 : 0x10 | couleur
	if got := PaperAttr(Black); got != 0x10 {
		t.Errorf("PaperAttr(Black) = %#02x, want 0x10", got)
	}
	if got := PaperAttr(Cyan); got != 0x16 { // 16 + 6
		t.Errorf("PaperAttr(Cyan) = %#02x, want 0x16", got)
	}
	if got := PaperAttr(White); got != 0x17 {
		t.Errorf("PaperAttr(White) = %#02x, want 0x17", got)
	}
}

func TestTextAttr(t *testing.T) {
	// groupe 8–15 ; bit0=alt, bit1=double, bit2=blink
	if got := TextAttr(false, false, false); got != 0x08 {
		t.Errorf("TextAttr(0,0,0) = %#02x, want 0x08", got)
	}
	if got := TextAttr(false, false, true); got != 0x09 { // alt
		t.Errorf("alt = %#02x, want 0x09", got)
	}
	if got := TextAttr(false, true, false); got != 0x0A { // double
		t.Errorf("double = %#02x, want 0x0A", got)
	}
	if got := TextAttr(true, false, false); got != 0x0C { // blink
		t.Errorf("blink = %#02x, want 0x0C", got)
	}
	if got := TextAttr(true, true, true); got != 0x0F {
		t.Errorf("all = %#02x, want 0x0F", got)
	}
}

func TestBuilderStream(t *testing.T) {
	got := New().Ink(Yellow).Paper(Blue).Text("OK").Bytes()
	want := []byte{0x03, 0x14, 'O', 'K'} // ink jaune(3), paper bleu(16+4=20=0x14)
	if !bytes.Equal(got, want) {
		t.Errorf("flux = % x, want % x", got, want)
	}
}

func TestNonPrintableBecomesSpace(t *testing.T) {
	// un octet de contrôle dans Text ne doit pas être pris pour un attribut
	got := New().Text("A\x01B").Bytes()
	want := []byte{'A', ' ', 'B'}
	if !bytes.Equal(got, want) {
		t.Errorf("flux = % x, want % x", got, want)
	}
}

func TestStickyReemitsAfterNewline(t *testing.T) {
	b := New().Sticky(true).Ink(Red)
	b.Text("a").Newline().Text("b")
	got := b.Bytes()
	// a : [ink red][a][CR][LF][ink red (réémis)][b]
	want := []byte{0x01, 'a', '\r', '\n', 0x01, 'b'}
	if !bytes.Equal(got, want) {
		t.Errorf("flux = % x, want % x", got, want)
	}
}

func TestNoStickyNoReemit(t *testing.T) {
	b := New().Ink(Red) // sticky désactivé par défaut
	b.Text("a").Newline().Text("b")
	got := b.Bytes()
	want := []byte{0x01, 'a', '\r', '\n', 'b'}
	if !bytes.Equal(got, want) {
		t.Errorf("flux = % x, want % x", got, want)
	}
}
