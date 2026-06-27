package render

import (
	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Hires sérialise une page graphique en un flux de commandes HIRES destiné au
// terminal (sous-commande série 1F FC, cf. oascii). L'ordre est : ouverture du
// flux → bascule HIRES + effacement → fond bitmap éventuel (Blit RLE) → primitives
// de tracé → fin du flux. C'est la SOURCE UNIQUE du rendu HIRES (serveur + studio).
//
// Un seul flux porte les DEUX modèles : `Background` (bitmap posé d'un bloc) et
// `Draw` (primitives appliquées par-dessus) ; l'auteur peut combiner les deux.
func Hires(p *content.Page) []byte {
	h := p.Hires
	if h == nil {
		return nil
	}
	out := []byte{oascii.PlotByte, 0xFC, oascii.HiOn} // 1F FC + HiOn (= HiresCmd + bascule)

	// Fond bitmap (modèle « bitmap ») : tout le buffer VRAM, compressé RLE.
	if len(h.Background) == content.HiresBitmapSize {
		rle := oascii.RLEEncode(h.Background)
		n := content.HiresBitmapSize
		out = append(out, oascii.HiBlit,
			0, 0, // offset $A000+0
			byte(n&0xFF), byte(n>>8)) // longueur décodée
		out = append(out, rle...)
	}

	// Primitives (modèle « vectoriel »), dans l'ordre déclaré.
	for _, op := range h.Draw {
		switch op.Op {
		case "ink":
			out = append(out, oascii.HiInk, byte(op.C&7))
		case "paper":
			out = append(out, oascii.HiPaper, byte(op.C&7))
		case "curset":
			out = append(out, oascii.HiCurset, byte(op.X), byte(op.Y))
		case "point":
			out = append(out, oascii.HiPoint, byte(op.X), byte(op.Y))
		case "line":
			out = append(out, oascii.HiLine, byte(op.X), byte(op.Y))
		case "box":
			out = append(out, oascii.HiBox, byte(op.X), byte(op.Y))
		case "fillbox":
			out = append(out, oascii.HiFillBox, byte(op.X), byte(op.Y))
		case "circle":
			out = append(out, oascii.HiCircle, byte(op.R))
		case "char":
			c := byte(' ')
			if op.Ch != "" {
				c = op.Ch[0]
			}
			out = append(out, oascii.HiChar, byte(op.X), byte(op.Y), c)
		}
	}

	out = append(out, oascii.HiEnd)
	return out
}
