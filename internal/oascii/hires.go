package oascii

// --- Flux de commandes HIRES (mode graphique 240×200) ---
//
// Le terminal route ses commandes sur l'octet PlotByte (0x1F) : 1F col row = plot,
// 1F FE/FD = transferts XMODEM. On réserve ici 1F FC = « flux de commandes HIRES ».
// 0xFC est hors de la plage des colonnes valides (0..39) donc sans ambiguïté avec
// un plot, et libre (FE/FD pris). Après 1F FC, le terminal lit une suite d'OPCODES
// (1 octet + arguments de longueur fixe, sauf Blit qui porte sa taille) jusqu'à
// HiEnd. Un terminal texte générique (telnet/PC) ignore la séquence.
const hiresByte = 0xFC

// HiresCmd ouvre un flux de commandes HIRES (1F FC).
func HiresCmd() string { return string([]byte{PlotByte, hiresByte}) }

// Opcodes du flux HIRES. Les coordonnées tiennent sur 1 octet (x:0-239, y:0-199).
// Le terminal maintient un « crayon » (pen) déplacé par Curset/Point/Line.
const (
	HiEnd     = 0x00 // fin du flux → retour au traitement texte normal
	HiOn      = 0x01 // bascule en HIRES + efface (encre blanc, fond noir)
	HiInk     = 0x02 // + c : couleur d'encre (0-7)
	HiPaper   = 0x03 // + c : couleur de fond (0-7)
	HiCurset  = 0x10 // + x, y : déplace le crayon (sans tracer)
	HiPoint   = 0x11 // + x, y : allume le pixel (crayon ← x,y)
	HiLine    = 0x12 // + x, y : trace du crayon vers (x,y) (crayon ← x,y)
	HiBox     = 0x13 // + x, y : rectangle vide du crayon à (x,y)
	HiFillBox = 0x14 // + x, y : rectangle plein du crayon à (x,y)
	HiCircle  = 0x15 // + r    : cercle de rayon r autour du crayon
	HiChar    = 0x16 // + x, y, ch : caractère ASCII en (x,y)
	HiBlit    = 0x20 // + off(lo,hi) + len(lo,hi) + flux RLE : écrit len octets en $A000+off
)

// RLEEncode compresse un buffer en paires (compteur 1-255, valeur) — un schéma
// simple et robuste sur la liaison série, suffisant pour un fond bitmap (grandes
// plages d'octets identiques). RLEDecode est l'inverse (taille décodée connue).
func RLEEncode(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		v := data[i]
		n := 1
		for i+n < len(data) && data[i+n] == v && n < 255 {
			n++
		}
		out = append(out, byte(n), v)
		i += n
	}
	return out
}

// RLEDecode reconstitue n octets depuis un flux RLE (paires compteur/valeur).
func RLEDecode(rle []byte, n int) []byte {
	out := make([]byte, 0, n)
	for i := 0; i+1 < len(rle) && len(out) < n; i += 2 {
		cnt, val := int(rle[i]), rle[i+1]
		for j := 0; j < cnt && len(out) < n; j++ {
			out = append(out, val)
		}
	}
	return out
}
