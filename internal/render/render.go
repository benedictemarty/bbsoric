// Package render produit le flux d'octets OASCII d'un écran de page. C'est la
// SOURCE UNIQUE du rendu : le serveur l'écrit vers la session, et le studio
// l'utilise pour l'aperçu (HTML et simulateur ULA) — aucune divergence.
package render

import (
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// rule trace une règle pleine largeur (40 colonnes).
func rule() string { return strings.Repeat("=", oascii.Cols) }

// center centre un texte sur 40 colonnes.
func center(text string) string {
	if len(text) >= oascii.Cols {
		return text
	}
	return strings.Repeat(" ", (oascii.Cols-len(text))/2) + text
}

// styleState suit les attributs courants le long d'une ligne afin de n'émettre
// que les changements (chaque octet d'attribut occupe une case écran).
type styleState struct {
	ink, paper      oascii.Color
	blink, dbl, alt bool
}

func defaultStyleState() styleState { return styleState{ink: oascii.White, paper: oascii.Black} }

// emitStyle émet les attributs nécessaires pour passer de cur au style st
// (valeurs non renseignées = défaut) puis met cur à jour. L'inverse n'est PAS
// un attribut ici : il s'applique par caractère (bit 7) au moment d'écrire.
// Renvoie le nombre d'octets d'attribut émis (chacun occupe une case écran).
func emitStyle(b *oascii.Builder, cur *styleState, st content.Style) int {
	ink := content.Ink(st.Ink) // blanc si vide
	paper := oascii.Black
	if st.Paper != "" {
		paper = content.Ink(st.Paper)
	}
	n := 0
	if paper != cur.paper {
		b.Paper(paper)
		cur.paper = paper
		n++
	}
	if st.Blink != cur.blink || st.DoubleHeight != cur.dbl || st.AltCharset != cur.alt {
		b.Attrs(st.Blink, st.DoubleHeight, st.AltCharset)
		cur.blink, cur.dbl, cur.alt = st.Blink, st.DoubleHeight, st.AltCharset
		n++
	}
	if ink != cur.ink {
		b.Ink(ink)
		cur.ink = ink
		n++
	}
	return n
}

// reemitState ré-émet les attributs non-défaut de st (l'ULA réinitialise au début
// de chaque ligne). Renvoie le nombre de cases consommées. Sert au repli (wrap).
func reemitState(b *oascii.Builder, st styleState) int {
	n := 0
	if st.paper != oascii.Black {
		b.Paper(st.paper)
		n++
	}
	if st.blink || st.dbl || st.alt {
		b.Attrs(st.blink, st.dbl, st.alt)
		n++
	}
	if st.ink != oascii.White {
		b.Ink(st.ink)
		n++
	}
	return n
}

// emitChars écrit du texte (inverse = bit 7 par caractère).
func emitChars(b *oascii.Builder, s string, inv bool) {
	if inv {
		b.InverseText(s)
	} else {
		b.Text(s)
	}
}

// emitLineSpans écrit les fragments stylés d'une ligne (sans saut de ligne ni
// repli — utilisé tel quel pour l'« écran brut »).
func emitLineSpans(b *oascii.Builder, ln content.Line) {
	spans := ln.Segments
	if len(spans) == 0 {
		spans = []content.Span{{Text: ln.Text, Style: ln.Style}}
	}
	cur := defaultStyleState()
	for _, sp := range spans {
		emitStyle(b, &cur, sp.Style)
		emitChars(b, sp.Text, sp.Inverse)
	}
}

// emitLineWrapped écrit les fragments d'une ligne en repliant à la largeur écran
// (40 colonnes), aux espaces (césure dure pour un mot trop long). Au passage à
// la ligne, les attributs courants (encre/fond/…) sont RÉ-ÉMIS pour conserver le
// même rendu sur la ligne n+1 (l'ULA réinitialise sinon).
func emitLineWrapped(b *oascii.Builder, ln content.Line) {
	spans := ln.Segments
	if len(spans) == 0 {
		spans = []content.Span{{Text: ln.Text, Style: ln.Style}}
	}
	cur := defaultStyleState()
	col := 0       // cases utilisées sur la ligne physique courante
	lineStart := 0 // cases occupées par les attributs ré-émis en début de ligne
	wrap := func() {
		b.Newline()
		col = reemitState(b, cur)
		lineStart = col
	}
	for _, sp := range spans {
		col += emitStyle(b, &cur, sp.Style)
		words := strings.Split(sp.Text, " ")
		for wi, w := range words {
			if wi > 0 { // séparateur espace entre mots
				if col >= oascii.Cols {
					wrap()
				} else {
					emitChars(b, " ", sp.Inverse)
					col++
				}
			}
			// le mot entier ne tient pas sur la fin de ligne → replier d'abord
			if col > lineStart && col+len(w) > oascii.Cols {
				wrap()
			}
			// mot plus long qu'une ligne entière → césure dure
			for col+len(w) > oascii.Cols && oascii.Cols-col > 0 {
				take := oascii.Cols - col
				emitChars(b, w[:take], sp.Inverse)
				w = w[take:]
				wrap()
			}
			emitChars(b, w, sp.Inverse)
			col += len(w)
		}
	}
}

// writeLine rend une ligne complète (fragments repliés + saut de ligne).
func writeLine(b *oascii.Builder, ln content.Line) {
	emitLineWrapped(b, ln)
	b.Newline()
}

// RawScreen rend une page « écran brut » sans barre de titre ni invite. Si la
// page porte un buffer Screen (40×28 octets posés par l'éditeur, attributs
// inclus comme cases), il est envoyé tel quel (ligne par ligne). Sinon, repli
// sur les Lines. Pas de saut de ligne après la dernière ligne (évite le scroll).
func RawScreen(p *content.Page) []byte {
	if len(p.Screen) > 0 {
		return screenRows(p.Screen)
	}
	b := oascii.New()
	for i, ln := range p.Lines {
		emitLineSpans(b, ln)
		if i < len(p.Lines)-1 {
			b.Newline()
		}
	}
	return b.Bytes()
}

// screenRows découpe un buffer écran (cases) en lignes de 40 et les émet,
// séparées par CR LF, en élaguant les lignes entièrement vides du bas.
func screenRows(buf []byte) []byte {
	const cols, rows = oascii.Cols, oascii.Rows
	last := -1
	get := func(r int) []byte {
		start := r * cols
		if start >= len(buf) {
			return nil
		}
		end := start + cols
		if end > len(buf) {
			end = len(buf)
		}
		return buf[start:end]
	}
	for r := 0; r < rows; r++ {
		row := get(r)
		for _, c := range row {
			if c != 0x20 {
				last = r
				break
			}
		}
	}
	var out []byte
	for r := 0; r <= last; r++ {
		out = append(out, get(r)...)
		if r < last {
			out = append(out, '\r', '\n')
		}
	}
	return out
}

// Screen renvoie le flux OASCII de l'écran complet d'une page : barre de titre,
// texte/segments éventuels, puis les choix + invite « Votre choix> » (si la page
// a des entrées) ou l'invite « Appuyez sur une touche... » (écran de contenu).
func Screen(p *content.Page) []byte {
	b := oascii.New()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(p.Title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	for _, ln := range p.Lines {
		writeLine(b, ln)
	}
	if len(p.Lines) > 0 && len(p.Entries) > 0 {
		b.Newline()
	}
	for _, e := range p.Entries {
		b.Ink(oascii.Cyan).Text(" " + e.Key)
		b.Ink(oascii.White).Text(" - " + e.Label).Newline()
	}
	b.Newline()
	if len(p.Entries) > 0 {
		b.Ink(oascii.Green).Text("Votre choix")
		b.Ink(oascii.White).Text("> ")
	} else {
		b.Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	}
	return b.Bytes()
}
