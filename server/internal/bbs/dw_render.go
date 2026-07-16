package bbs

import (
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Disposition de la grille DataWindow (lignes 0..27 de l'écran Oric).
const (
	gridTitleRow  = 0 // bandeau titre
	gridHeaderRow = 2 // entête des colonnes
	gridDataTop   = 3 // première ligne de données
	gridFooterRow = 24
	gridLegendRow = 26
	gridContentX  = 1 // les colonnes 1..39 portent le texte (col 0 = attribut couleur)
)

// gridLignesMax renvoie le nombre de lignes de données affichables.
func gridLignesMax(dw *content.DataWindow) int {
	n := dw.LignesMax
	max := gridFooterRow - gridDataTop - 1
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
	if len(s) > oascii.Cols-gridContentX {
		s = s[:oascii.Cols-gridContentX]
	}
	return s
}

// putLigne pose une ligne de texte à partir de gridContentX, avec une couleur
// d'encre (attribut en col 0) et, si inverse, le bit 7 sur chaque caractère.
func putLigne(scr *oascii.Screen, row int, ink oascii.Color, inverse bool, texte string) {
	scr.Put(0, row, oascii.InkAttr(ink))
	if inverse {
		bs := []byte(texte)
		for i := range bs {
			bs[i] |= 0x80
		}
		texte = string(bs)
	}
	scr.PutText(gridContentX, row, texte)
}

// renderGrid compose la grille dans le buffer écran différentiel.
//   - rows      : lignes de la page courante (map colonne→texte)
//   - sel       : index de la ligne sélectionnée (0-based) dans rows
//   - page,total: pagination ; parPage = taille de page
//   - filtre    : filtre LIKE courant (affiché s'il est posé)
func renderGrid(scr *oascii.Screen, dw *content.DataWindow, src content.SourceDonnees,
	rows []map[string]string, sel, page, parPage, total int, filtre, triLabel string, editable bool) {

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
	putLigne(scr, gridTitleRow, oascii.Cyan, true, cell(titre, oascii.Cols-gridContentX))

	// Entête des colonnes.
	entete := ligneGrille(dw, "No", func(col string) string {
		if cd, ok := src.Colonnes[col]; ok && cd.Libelle != "" {
			return cd.Libelle
		}
		return col
	})
	putLigne(scr, gridHeaderRow, inkEntete, false, entete)

	// Lignes de données.
	for i, r := range rows {
		row := gridDataTop + i
		if row >= gridFooterRow {
			break
		}
		numAbs := (page-1)*parPage + i + 1
		texte := ligneGrille(dw, fmt.Sprintf("%d", numAbs), func(col string) string { return r[col] })
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
	putLigne(scr, gridFooterRow, oascii.Green, false, cell(pied, oascii.Cols-gridContentX))

	// Légende des touches.
	legende := "+/- S/R V=fiche"
	if editable {
		legende += " N/E/D"
	}
	legende += " F/T Q"
	putLigne(scr, gridLegendRow, oascii.Cyan, false, cell(legende, oascii.Cols-gridContentX))
}
