package oascii

// HiresScreen est le pendant HIRES de Screen : un buffer d'écran GRAPHIQUE
// « intelligent » (VRAM 240×200, 8000 octets) qui maintient l'état COMPOSÉ (ce
// qu'on veut afficher) et l'état AFFICHÉ (ce que le terminal montre déjà).
// Render() ne produit QUE le flux nécessaire pour passer de l'affiché au composé :
// les octets VRAM inchangés ne sont pas réémis.
//
// C'est le socle de l'ANIMATION HIRES : sur la liaison série (9600 bauds), pousser
// tout le fond (8000 octets, même compressé) à chaque image est lent ; un diff de
// quelques octets est quasi instantané. Le serveur rastérise chaque image dans la
// VRAM composée puis appelle Render() — la boucle d'animation recompose (Clear +
// tracé) à chaque image, et le diff gère naturellement l'EFFACEMENT (l'ancienne
// position revient à l'octet vide, la nouvelle s'allume).
//
// Aucun changement firmware : le diff est émis avec l'opcode HIRES existant
// **HiBlit** (écrit N octets à $A000+offset), déjà interprété par client/hires.s.
// La première image émet HiOn (bascule HIRES + efface) ; les suivantes n'émettent
// que des HiBlit (le terminal reste en HIRES entre les images).
//
// Modèle VRAM (identique à hires.s) : 40 octets/ligne, 6 pixels/octet ; un octet
// « vide » vaut 0x40 (bit 6 à 1, aucun pixel) ; le pixel x d'une cellule est le
// bit (5 − x mod 6). Rendu MONOCHROME (encre par défaut) — la couleur par ligne
// reste au modèle « page statique » (render.Hires).
// Dimensions HIRES exportées (pour les auteurs d'animations côté serveur).
const (
	HiresPixW = 240 // largeur en pixels
	HiresPixH = 200 // hauteur en pixels (lignes)
)

const (
	hiresBytesPerRow = 40
	hiresRows        = 200
	hiresVRAMSize    = hiresBytesPerRow * hiresRows // 8000
	hiresPixW        = HiresPixW
	hiresPixH        = HiresPixH
	hiresClearByte   = 0x40 // bit 6 = 1, aucun pixel allumé
)

// HiresScreen porte la VRAM composée et la dernière VRAM émise.
type HiresScreen struct {
	buf     []byte // composition courante (ce qu'on veut)
	shown   []byte // dernier état émis (ce que le terminal affiche)
	started bool   // HiOn déjà envoyé ? (1ʳᵉ image = bascule HIRES)
}

// NewHiresScreen crée un buffer HIRES vide (écran effacé). Le premier Render()
// émet HiOn puis le contenu non vide.
func NewHiresScreen() *HiresScreen {
	h := &HiresScreen{buf: make([]byte, hiresVRAMSize), shown: make([]byte, hiresVRAMSize)}
	h.Clear()
	return h
}

// Clear remet la composition à l'écran vide (n'émet rien tant que Render n'est pas
// appelé). Appelé au début de chaque image d'une animation.
func (h *HiresScreen) Clear() {
	for i := range h.buf {
		h.buf[i] = hiresClearByte
	}
}

// Reset oublie l'état affiché ET la bascule : le prochain Render() réémettra HiOn
// et tout le contenu. Utile après une (re)connexion.
func (h *HiresScreen) Reset() { h.started = false }

// inBounds indique si (x,y) est dans la zone 240×200.
func inBounds(x, y int) bool { return x >= 0 && x < hiresPixW && y >= 0 && y < hiresPixH }

// SetPixel allume le pixel (x,y). Hors champ : ignoré (lecture défensive).
func (h *HiresScreen) SetPixel(x, y int) {
	if !inBounds(x, y) {
		return
	}
	h.buf[y*hiresBytesPerRow+x/6] |= 1 << uint(5-x%6)
}

// ClearPixel éteint le pixel (x,y) sans toucher aux autres de la cellule.
func (h *HiresScreen) ClearPixel(x, y int) {
	if !inBounds(x, y) {
		return
	}
	h.buf[y*hiresBytesPerRow+x/6] &^= 1 << uint(5-x%6)
}

// Line trace un segment de (x0,y0) à (x1,y1) (Bresenham).
func (h *HiresScreen) Line(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		h.SetPixel(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Box trace le contour du rectangle défini par deux coins.
func (h *HiresScreen) Box(x0, y0, x1, y1 int) {
	h.Line(x0, y0, x1, y0)
	h.Line(x1, y0, x1, y1)
	h.Line(x1, y1, x0, y1)
	h.Line(x0, y1, x0, y0)
}

// FillBox remplit le rectangle défini par deux coins (lignes horizontales).
func (h *HiresScreen) FillBox(x0, y0, x1, y1 int) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		h.Line(x0, y, x1, y)
	}
}

// Circle trace un cercle de centre (cx,cy) et de rayon r (midpoint).
func (h *HiresScreen) Circle(cx, cy, r int) {
	if r < 0 {
		return
	}
	x, y := r, 0
	err := 1 - r
	for x >= y {
		h.SetPixel(cx+x, cy+y)
		h.SetPixel(cx+y, cy+x)
		h.SetPixel(cx-y, cy+x)
		h.SetPixel(cx-x, cy+y)
		h.SetPixel(cx-x, cy-y)
		h.SetPixel(cx-y, cy-x)
		h.SetPixel(cx+y, cy-x)
		h.SetPixel(cx+x, cy-y)
		y++
		if err < 0 {
			err += 2*y + 1
		} else {
			x--
			err += 2*(y-x) + 1
		}
	}
}

// SetBitmap charge un buffer VRAM complet (8000 octets) dans la composition.
// Une taille incorrecte est ignorée (lecture défensive).
func (h *HiresScreen) SetBitmap(vram []byte) {
	if len(vram) != hiresVRAMSize {
		return
	}
	copy(h.buf, vram)
}

// Render produit le flux minimal pour mettre l'écran HIRES du terminal à jour, puis
// mémorise le nouvel état affiché. La 1ʳᵉ image émet HiOn (bascule + efface) ; les
// suivantes n'émettent que les runs d'octets modifiés (HiBlit + RLE). Renvoie nil
// si rien n'a changé depuis la dernière image (hors 1ʳᵉ image).
func (h *HiresScreen) Render() []byte {
	first := !h.started
	if first {
		// Après HiOn, le terminal a effacé la VRAM à 0x40 → base de diff.
		for i := range h.shown {
			h.shown[i] = hiresClearByte
		}
	}

	var blits []byte
	i := 0
	for i < len(h.buf) {
		if h.buf[i] == h.shown[i] {
			i++
			continue
		}
		start := i
		for i < len(h.buf) && h.buf[i] != h.shown[i] {
			i++
		}
		run := h.buf[start:i]
		blits = append(blits, HiBlit,
			byte(start&0xFF), byte(start>>8), // offset dans la VRAM
			byte(len(run)&0xFF), byte(len(run)>>8)) // longueur décodée
		blits = append(blits, RLEEncode(run)...)
	}

	if !first && len(blits) == 0 {
		return nil // rien à envoyer
	}
	copy(h.shown, h.buf)
	h.started = true

	out := []byte{PlotByte, hiresByte} // 1F FC
	if first {
		out = append(out, HiOn)
	}
	out = append(out, blits...)
	out = append(out, HiEnd)
	return out
}

// Buffer renvoie la VRAM composée (8000 octets). À ne pas modifier directement
// (utiliser SetPixel/Line/… ou SetBitmap).
func (h *HiresScreen) Buffer() []byte { return h.buf }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
