// Package dwgrid produit le rendu OASCII d'une grille DataWindow (source unique,
// partagée par le serveur — applet interactif — et le studio Forge — aperçu de
// grille). Le rendu s'appuie sur le buffer différentiel oascii.Screen.
package dwgrid

import (
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Disposition de la grille DataWindow (lignes 0..27 de l'écran Oric).
const (
	titleRow  = 0 // bandeau titre
	headerRow = 2 // entête des colonnes
	dataTop   = 3 // première ligne de données
	footerRow = 24
	legendRow = 26
	contentX  = 1 // les colonnes 1..39 portent le texte (col 0 = attribut couleur)
)

// GridLignesMax renvoie le nombre de lignes de données affichables sur un écran.
func GridLignesMax(dw *content.DataWindow) int {
	n := dw.LignesMax
	max := footerRow - dataTop - 1
	if n <= 0 || n > max {
		n = max
	}
	return n
}

// cell tronque ou complète s à exactement n caractères.
func cell(s string, n int) string {
	if n < 0 {
		n = 0
	}
	if len(s) > n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// largeurCol renvoie la largeur configurée d'une colonne affichée (défaut partagé
// content.DefaultColWidth, aligné sur le budget vérifié par Site.Validate).
func largeurCol(dw *content.DataWindow, i int) int {
	if i < len(dw.Largeurs) && dw.Largeurs[i] > 0 {
		return dw.Largeurs[i]
	}
	return content.DefaultColWidth
}

// ligneGrille assemble le texte d'une ligne (index + cellules paddées).
func ligneGrille(dw *content.DataWindow, index string, valeurs func(col string) string) string {
	var b strings.Builder
	b.WriteString(cell(index, content.GridIndexWidth))
	for i, col := range dw.ColonnesAffichees {
		b.WriteString(cell(valeurs(col), largeurCol(dw, i)))
		b.WriteByte(' ')
	}
	s := b.String()
	if len(s) > oascii.Cols-contentX {
		s = s[:oascii.Cols-contentX]
	}
	return s
}

// ligneScroll rend la ligne sélectionnée avec un décalage horizontal : le contenu
// COMPLET (toutes les colonnes affichées, non tronqué, séparé par un espace) est
// décalé de scroll caractères pour révéler le texte coupé par les largeurs.
func ligneScroll(dw *content.DataWindow, index string, valeurs func(col string) string, scroll int) string {
	var full strings.Builder
	for i, col := range dw.ColonnesAffichees {
		if i > 0 {
			full.WriteByte(' ')
		}
		full.WriteString(valeurs(col))
	}
	data := full.String()
	if scroll > len(data) {
		scroll = len(data)
	}
	line := cell(index, content.GridIndexWidth) + data[scroll:]
	if len(line) > oascii.Cols-contentX {
		line = line[:oascii.Cols-contentX]
	}
	return line
}

// putLigne pose une ligne de texte à partir de contentX, avec une couleur d'encre
// (attribut en col 0) et, si inverse, le bit 7 sur chaque caractère.
func putLigne(scr *oascii.Screen, row int, ink oascii.Color, inverse bool, texte string) {
	scr.Put(0, row, oascii.InkAttr(ink))
	if inverse {
		bs := []byte(texte)
		for i := range bs {
			bs[i] |= 0x80
		}
		texte = string(bs)
	}
	scr.PutText(contentX, row, texte)
}

// RenderGrid compose la grille dans le buffer écran différentiel.
//   - rows      : lignes de la page courante (map colonne→texte)
//   - sel       : index de la ligne sélectionnée (0-based) dans rows
//   - page,total: pagination ; parPage = taille de page
//   - filtre    : filtre LIKE courant (affiché s'il est posé)
func RenderGrid(scr *oascii.Screen, dw *content.DataWindow, src content.SourceDonnees,
	rows []map[string]string, sel, page, parPage, total int, filtre, triLabel string, editable, downloadable bool, scroll int) {

	scr.Clear()
	inkEntete := content.Ink(dw.CouleurEntete)
	if dw.CouleurEntete == "" {
		inkEntete = oascii.Yellow
	}
	inkLignes := content.Ink(dw.CouleurLignes)
	if dw.CouleurLignes == "" {
		inkLignes = oascii.White
	}

	// Bandeau titre (inverse).
	titre := " " + strings.ToUpper(src.Table)
	if filtre != "" {
		titre += " /" + filtre
	}
	putLigne(scr, titleRow, oascii.Cyan, true, cell(titre, oascii.Cols-contentX))

	// Entête des colonnes.
	entete := ligneGrille(dw, "No", func(col string) string {
		if cd, ok := src.Colonnes[col]; ok && cd.Libelle != "" {
			return cd.Libelle
		}
		return col
	})
	putLigne(scr, headerRow, inkEntete, false, entete)

	// Lignes de données. Le numéro « No » est la position SUR LA PAGE (1..parPage),
	// pas l'index absolu : la colonne ne fait que GridIndexWidth (3) cases, or un
	// index absolu de grande table (100+, jusqu'à des milliers) collerait le titre
	// (plus d'espace séparateur) voire serait tronqué. Le contexte global est donné
	// par le pied « Page X/Y  N enreg. ».
	for i, r := range rows {
		row := dataTop + i
		if row >= footerRow {
			break
		}
		numPage := i + 1
		vals := func(col string) string { return r[col] }
		var texte string
		if i == sel && scroll > 0 { // ligne sélectionnée : scroll horizontal (texte complet décalé)
			texte = ligneScroll(dw, fmt.Sprintf("%d", numPage), vals, scroll)
		} else {
			texte = ligneGrille(dw, fmt.Sprintf("%d", numPage), vals)
		}
		putLigne(scr, row, inkLignes, i == sel, texte)
	}

	// Pied : pagination + total.
	nbPages := (total + parPage - 1) / parPage
	if nbPages < 1 {
		nbPages = 1
	}
	pied := fmt.Sprintf(" Page %d/%d  %d enreg.", page, nbPages, total)
	if triLabel != "" {
		pied += "  " + triLabel
	}
	putLigne(scr, footerRow, oascii.Green, false, cell(pied, oascii.Cols-contentX))

	// Légende des touches. Les flèches ^v<> sont rendues en VRAIS GLYPHES de la
	// police BBS (charset alternatif : ^=▲ v=▼ <=◄ >=►, cf. tools/genfont) en les
	// encadrant de l'attribut altCharset ($09 = ON, $08 = OFF) ; le reste de la
	// légende reste en charset standard. ▲▼ = sélection (aussi +/-), ◄► = scroll.
	// F/C = filtrer / effacer le filtre.
	const altOn, altOff = "\x09", "\x08"
	legende := altOn + "^v<>" + altOff + " S/R V=fiche"
	if editable {
		legende += " N/E/D"
	}
	if downloadable {
		legende += " X=DL"
	}
	legende += " F/C T Q"
	putLigne(scr, legendRow, oascii.Cyan, false, cell(legende, oascii.Cols-contentX))
}
