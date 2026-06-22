// Package preview rend une page du site en HTML coloré, fidèle au rendu du
// moteur (server/internal/bbs/engine.go) : barre de titre jaune, entrées de menu
// (touche cyan + libellé blanc), lignes colorées par leur encre, invites vertes.
// Réutilise la palette de internal/oascii et la table de couleurs de content.
package preview

import (
	"fmt"
	"html"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// cssByColor mappe une couleur Oric vers une couleur CSS (palette RGB pure).
var cssByColor = map[oascii.Color]string{
	oascii.Black:   "#000000",
	oascii.Red:     "#ff2222",
	oascii.Green:   "#22dd22",
	oascii.Yellow:  "#dddd22",
	oascii.Blue:    "#4444ff",
	oascii.Magenta: "#dd22dd",
	oascii.Cyan:    "#22dddd",
	oascii.White:   "#eeeeee",
}

func css(c oascii.Color) string {
	if v, ok := cssByColor[c]; ok {
		return v
	}
	return "#eeeeee"
}

// span produit un fragment HTML coloré (texte échappé).
func span(text string, c oascii.Color) string {
	return fmt.Sprintf(`<span style="color:%s">%s</span>`, css(c), html.EscapeString(text))
}

// spanHTML rend un fragment stylé : encre/fond (inverse = échange), double
// hauteur, clignotement et charset alternatif (classes CSS ; semi-graphiques
// seulement APPROXIMÉS — rendu fidèle = émulateur/Oric).
func spanHTML(text string, st content.Style) string {
	fg := content.Ink(st.Ink)
	bg := oascii.Black
	if st.Paper != "" {
		bg = content.Ink(st.Paper)
	}
	if st.Inverse {
		fg, bg = bg, fg
	}
	style := "color:" + css(fg) + ";background:" + css(bg)
	if st.DoubleHeight {
		style += ";font-size:1.7em;line-height:1"
	}
	var classes []string
	if st.Blink {
		classes = append(classes, "blink")
	}
	if st.AltCharset {
		classes = append(classes, "alt")
	}
	cls := ""
	if len(classes) > 0 {
		cls = ` class="` + strings.Join(classes, " ") + `"`
	}
	return fmt.Sprintf(`<span%s style="%s">%s</span>`, cls, style, html.EscapeString(text))
}

// lineSpan rend une ligne : texte simple stylé, ou suite de segments stylés.
func lineSpan(ln content.Line) string {
	if len(ln.Segments) == 0 {
		return spanHTML(ln.Text, ln.Style)
	}
	var b strings.Builder
	for _, sp := range ln.Segments {
		b.WriteString(spanHTML(sp.Text, sp.Style))
	}
	return b.String()
}

// rule trace une règle pleine largeur (40 colonnes), blanche.
func rule() string { return span(strings.Repeat("=", oascii.Cols), oascii.White) }

// center centre un texte sur 40 colonnes (comme bbs.center).
func center(text string) string {
	if len(text) >= oascii.Cols {
		return text
	}
	pad := (oascii.Cols - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}

// RenderHTML rend la page pageID du site en un bloc HTML (lignes séparées par
// des sauts de ligne, à insérer dans un <pre>). Erreur si la page est absente.
func RenderHTML(site *content.Site, pageID string) (string, error) {
	if site == nil {
		return "", fmt.Errorf("site nil")
	}
	p := site.Pages[pageID]
	if p == nil {
		return "", fmt.Errorf("page %q introuvable", pageID)
	}
	var b strings.Builder
	line := func(s string) { b.WriteString(s); b.WriteByte('\n') }

	switch {
	case p.Applet != "": // page applet auto-lancée (compat)
		line(rule())
		line(span(center(p.Title), oascii.Yellow))
		line(rule())
		line("")
		for _, ln := range p.Lines {
			line(lineSpan(ln))
		}
		line("")
		line(span("[applet: "+p.Applet+"]", oascii.Magenta))
	case len(p.Entries) > 0: // écran interactif (menu) : texte optionnel + choix
		line(rule())
		line(span(center(p.Title), oascii.Yellow))
		line(rule())
		line("")
		for _, ln := range p.Lines {
			line(lineSpan(ln))
		}
		if len(p.Lines) > 0 {
			line("")
		}
		for _, e := range p.Entries {
			line(span(" "+e.Key, oascii.Cyan) + span(" - "+e.Label, oascii.White))
		}
		line("")
		line(span("Votre choix", oascii.Green) + span("> ", oascii.White))
	default: // écran de contenu
		line(rule())
		line(span(center(p.Title), oascii.Yellow))
		line(rule())
		line("")
		for _, ln := range p.Lines {
			line(lineSpan(ln))
		}
		line("")
		line(span("Appuyez sur une touche...", oascii.Green))
	}
	return b.String(), nil
}
