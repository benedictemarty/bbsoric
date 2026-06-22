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
func emitStyle(b *oascii.Builder, cur *styleState, st content.Style) {
	ink := content.Ink(st.Ink) // blanc si vide
	paper := oascii.Black
	if st.Paper != "" {
		paper = content.Ink(st.Paper)
	}
	if paper != cur.paper {
		b.Paper(paper)
		cur.paper = paper
	}
	if st.Blink != cur.blink || st.DoubleHeight != cur.dbl || st.AltCharset != cur.alt {
		b.Attrs(st.Blink, st.DoubleHeight, st.AltCharset)
		cur.blink, cur.dbl, cur.alt = st.Blink, st.DoubleHeight, st.AltCharset
	}
	if ink != cur.ink {
		b.Ink(ink)
		cur.ink = ink
	}
}

// writeLine rend une ligne : texte simple stylé, ou suite de segments stylés
// (multicolore/multi-attribut). L'inverse pose le bit 7 par caractère.
func writeLine(b *oascii.Builder, ln content.Line) {
	spans := ln.Segments
	if len(spans) == 0 {
		spans = []content.Span{{Text: ln.Text, Style: ln.Style}}
	}
	cur := defaultStyleState()
	for _, sp := range spans {
		emitStyle(b, &cur, sp.Style)
		if sp.Inverse {
			b.InverseText(sp.Text)
		} else {
			b.Text(sp.Text)
		}
	}
	b.Newline()
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
