package oascii

import (
	"bytes"
	"testing"
)

// applyHiresStream rejoue un flux HIRES (1F FC … HiEnd) sur une VRAM, comme le
// ferait le terminal : HiOn efface à 0x40, HiBlit écrit len octets (RLE) à offset.
// Renvoie la VRAM résultante. Sert à prouver que les diffs reconstruisent l'état.
func applyHiresStream(vram []byte, stream []byte) []byte {
	out := make([]byte, len(vram))
	copy(out, vram)
	i := 0
	// En-tête 1F FC.
	if len(stream) < 2 || stream[0] != PlotByte || stream[1] != hiresByte {
		panic("flux HIRES sans en-tête 1F FC")
	}
	i = 2
	for i < len(stream) {
		op := stream[i]
		i++
		switch op {
		case HiEnd:
			return out
		case HiOn:
			for j := range out {
				out[j] = hiresClearByte
			}
		case HiBlit:
			off := int(stream[i]) | int(stream[i+1])<<8
			n := int(stream[i+2]) | int(stream[i+3])<<8
			i += 4
			// Décode n octets RLE à partir de stream[i:].
			dec := make([]byte, 0, n)
			for len(dec) < n {
				cnt := int(stream[i])
				val := stream[i+1]
				i += 2
				for k := 0; k < cnt; k++ {
					dec = append(dec, val)
				}
			}
			copy(out[off:off+n], dec)
		default:
			panic("opcode inattendu dans le flux différentiel")
		}
	}
	return out
}

func TestSetPixelLayout(t *testing.T) {
	h := NewHiresScreen()
	h.SetPixel(0, 0) // cellule 0, bit 5
	if got := h.buf[0]; got != hiresClearByte|0x20 {
		t.Errorf("pixel (0,0) = %#02x, veut %#02x", got, hiresClearByte|0x20)
	}
	h.SetPixel(5, 0) // cellule 0, bit 0
	if got := h.buf[0]; got != hiresClearByte|0x20|0x01 {
		t.Errorf("pixel (5,0) = %#02x, veut %#02x", got, hiresClearByte|0x20|0x01)
	}
	h.SetPixel(0, 1) // ligne 1, cellule 0 -> offset 40, bit 5
	if got := h.buf[40]; got != hiresClearByte|0x20 {
		t.Errorf("pixel (0,1) offset 40 = %#02x, veut %#02x", got, hiresClearByte|0x20)
	}
	h.SetPixel(6, 1) // ligne 1, cellule 1 -> offset 41, bit 5
	if got := h.buf[41]; got != hiresClearByte|0x20 {
		t.Errorf("pixel (6,1) offset 41 = %#02x, veut %#02x", got, hiresClearByte|0x20)
	}
}

func TestSetPixelOutOfBounds(t *testing.T) {
	h := NewHiresScreen()
	before := append([]byte(nil), h.buf...)
	h.SetPixel(-1, 0)
	h.SetPixel(240, 0)
	h.SetPixel(0, 200)
	if !bytes.Equal(before, h.buf) {
		t.Error("un pixel hors champ a modifié la VRAM")
	}
}

func TestHiresScreenFirstFrameEmitsHiOn(t *testing.T) {
	h := NewHiresScreen()
	h.SetPixel(10, 10)
	out := h.Render()
	if len(out) < 3 || out[0] != PlotByte || out[1] != hiresByte || out[2] != HiOn {
		t.Fatalf("1ʳᵉ image doit débuter par 1F FC HiOn, a %v", out[:min(3, len(out))])
	}
	if out[len(out)-1] != HiEnd {
		t.Errorf("le flux doit finir par HiEnd, a %#02x", out[len(out)-1])
	}
	if !bytes.Contains(out, []byte{HiBlit}) {
		t.Error("le flux devrait contenir au moins un HiBlit")
	}
}

func TestHiresScreenDiffIsSmall(t *testing.T) {
	h := NewHiresScreen()
	// Fond COMPLEXE (peu compressible en RLE) : c'est le cas où le différentiel est
	// utile — réémettre tout le fond à chaque image coûterait cher.
	bg := make([]byte, hiresVRAMSize)
	for i := range bg {
		bg[i] = 0x40 | byte((i*37+11)&0x3F) // bit 6 posé, 6 pixels variés
	}
	h.SetBitmap(bg)
	f1 := h.Render() // image pleine : gros flux

	// Image 2 : même fond + un petit sprite localisé → seule cette zone change.
	h.SetBitmap(bg)
	h.FillBox(100, 100, 112, 112)
	f2 := h.Render()

	// Une image suivante ne réémet PAS HiOn : le 1ᵉʳ opcode après 1F FC est HiBlit.
	if f2[2] == HiOn {
		t.Error("une image suivante ne doit PAS réémettre HiOn")
	}
	if f2[2] != HiBlit {
		t.Errorf("1ᵉʳ opcode du diff = %#02x, veut HiBlit", f2[2])
	}
	if len(f2) >= len(f1)/4 {
		t.Errorf("un diff localisé (%d o) devrait être bien plus petit que l'image pleine (%d o)", len(f2), len(f1))
	}
}

func TestHiresScreenNoChangeReturnsNil(t *testing.T) {
	h := NewHiresScreen()
	h.FillBox(0, 0, 50, 50)
	_ = h.Render()
	// Sans modification, la 2ᵉ image est vide.
	if out := h.Render(); out != nil {
		t.Errorf("image inchangée devrait renvoyer nil, a %d octets", len(out))
	}
}

// TestHiresScreenRoundTrip : appliquer image 1 puis le diff de l'image 2 sur une
// VRAM vierge doit reconstruire EXACTEMENT la composition de l'image 2 — preuve
// que le rendu différentiel est correct (effacement + tracé).
func TestHiresScreenRoundTrip(t *testing.T) {
	h := NewHiresScreen()
	h.Circle(120, 100, 40)
	h.FillBox(10, 10, 30, 30)
	f1 := h.Render()

	vram := make([]byte, hiresVRAMSize) // état terminal supposé quelconque
	vram = applyHiresStream(vram, f1)
	if !bytes.Equal(vram, h.buf) {
		t.Fatal("après image 1, la VRAM reconstruite diffère de la composition")
	}

	// Image 2 : scène différente (objet déplacé + forme retirée).
	h.Clear()
	h.Circle(130, 90, 40)
	f2 := h.Render()
	vram = applyHiresStream(vram, f2)
	if !bytes.Equal(vram, h.buf) {
		t.Fatal("après le diff image 2, la VRAM reconstruite diffère de la composition")
	}
}

func TestHiresScreenReset(t *testing.T) {
	h := NewHiresScreen()
	h.SetPixel(1, 1)
	_ = h.Render()
	h.Reset()
	h.SetPixel(1, 1)
	out := h.Render()
	if len(out) < 3 || out[2] != HiOn {
		t.Error("après Reset, la prochaine image doit réémettre HiOn")
	}
}

func TestHiresLineEndpoints(t *testing.T) {
	h := NewHiresScreen()
	h.Line(0, 0, 10, 0) // horizontale
	for x := 0; x <= 10; x++ {
		if h.buf[x/6]&(1<<uint(5-x%6)) == 0 {
			t.Errorf("pixel (%d,0) de la ligne horizontale non allumé", x)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestHiresScreenAnimationSequence simule la boucle de l'applet hiresanim (cadre
// fixe + balle qui se déplace) sur plusieurs images et vérifie les propriétés du
// rendu différentiel : HiOn UNE seule fois (1ʳᵉ image), images suivantes non vides
// (la balle bouge) et SANS HiOn, et reconstruction VRAM exacte en rejouant tout.
func TestHiresScreenAnimationSequence(t *testing.T) {
	h := NewHiresScreen()
	vram := make([]byte, hiresVRAMSize)

	drawFrame := func(cx, cy int) {
		h.Clear()
		h.Box(0, 0, HiresPixW-1, HiresPixH-1) // cadre fixe
		h.Circle(cx, cy, 8)                   // balle
	}

	// Image 0 : émet HiOn + le contenu.
	drawFrame(40, 40)
	f0 := h.Render()
	if f0[2] != HiOn {
		t.Fatalf("image 0 doit commencer par HiOn, a %#02x", f0[2])
	}
	vram = applyHiresStream(vram, f0)
	if !bytes.Equal(vram, h.buf) {
		t.Fatal("VRAM ≠ composition après image 0")
	}

	// Images suivantes : balle déplacée → diff non vide, jamais de HiOn.
	positions := [][2]int{{43, 42}, {46, 44}, {49, 46}, {52, 48}}
	for i, p := range positions {
		drawFrame(p[0], p[1])
		f := h.Render()
		if f == nil {
			t.Fatalf("image %d : diff vide alors que la balle a bougé", i+1)
		}
		if f[2] == HiOn {
			t.Errorf("image %d : ne doit pas réémettre HiOn", i+1)
		}
		vram = applyHiresStream(vram, f)
		if !bytes.Equal(vram, h.buf) {
			t.Fatalf("image %d : VRAM reconstruite ≠ composition", i+1)
		}
	}
}
