package hires

import (
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// feed injecte une suite d'octets dans le rasteriseur.
func feed(r *Raster, b ...byte) {
	for _, x := range b {
		r.Feed(x)
	}
}

func TestPointEtCurset(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiPoint, 10, 20)
	if !r.Pixel(10, 20) {
		t.Error("pixel (10,20) devrait être allumé")
	}
	if r.Pixel(11, 20) {
		t.Error("pixel voisin ne devrait pas être allumé")
	}
	feed(r, oascii.HiCurset, 100, 50) // déplace sans tracer
	if r.Pixel(100, 50) {
		t.Error("curset ne doit pas tracer")
	}
}

func TestLigneHorizontaleEtVerticale(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiCurset, 10, 30)
	feed(r, oascii.HiLine, 20, 30) // horizontale y=30 de x=10 à 20
	for x := 10; x <= 20; x++ {
		if !r.Pixel(x, 30) {
			t.Fatalf("pixel (%d,30) manquant sur l'horizontale", x)
		}
	}
	feed(r, oascii.HiLine, 20, 40) // verticale x=20 de y=30 à 40
	for y := 30; y <= 40; y++ {
		if !r.Pixel(20, y) {
			t.Fatalf("pixel (20,%d) manquant sur la verticale", y)
		}
	}
}

func TestBoxEtFillBox(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiCurset, 10, 10)
	feed(r, oascii.HiBox, 20, 15)
	// Coins et milieux de côtés présents.
	for _, p := range [][2]int{{10, 10}, {20, 10}, {10, 15}, {20, 15}, {15, 10}, {15, 15}, {10, 12}, {20, 12}} {
		if !r.Pixel(p[0], p[1]) {
			t.Errorf("box : pixel (%d,%d) manquant", p[0], p[1])
		}
	}
	// L'intérieur du cadre reste vide.
	if r.Pixel(15, 12) {
		t.Error("box : l'intérieur devrait être vide")
	}

	r2 := New()
	feed(r2, oascii.HiOn)
	feed(r2, oascii.HiCurset, 10, 10)
	feed(r2, oascii.HiFillBox, 20, 15)
	if !r2.Pixel(15, 12) {
		t.Error("fillbox : l'intérieur devrait être plein")
	}
}

func TestCircle(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiCurset, 120, 100)
	feed(r, oascii.HiCircle, 30)
	// Les 4 points cardinaux du cercle de rayon 30.
	for _, p := range [][2]int{{150, 100}, {90, 100}, {120, 130}, {120, 70}} {
		if !r.Pixel(p[0], p[1]) {
			t.Errorf("circle : point cardinal (%d,%d) manquant", p[0], p[1])
		}
	}
	// Le centre n'est pas tracé.
	if r.Pixel(120, 100) {
		t.Error("circle : le centre ne doit pas être tracé")
	}
}

func TestModeCouleur(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiInk, byte(oascii.Red)) // encre rouge -> mode couleur
	feed(r, oascii.HiCurset, 60, 40)
	feed(r, oascii.HiLine, 120, 40) // ligne y=40
	// Un pixel de la ligne (x>=6) est rouge.
	if got := r.ColorAt(60, 40); got != byte(oascii.Red) {
		t.Errorf("couleur du pixel = %d, veut %d (rouge)", got, oascii.Red)
	}
	// La 1re cellule (x<6) est sacrifiée à l'attribut : pas de pixel.
	if r.Pixel(0, 40) || r.Pixel(5, 40) {
		t.Error("en mode couleur, x<6 ne doit pas être tracé (cellule d'attribut)")
	}
}

func TestBlitRLE(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	// Remplit les 40 premiers octets (ligne 0) de 0x7F (6 pixels allumés) via Blit.
	data := make([]byte, rowByte)
	for i := range data {
		data[i] = 0x7F // bit6 + 6 pixels
	}
	rle := oascii.RLEEncode(data)
	stream := []byte{oascii.HiBlit, 0x00, 0x00, byte(len(data) & 0xFF), byte(len(data) >> 8)}
	stream = append(stream, rle...)
	feed(r, stream...)
	for x := 0; x < W; x++ {
		if !r.Pixel(x, 0) {
			t.Fatalf("blit : pixel (%d,0) manquant", x)
		}
	}
	if r.Pixel(0, 1) {
		t.Error("blit : la ligne 1 ne devrait pas être touchée")
	}
}

func TestFluxHiEndEtRendu(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn, oascii.HiPoint, 0, 0)
	if r.Done() {
		t.Error("le flux ne doit pas être terminé avant HiEnd")
	}
	feed(r, oascii.HiEnd)
	if !r.Done() {
		t.Error("HiEnd doit terminer le flux")
	}
	ansi := r.RenderANSI()
	if !strings.Contains(ansi, "▀") {
		t.Error("le rendu ANSI devrait contenir des demi-blocs ▀")
	}
}

func TestClampHorsBornes(t *testing.T) {
	r := New()
	feed(r, oascii.HiOn)
	feed(r, oascii.HiPoint, 239, 199) // coin extrême valide
	if !r.Pixel(239, 199) {
		t.Error("le pixel (239,199) devrait être allumé")
	}
	// Point hors bornes : ignoré silencieusement (pas de panique).
	feed(r, oascii.HiPoint, 255, 255)
}
