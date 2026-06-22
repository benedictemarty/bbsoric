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
			line(span(ln.Text, content.Ink(ln.Ink)))
		}
		line("")
		line(span("[applet: "+p.Applet+"]", oascii.Magenta))
	case len(p.Entries) > 0: // écran interactif (menu) : texte optionnel + choix
		line(rule())
		line(span(center(p.Title), oascii.Yellow))
		line(rule())
		line("")
		for _, ln := range p.Lines {
			line(span(ln.Text, content.Ink(ln.Ink)))
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
			line(span(ln.Text, content.Ink(ln.Ink)))
		}
		line("")
		line(span("Appuyez sur une touche...", oascii.Green))
	}
	return b.String(), nil
}
