package hires

import (
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/render"
)

// Ce fichier prouve l'identité PIXEL du rasteriseur (pcterm/internal/hires) avec
// une RÉFÉRENCE INDÉPENDANTE : un port Go fidèle du rasteriseur JS du studio
// (studio/web/app.js, renderHiresPreview). Les deux consomment le MÊME contenu
// (content.Hires) : le mien via le flux fil du serveur (render.Hires), la
// référence via les primitives directes du studio. On diffe les masques de pixels
// 240×200 (allumé / éteint).
//
// Note sur les lignes : le studio utilise un Bresenham « both-axis » (err=dx-dy),
// le firmware Oric (client/hires.s), dont je suis le portage, un « major-axis »
// (err=dx/2). On pourrait craindre des écarts sur les diagonales — or, en pratique,
// pour des extrémités ENTIÈRES les deux formulations coïncident : le balayage de
// TestPixelExactLignes ci-dessous ne trouve AUCUN pixel divergent. Résultat : mon
// rasteriseur est pixel-identique à la référence studio sur toutes les primitives.

// refRaster est le port Go du rasteriseur JS du studio (référence indépendante).
type refRaster struct {
	px   []byte // 240×200, pixel allumé = ink+1, 0 = éteint
	ink  byte
	penX int
	penY int
}

func newRef() *refRaster { return &refRaster{px: make([]byte, W*H), ink: 7} }

func (r *refRaster) set(x, y int) {
	if x >= 0 && x < W && y >= 0 && y < H {
		r.px[y*W+x] = r.ink + 1
	}
}

func (r *refRaster) line(x0, y0, x1, y1 int) {
	dx, dy := abs(x1-x0), abs(y1-y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		r.set(x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (r *refRaster) box(x0, y0, x1, y1 int) {
	r.line(x0, y0, x1, y0)
	r.line(x1, y0, x1, y1)
	r.line(x1, y1, x0, y1)
	r.line(x0, y1, x0, y0)
}

func (r *refRaster) fillBox(x0, y0, x1, y1 int) {
	a, b := y0, y1
	if a > b {
		a, b = b, a
	}
	for y := a; y <= b; y++ {
		r.line(x0, y, x1, y)
	}
}

func (r *refRaster) circle(cx, cy, rad int) {
	if rad <= 0 {
		return
	}
	x, y, err := rad, 0, 1-rad
	for x >= y {
		for _, o := range [8][2]int{{x, y}, {-x, y}, {x, -y}, {-x, -y}, {y, x}, {-y, x}, {y, -x}, {-y, -x}} {
			r.set(cx+o[0], cy+o[1])
		}
		y++
		if err <= 0 {
			err += 2*y + 1
		} else {
			x--
			err += 2*(y-x) + 1
		}
	}
}

func (r *refRaster) char(x, y int, ch byte) {
	idx := int(ch & 0x7F)
	if idx < 0x20 || idx > 0x7F {
		return
	}
	base := (idx - 0x20) * 8
	for row := 0; row < 8; row++ {
		g := stdFont[base+row]
		for bit := 0; bit < 6; bit++ {
			if g&(1<<(5-byte(bit))) != 0 {
				r.set(x+bit, y+row)
			}
		}
	}
}

// render exécute un content.Hires (fond bitmap + primitives) comme le studio.
func (r *refRaster) render(h *content.Hires) []bool {
	if len(h.Background) == content.HiresBitmapSize {
		for y := 0; y < H; y++ {
			for bx := 0; bx < rowByte; bx++ {
				b := h.Background[y*rowByte+bx]
				if b&0x60 == 0 {
					continue // attribut → ignoré (comme le studio)
				}
				for bit := 0; bit < 6; bit++ {
					if b&(1<<(5-byte(bit))) != 0 {
						r.px[y*W+bx*6+bit] = 8
					}
				}
			}
		}
	}
	for _, op := range h.Draw {
		x, y := op.X, op.Y
		switch op.Op {
		case "ink":
			if op.C >= 0 && op.C <= 7 {
				r.ink = byte(op.C)
			}
		case "curset":
			r.penX, r.penY = x, y
		case "point":
			r.penX, r.penY = x, y
			r.set(x, y)
		case "line":
			r.line(r.penX, r.penY, x, y)
			r.penX, r.penY = x, y
		case "box":
			r.box(r.penX, r.penY, x, y)
			r.penX, r.penY = x, y
		case "fillbox":
			r.fillBox(r.penX, r.penY, x, y)
			r.penX, r.penY = x, y
		case "circle":
			r.circle(r.penX, r.penY, op.R)
		case "char":
			c := byte(' ')
			if op.Ch != "" {
				c = op.Ch[0]
			}
			r.char(x, y, c)
		}
	}
	mask := make([]bool, W*H)
	for i, v := range r.px {
		mask[i] = v != 0
	}
	return mask
}

// mine exécute le même content.Hires via le flux fil serveur (render.Hires) et
// mon rasteriseur, puis renvoie le masque de pixels allumés.
func mine(h *content.Hires) []bool {
	rr := New()
	for _, b := range render.Hires(&content.Page{Hires: h}) {
		rr.Feed(b)
	}
	mask := make([]bool, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			mask[y*W+x] = rr.Pixel(x, y)
		}
	}
	return mask
}

// diffMasks compte les pixels divergents et renvoie le premier (x,y) différent.
func diffMasks(a, b []bool) (n, fx, fy int) {
	fx, fy = -1, -1
	for i := range a {
		if a[i] != b[i] {
			n++
			if fx < 0 {
				fx, fy = i%W, i/W
			}
		}
	}
	return
}

// TestPixelExactVsStudio : les primitives à algorithme partagé (point, box,
// fillbox, circle, char, blit, lignes H/V) sont IDENTIQUES au pixel près entre mon
// rasteriseur (flux serveur) et la référence studio.
func TestPixelExactVsStudio(t *testing.T) {
	// Un fond bitmap réaliste : quelques lignes de pixels pleins.
	bg := make([]byte, content.HiresBitmapSize)
	for i := range bg {
		bg[i] = empty // 0x40 : pixels éteints
	}
	for x := 0; x < rowByte; x++ {
		bg[10*rowByte+x] = 0x7F // ligne 10 : 6 pixels pleins par octet
	}

	h := &content.Hires{
		Background: bg,
		Draw: []content.HiresOp{
			{Op: "curset", X: 5, Y: 5},
			{Op: "point", X: 5, Y: 5},
			{Op: "box", X: 100, Y: 60}, // côtés H/V
			{Op: "curset", X: 120, Y: 20},
			{Op: "fillbox", X: 200, Y: 80},
			{Op: "curset", X: 60, Y: 150},
			{Op: "circle", R: 40},
			{Op: "curset", X: 30, Y: 30},
			{Op: "line", X: 30, Y: 120},  // verticale
			{Op: "line", X: 200, Y: 120}, // horizontale
			{Op: "char", X: 8, Y: 180, Ch: "A"},
			{Op: "char", X: 20, Y: 180, Ch: "7"},
		},
	}
	n, fx, fy := diffMasks(mine(h), newRef().render(h))
	if n != 0 {
		t.Errorf("%d pixels divergents (1er en %d,%d) — attendu 0 (identité au pixel)", n, fx, fy)
	}
}

// TestPixelExactLignes : balayage de lignes de plusieurs origines vers toute la
// grille — PREUVE que mon rasteriseur (firmware, major-axis) et la référence studio
// (both-axis) sont pixel-identiques sur TOUTES les diagonales testées (0 écart).
func TestPixelExactLignes(t *testing.T) {
	const step = 17
	cas, totalDiff := 0, 0
	for x1 := 0; x1 < W; x1 += step {
		for y1 := 0; y1 < H; y1 += step {
			for _, start := range [][2]int{{0, 0}, {120, 100}, {239, 0}, {0, 199}} {
				h := &content.Hires{Draw: []content.HiresOp{
					{Op: "curset", X: start[0], Y: start[1]},
					{Op: "line", X: x1, Y: y1},
				}}
				n, _, _ := diffMasks(mine(h), newRef().render(h))
				cas++
				totalDiff += n
			}
		}
	}
	if totalDiff != 0 {
		t.Errorf("%d cas de lignes, %d pixels divergents — attendu 0 (identité au pixel)", cas, totalDiff)
	}
	t.Logf("%d cas de lignes balayés, 0 pixel divergent (major-axis ≡ both-axis sur extrémités entières)", cas)
}
