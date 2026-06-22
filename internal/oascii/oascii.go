// Package oascii (clin d'œil à PETSCII / ATASCII) encode la couche d'affichage
// de l'Oric.
//
// IMPORTANT — contrairement à PETSCII (Commodore) ou ATASCII (Atari) qui sont
// des jeux de caractères PROPRIÉTAIRES, l'Oric utilise l'ASCII STANDARD pour
// les caractères imprimables. « OASCII » désigne ici non pas un encodage de
// caractères, mais le modèle d'affichage particulier de l'Oric : son mode TEXT
// 40×28 de type Téletexte, où les couleurs et attributs sont posés par des
// « attributs sériels » — des octets de contrôle (0–31) qui OCCUPENT une case
// écran et s'appliquent jusqu'à la fin de la ligne.
//
// Table d'attributs (source de vérité : décodeur ULA de l'émulateur de
// référence Oric1/oric1-emu, src/video/video.c, fonction decode_attr) :
//
//	val & 0x18 == 0x00  → 0–7   : INK   (encre)  = val & 7
//	val & 0x18 == 0x08  → 8–15  : attrs texte    : bit0=charset alt, bit1=double hauteur, bit2=clignotement
//	val & 0x18 == 0x10  → 16–23 : PAPER (fond)   = val & 7
//	val & 0x18 == 0x18  → 24–31 : mode vidéo     (+ 28=inverse off, 29=inverse on)
//
// L'ULA réinitialise les attributs au début de CHAQUE ligne (encre=blanc,
// fond=noir). Le Builder gère donc, en option, la ré-émission automatique des
// attributs courants après un saut de ligne (mode « sticky »).
package oascii

import "bytes"

// Color est une couleur Oric (0–7).
type Color uint8

// Palette Oric (bits R, G, B). Voir palette[] dans src/video/video.c.
const (
	Black   Color = 0 // 000
	Red     Color = 1 // R00
	Green   Color = 2 // 0G0
	Yellow  Color = 3 // RG0
	Blue    Color = 4 // 00B
	Magenta Color = 5 // R0B
	Cyan    Color = 6 // 0GB
	White   Color = 7 // RGB
)

// Largeur et hauteur de l'écran TEXT de l'Oric.
const (
	Cols = 40
	Rows = 28
)

// Octets d'attribut sériel. Chacun occupe une case écran.

// InkAttr renvoie l'octet d'attribut qui fixe la couleur d'encre.
func InkAttr(c Color) byte { return byte(c) & 0x07 } // 0–7

// PaperAttr renvoie l'octet d'attribut qui fixe la couleur de fond.
func PaperAttr(c Color) byte { return 0x10 | (byte(c) & 0x07) } // 16–23

// InverseAttr renvoie l'octet d'attribut vidéo inverse (groupe 24–31) :
// 29 active la vidéo inverse, 28 la désactive.
func InverseAttr(on bool) byte {
	if on {
		return 29
	}
	return 28
}

// TextAttr renvoie l'octet d'attribut texte (clignotement / double hauteur /
// charset alternatif).
func TextAttr(blink, doubleHeight, altCharset bool) byte {
	v := byte(0x08) // groupe 8–15
	if altCharset {
		v |= 0x01
	}
	if doubleHeight {
		v |= 0x02
	}
	if blink {
		v |= 0x04
	}
	return v
}

// Builder construit un flux d'octets destiné au terminal Oric : caractères
// ASCII imprimables + attributs sériels intercalés. Le terminal Oric écrit ces
// octets en mémoire écran ; les octets 0–31 y deviennent des attributs.
type Builder struct {
	buf    bytes.Buffer
	ink    Color
	paper  Color
	blink  bool
	dbl    bool
	alt    bool
	sticky bool // ré-émettre les attributs après chaque saut de ligne
}

// New crée un Builder à l'état par défaut de l'Oric (encre blanche, fond noir).
func New() *Builder {
	return &Builder{ink: White, paper: Black}
}

// Sticky active/désactive la ré-émission automatique des attributs courants en
// début de ligne (utile pour conserver une couleur sur plusieurs lignes,
// l'ULA les réinitialisant à chaque ligne). Renvoie b pour chaînage.
func (b *Builder) Sticky(on bool) *Builder { b.sticky = on; return b }

// Ink fixe la couleur d'encre (émet l'octet d'attribut correspondant).
func (b *Builder) Ink(c Color) *Builder {
	b.ink = c
	b.buf.WriteByte(InkAttr(c))
	return b
}

// Paper fixe la couleur de fond.
func (b *Builder) Paper(c Color) *Builder {
	b.paper = c
	b.buf.WriteByte(PaperAttr(c))
	return b
}

// Blink active/désactive le clignotement (recompose l'octet d'attribut texte).
func (b *Builder) Blink(on bool) *Builder {
	b.blink = on
	b.buf.WriteByte(TextAttr(b.blink, b.dbl, b.alt))
	return b
}

// DoubleHeight active/désactive la double hauteur.
func (b *Builder) DoubleHeight(on bool) *Builder {
	b.dbl = on
	b.buf.WriteByte(TextAttr(b.blink, b.dbl, b.alt))
	return b
}

// AltCharset bascule le jeu de caractères alternatif.
func (b *Builder) AltCharset(on bool) *Builder {
	b.alt = on
	b.buf.WriteByte(TextAttr(b.blink, b.dbl, b.alt))
	return b
}

// Attrs fixe en une seule fois clignotement / double hauteur / charset alternatif
// (un seul octet d'attribut, contrairement à des appels Blink/DoubleHeight/AltCharset
// successifs qui en émettraient plusieurs).
func (b *Builder) Attrs(blink, doubleHeight, altCharset bool) *Builder {
	b.blink, b.dbl, b.alt = blink, doubleHeight, altCharset
	b.buf.WriteByte(TextAttr(blink, doubleHeight, altCharset))
	return b
}

// Inverse active/désactive la vidéo inverse.
func (b *Builder) Inverse(on bool) *Builder {
	b.buf.WriteByte(InverseAttr(on))
	return b
}

// Text écrit du texte ASCII. Les octets non imprimables (< 32 ou > 126) sont
// remplacés par un espace pour ne pas être pris à tort pour des attributs.
func (b *Builder) Text(s string) *Builder {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c > 0x7E {
			c = ' '
		}
		b.buf.WriteByte(c)
	}
	return b
}

// Newline termine la ligne (CR LF). En mode sticky, ré-émet les attributs
// courants non par défaut pour la ligne suivante.
func (b *Builder) Newline() *Builder {
	b.buf.WriteString("\r\n")
	if b.sticky {
		if b.ink != White {
			b.buf.WriteByte(InkAttr(b.ink))
		}
		if b.paper != Black {
			b.buf.WriteByte(PaperAttr(b.paper))
		}
		if b.blink || b.dbl || b.alt {
			b.buf.WriteByte(TextAttr(b.blink, b.dbl, b.alt))
		}
	}
	return b
}

// Bytes renvoie le flux construit.
func (b *Builder) Bytes() []byte { return b.buf.Bytes() }

// String renvoie le flux construit (les octets 0–31 sont préservés).
func (b *Builder) String() string { return b.buf.String() }
