package render

import (
	"bytes"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

func TestHiresNil(t *testing.T) {
	if got := Hires(&content.Page{}); got != nil {
		t.Errorf("page sans hires : attendu nil, obtenu %v", got)
	}
}

func TestHiresPrimitives(t *testing.T) {
	p := &content.Page{Hires: &content.Hires{Draw: []content.HiresOp{
		{Op: "ink", C: 3},
		{Op: "curset", X: 10, Y: 20},
		{Op: "line", X: 230, Y: 190},
		{Op: "circle", R: 30},
		{Op: "char", X: 5, Y: 6, Ch: "A"},
	}}}
	want := []byte{
		oascii.PlotByte, 0xFC, oascii.HiOn, // ouverture + bascule
		oascii.HiInk, 3,
		oascii.HiCurset, 10, 20,
		oascii.HiLine, 230, 190,
		oascii.HiCircle, 30,
		oascii.HiChar, 5, 6, 'A',
		oascii.HiEnd,
	}
	if got := Hires(p); !bytes.Equal(got, want) {
		t.Errorf("flux primitives inattendu\n got=%v\nwant=%v", got, want)
	}
}

func TestHiresBitmapBlit(t *testing.T) {
	bg := make([]byte, content.HiresBitmapSize)
	for i := range bg {
		bg[i] = 0x40 // plage uniforme → bien compressée
	}
	bg[0] = 0x7F // une variation en tête
	p := &content.Page{Hires: &content.Hires{Background: bg}}
	out := Hires(p)

	// En-tête attendu : 1F FC HiOn HiBlit off_lo off_hi len_lo len_hi …
	n := content.HiresBitmapSize
	head := []byte{oascii.PlotByte, 0xFC, oascii.HiOn, oascii.HiBlit, 0, 0, byte(n & 0xFF), byte(n >> 8)}
	if !bytes.HasPrefix(out, head) {
		t.Fatalf("en-tête Blit incorrect : %v", out[:len(head)])
	}
	if out[len(out)-1] != oascii.HiEnd {
		t.Fatalf("flux non terminé par HiEnd")
	}
	// Le corps RLE doit redécoder exactement le bitmap.
	rle := out[len(head) : len(out)-1]
	if dec := oascii.RLEDecode(rle, n); !bytes.Equal(dec, bg) {
		t.Errorf("le Blit ne redécode pas le bitmap source")
	}
	// Et il doit être nettement plus court que les 8000 octets bruts.
	if len(rle) > 200 {
		t.Errorf("Blit RLE trop long (%d octets) pour une plage quasi uniforme", len(rle))
	}
}
