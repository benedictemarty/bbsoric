// Package hires rasterise le flux de commandes HIRES du BBS Oric (mode graphique
// 240×200) dans une VRAM 200×40 octets, puis le restitue en couleurs pour un
// terminal moderne (demi-blocs Unicode ▀ : un caractère = 2 pixels verticaux).
//
// Sémantique portée fidèlement du firmware terminal (client/hires.s) et de
// l'aperçu du studio :
//   - VRAM 200 lignes × 40 octets ; octet à l'offset y*40 + x/6 ; le pixel x est
//     le bit (5 - x%6) (bit 5 = pixel le plus à gauche) ;
//   - un octet « pixel » a le bit 6 (0x40) posé ; l'effacement remplit la VRAM
//     de 0x40 (bit 6 seul, tous pixels éteints) ;
//   - la couleur est un attribut SÉRIEL par ligne : en mode couleur (après un op
//     ink), l'octet d'encre (0–7, bit 6 à 0) est posé en colonne 0 de chaque ligne
//     dessinée, la 1re cellule (x 0–5) est alors sacrifiée à l'attribut ;
//   - sans op ink, rendu monochrome encre blanche.
//
// Opcodes : voir internal/oascii/hires.go. Le décodage lit un opcode puis ses
// arguments de longueur fixe (Blit porte sa taille), jusqu'à HiEnd (0x00).
package hires

import (
	"strconv"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Dimensions de l'écran HIRES.
const (
	W       = 240
	H       = 200
	rowByte = W / 6 // 40 octets par ligne VRAM
	empty   = 0x40  // octet pixel vide (bit 6 posé, pixels éteints)
)

// Raster maintient la VRAM et l'état de décodage du flux HIRES.
type Raster struct {
	vram      [H * rowByte]byte
	penX      int
	penY      int
	ink       byte
	colorMode bool // hcolor : pose l'attribut d'encre en col 0 des lignes dessinées

	// Décodage du flux : opcode courant + arguments accumulés.
	op    byte
	args  []byte
	need  int  // nombre d'octets d'arguments attendus (hors Blit)
	inCmd bool // un opcode est en cours de lecture d'arguments
	blit  *blitState
	done  bool
}

// New crée un rasteriseur prêt (VRAM effacée, encre blanche, monochrome).
func New() *Raster {
	r := &Raster{}
	r.reset()
	return r
}

func (r *Raster) reset() {
	for i := range r.vram {
		r.vram[i] = empty
	}
	r.penX, r.penY = 0, 0
	r.ink = 7
	r.colorMode = false
	r.op, r.args, r.need, r.inCmd, r.blit, r.done = 0, r.args[:0], 0, false, nil, false
}

// Done indique que le flux s'est terminé (HiEnd reçu).
func (r *Raster) Done() bool { return r.done }

// argCount renvoie le nombre d'octets d'arguments d'un opcode à taille fixe.
// Blit (-1) porte sa propre longueur et est traité à part.
func argCount(op byte) int {
	switch op {
	case oascii.HiEnd, oascii.HiOn:
		return 0
	case oascii.HiInk, oascii.HiPaper, oascii.HiCircle:
		return 1
	case oascii.HiCurset, oascii.HiPoint, oascii.HiLine, oascii.HiBox, oascii.HiFillBox:
		return 2
	case oascii.HiChar:
		return 3
	case oascii.HiBlit:
		return -1
	default:
		return 0
	}
}

// Feed injecte un octet du flux HIRES. Renvoie true quand le flux est terminé
// (HiEnd) : l'appelant peut alors afficher l'image et repasser en mode texte à la
// prochaine commande 1F FB.
func (r *Raster) Feed(b byte) bool {
	if r.done {
		return true
	}
	if r.blit != nil {
		if r.blit.feed(r, b) {
			r.blit = nil
			r.inCmd = false
		}
		return false
	}
	if !r.inCmd {
		r.op = b
		r.need = argCount(b)
		r.args = r.args[:0]
		if r.op == oascii.HiBlit {
			r.blit = &blitState{}
			r.inCmd = true
			return false
		}
		if r.need == 0 {
			r.exec()
			return r.done
		}
		r.inCmd = true
		return false
	}
	r.args = append(r.args, b)
	if len(r.args) >= r.need {
		r.exec()
		r.inCmd = false
	}
	return r.done
}

// exec applique l'opcode courant avec ses arguments accumulés.
func (r *Raster) exec() {
	switch r.op {
	case oascii.HiEnd:
		r.done = true
	case oascii.HiOn:
		for i := range r.vram {
			r.vram[i] = empty
		}
		r.penX, r.penY, r.ink, r.colorMode = 0, 0, 7, false
	case oascii.HiInk:
		r.ink = r.args[0] & 7
		r.colorMode = true
	case oascii.HiPaper:
		// Non rendu par le firmware (fond noir) — accepté et ignoré (parité).
	case oascii.HiCurset:
		r.penX, r.penY = int(r.args[0]), int(r.args[1])
	case oascii.HiPoint:
		r.penX, r.penY = int(r.args[0]), int(r.args[1])
		r.setPixel(r.penX, r.penY)
	case oascii.HiLine:
		x, y := int(r.args[0]), int(r.args[1])
		r.line(r.penX, r.penY, x, y)
		r.penX, r.penY = x, y
	case oascii.HiBox:
		x, y := int(r.args[0]), int(r.args[1])
		r.line(r.penX, r.penY, x, r.penY)
		r.line(x, r.penY, x, y)
		r.line(x, y, r.penX, y)
		r.line(r.penX, y, r.penX, r.penY)
		r.penX, r.penY = x, y
	case oascii.HiFillBox:
		x, y := int(r.args[0]), int(r.args[1])
		r.fillBox(r.penX, r.penY, x, y)
		r.penX, r.penY = x, y
	case oascii.HiCircle:
		r.circle(r.penX, r.penY, int(r.args[0]))
	case oascii.HiChar:
		r.char(int(r.args[0]), int(r.args[1]), r.args[2])
	}
}

// setPixel allume le pixel (x,y). En mode couleur, pose l'attribut d'encre en
// colonne 0 de la ligne et ignore x < 6 (cellule sacrifiée à l'attribut), fidèle
// à client/hires.s.
func (r *Raster) setPixel(x, y int) {
	if x < 0 || x >= W || y < 0 || y >= H {
		return
	}
	if r.colorMode {
		r.vram[y*rowByte] = r.ink & 7 // attribut d'encre (bit 6 = 0)
		if x < 6 {
			return
		}
	}
	r.vram[y*rowByte+x/6] |= 1 << (5 - byte(x%6))
}

// line trace un segment (Bresenham) de (x0,y0) à (x1,y1) inclus.
func (r *Raster) line(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	if dx >= dy {
		err := dx / 2
		for {
			r.setPixel(x0, y0)
			if x0 == x1 {
				break
			}
			x0 += sx
			err -= dy
			if err < 0 {
				y0 += sy
				err += dx
			}
		}
	} else {
		err := dy / 2
		for {
			r.setPixel(x0, y0)
			if y0 == y1 {
				break
			}
			y0 += sy
			err -= dx
			if err < 0 {
				x0 += sx
				err += dy
			}
		}
	}
}

// fillBox remplit le rectangle (x0,y0)-(x1,y1) de lignes horizontales.
func (r *Raster) fillBox(x0, y0, x1, y1 int) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	xa, xb := x0, x1
	if xa > xb {
		xa, xb = xb, xa
	}
	for y := y0; y <= y1; y++ {
		for x := xa; x <= xb; x++ {
			r.setPixel(x, y)
		}
	}
}

// circle trace un cercle (midpoint) de rayon r0 autour de (cx,cy).
func (r *Raster) circle(cx, cy, r0 int) {
	if r0 < 0 {
		return
	}
	x, y := r0, 0
	err := 1 - r0
	for x >= y {
		r.setPixel(cx+x, cy+y)
		r.setPixel(cx-x, cy+y)
		r.setPixel(cx+x, cy-y)
		r.setPixel(cx-x, cy-y)
		r.setPixel(cx+y, cy+x)
		r.setPixel(cx-y, cy+x)
		r.setPixel(cx+y, cy-x)
		r.setPixel(cx-y, cy-x)
		y++
		if err <= 0 {
			err += 2*y + 1
		} else {
			x--
			err += 2*(y-x) + 1
		}
	}
}

// char trace le glyphe ch (police standard 6×8) en (x,y).
func (r *Raster) char(x, y int, ch byte) {
	idx := int(ch & 0x7F)
	if idx < 0x20 || idx > 0x7F {
		return
	}
	base := (idx - 0x20) * 8
	for row := 0; row < 8; row++ {
		bits := stdFont[base+row]
		for col := 0; col < 6; col++ {
			if bits&(1<<(5-byte(col))) != 0 {
				r.setPixel(x+col, y+row)
			}
		}
	}
}

// resolve interprète la VRAM en image couleur 240×200 : chaque ligne applique en
// SÉRIE l'attribut d'encre (octet bit 6 à 0) et les octets pixels (bit 6 posé),
// pixel allumé -> encre, éteint -> fond noir (0). Renvoie color[y*W+x] ∈ 0..7.
func (r *Raster) resolve() []byte {
	img := make([]byte, W*H)
	for y := 0; y < H; y++ {
		ink := byte(7)
		for c := 0; c < rowByte; c++ {
			b := r.vram[y*rowByte+c]
			if b&0x40 == 0 { // attribut sériel (ici : encre 0–7)
				if b&0x18 == 0x00 {
					ink = b & 7
				}
				// La cellule d'attribut s'affiche en fond (déjà 0 par défaut).
				continue
			}
			for p := 0; p < 6; p++ {
				if b&(1<<(5-byte(p))) != 0 {
					img[y*W+c*6+p] = ink
				}
			}
		}
	}
	return img
}

// Image renvoie l'image HIRES résolue : W*H couleurs (0..7), pixel (x,y) à
// l'indice y*W+x. C'est la sortie « pixels » du rasteriseur, que l'appelant peut
// composer avec d'autres zones (les 3 lignes texte du bas de l'écran HIRES).
func (r *Raster) Image() []byte { return r.resolve() }

// Rows est le nombre de lignes de terminal d'une image HIRES rendue en demi-blocs
// (2 pixels verticaux par ligne).
const Rows = H / 2

// HalfBlockFrame restitue une image 240×200 (W*H couleurs) en demi-blocs Unicode
// ▀ : 100 lignes de 240 caractères, encre du pixel haut en avant-plan, du pixel
// bas en arrière-plan. Débute par un retour en haut à gauche (sans gérer le
// curseur : à l'appelant de le masquer/positionner). Les 8 couleurs Oric
// coïncident avec l'ordre ANSI (0=noir … 7=blanc).
func HalfBlockFrame(img []byte) string {
	var sb strings.Builder
	sb.WriteString("\x1b[H")
	lastTop, lastBot := byte(255), byte(255)
	for y := 0; y < H; y += 2 {
		for x := 0; x < W; x++ {
			top := img[y*W+x]
			bot := img[(y+1)*W+x]
			if top != lastTop || bot != lastBot {
				sb.WriteString("\x1b[3")
				sb.WriteByte('0' + top) // avant-plan = pixel haut
				sb.WriteString(";4")
				sb.WriteByte('0' + bot) // arrière-plan = pixel bas
				sb.WriteByte('m')
				lastTop, lastBot = top, bot
			}
			sb.WriteString("▀") // ▀ demi-bloc haut
		}
		sb.WriteString("\x1b[0m")
		lastTop, lastBot = 255, 255
		if y+2 < H {
			sb.WriteString("\r\n")
		}
	}
	return sb.String()
}

// RenderANSI restitue l'image HIRES seule (240×200) en demi-blocs, curseur géré
// (masqué puis replacé sous l'image). Pour un écran HIRES complet avec les 3
// lignes texte du bas, l'appelant compose HalfBlockFrame(Image()) + le texte.
func (r *Raster) RenderANSI() string {
	return "\x1b[?25l" + HalfBlockFrame(r.resolve()) +
		"\x1b[" + strconv.Itoa(Rows+1) + ";1H\x1b[?25h"
}

// ColorAt renvoie la couleur résolue (0..7) du pixel (x,y) — utile aux tests.
func (r *Raster) ColorAt(x, y int) byte {
	if x < 0 || x >= W || y < 0 || y >= H {
		return 0
	}
	return r.resolve()[y*W+x]
}

// Pixel indique si le pixel (x,y) est allumé dans la VRAM (indépendamment de la
// couleur) — utile aux tests des primitives.
func (r *Raster) Pixel(x, y int) bool {
	if x < 0 || x >= W || y < 0 || y >= H {
		return false
	}
	b := r.vram[y*rowByte+x/6]
	return b&0x40 != 0 && b&(1<<(5-byte(x%6))) != 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// blitState décode l'en-tête Blit (offset lo/hi, len lo/hi) puis le flux RLE
// (paires compteur/valeur) écrivant len octets en VRAM à partir de l'offset.
type blitState struct {
	hdr    []byte // 4 octets d'en-tête (off lo/hi, len lo/hi)
	dst    int    // offset courant d'écriture dans la VRAM
	remain int    // octets restant à écrire
	run    int    // compteur RLE en attente d'une valeur (0 = attend un compteur)
}

// feed consomme un octet du flux Blit. Renvoie true quand le blit est terminé.
func (bs *blitState) feed(r *Raster, b byte) bool {
	if len(bs.hdr) < 4 {
		bs.hdr = append(bs.hdr, b)
		if len(bs.hdr) == 4 {
			bs.dst = int(bs.hdr[0]) | int(bs.hdr[1])<<8
			bs.remain = int(bs.hdr[2]) | int(bs.hdr[3])<<8
			if bs.remain == 0 {
				return true
			}
		}
		return false
	}
	if bs.run == 0 { // octet = compteur RLE
		bs.run = int(b)
		if bs.run == 0 {
			bs.run = 1 // compteur nul : évite un blocage (parité défensive)
		}
		return false
	}
	for i := 0; i < bs.run && bs.remain > 0; i++ { // octet = valeur, répétée run fois
		if bs.dst >= 0 && bs.dst < len(r.vram) {
			r.vram[bs.dst] = b
		}
		bs.dst++
		bs.remain--
	}
	bs.run = 0
	return bs.remain == 0
}
