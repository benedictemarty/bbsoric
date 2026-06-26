package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/datawindow"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

func init() { Register("datawindow", dataWindowApplet) }

// dataWindowApplet présente une source de données en grille paginée navigable
// au clavier, avec CRUD si la page est éditable. Touches :
//
//	+/-  bouger la sélection      S/R  page suivante/précédente
//	V    fiche détail             F    filtre LIKE   C  effacer le filtre
//	N/E/D créer/éditer/supprimer (si éditable et connecté)   Q/ESC  quitter
func dataWindowApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	dw := ac.Page.DataWindow
	if dw == nil || ac.Site == nil {
		writeErr(s, "Grille indisponible.")
		anyKey(s)
		return Outcome{}
	}
	src, ok := ac.Site.SourcesDonnees[dw.Source]
	if !ok {
		writeErr(s, "Source \""+dw.Source+"\" introuvable.")
		anyKey(s)
		return Outcome{}
	}
	eng := ac.State.Data
	if eng == nil {
		writeErr(s, "Moteur de donnees desactive (flag -data).")
		anyKey(s)
		return Outcome{}
	}

	parPage := gridLignesMax(dw)
	page, sel := 1, 0
	filtre := ""
	var rows []map[string]string
	var total int

	load := func() {
		var err error
		rows, total, err = eng.Lister(src, filtre, "", page, parPage)
		if err == nil && len(rows) == 0 && total > 0 && page > 1 {
			page = 1
			rows, total, err = eng.Lister(src, filtre, "", page, parPage)
		}
		if err != nil {
			rows, total = nil, 0
		}
		if sel >= len(rows) {
			sel = len(rows) - 1
		}
		if sel < 0 {
			sel = 0
		}
	}
	nbPages := func() int {
		if n := (total + parPage - 1) / parPage; n > 1 {
			return n
		}
		return 1
	}
	scr := oascii.NewScreen()
	draw := func() bool {
		renderGrid(scr, dw, src, rows, sel, page, parPage, total, filtre)
		return s.Write(string(scr.Render())) == nil
	}
	// Recharge complète de l'écran (après une saisie plein écran qui a brouillé
	// l'affichage côté terminal).
	redrawAll := func() bool { scr.Reset(); return draw() }

	editable := dw.Editable && ac.State.LoggedIn()

	load()
	if !redrawAll() {
		return Outcome{Quit: true}
	}

	for {
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		switch key {
		case '+': // descendre la sélection
			if sel < len(rows)-1 {
				sel++
			} else if page < nbPages() {
				page++
				sel = 0
				load()
			}
			if !draw() {
				return Outcome{Quit: true}
			}
		case '-': // monter la sélection
			if sel > 0 {
				sel--
			} else if page > 1 {
				page--
				load()
				sel = len(rows) - 1
			}
			if !draw() {
				return Outcome{Quit: true}
			}
		case 'S', 's': // page suivante
			if page < nbPages() {
				page++
				sel = 0
				load()
				if !draw() {
					return Outcome{Quit: true}
				}
			}
		case 'R', 'r': // page précédente
			if page > 1 {
				page--
				sel = 0
				load()
				if !draw() {
					return Outcome{Quit: true}
				}
			}
		case 'F', 'f': // poser un filtre LIKE
			val, err := prompt(s, "Filtre (vide=tout)")
			if err != nil {
				return Outcome{Quit: true}
			}
			filtre = strings.TrimSpace(val)
			page, sel = 1, 0
			load()
			if !redrawAll() {
				return Outcome{Quit: true}
			}
		case 'C', 'c': // effacer le filtre
			if filtre != "" {
				filtre = ""
				page, sel = 1, 0
				load()
				if !redrawAll() {
					return Outcome{Quit: true}
				}
			}
		case 'V', 'v': // fiche détail
			if len(rows) > 0 {
				dwDetail(s, src, dw, rows[sel])
			}
			if !redrawAll() {
				return Outcome{Quit: true}
			}
		case 'N', 'n': // créer
			if editable {
				dwCreer(s, eng, src, dw)
				load()
				if !redrawAll() {
					return Outcome{Quit: true}
				}
			}
		case 'E', 'e': // éditer
			if editable && len(rows) > 0 {
				dwEditer(s, eng, src, dw, rows[sel])
				load()
				if !redrawAll() {
					return Outcome{Quit: true}
				}
			}
		case 'D', 'd': // supprimer
			if editable && len(rows) > 0 {
				dwSupprimer(s, eng, src, rows[sel])
				load()
				if !redrawAll() {
					return Outcome{Quit: true}
				}
			}
		case 'Q', 'q', 27: // quitter
			return Outcome{Done: true}
		}
	}
}

// colonnesOrdre renvoie les colonnes de la source dans un ordre stable
// (colonnes affichées d'abord, puis le reste), pour des écrans déterministes.
func colonnesOrdre(src content.SourceDonnees, dw *content.DataWindow) []string {
	vu := map[string]bool{}
	var ordre []string
	if dw != nil {
		for _, c := range dw.ColonnesAffichees {
			if _, ok := src.Colonnes[c]; ok && !vu[c] {
				ordre = append(ordre, c)
				vu[c] = true
			}
		}
	}
	// Le reste, trié pour la stabilité.
	var reste []string
	for c := range src.Colonnes {
		if !vu[c] {
			reste = append(reste, c)
		}
	}
	sortStrings(reste)
	return append(ordre, reste...)
}

// dwDetail affiche une fiche détail plein écran (lecture seule).
func dwDetail(s *server.Session, src content.SourceDonnees, dw *content.DataWindow, row map[string]string) {
	header(s, "FICHE")
	b := oascii.New()
	for _, col := range colonnesOrdre(src, dw) {
		lib := col
		if cd, ok := src.Colonnes[col]; ok && cd.Libelle != "" {
			lib = cd.Libelle
		}
		b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %-14s ", trunc(lib, 14)))
		b.Ink(oascii.White).Text(trunc(row[col], 22)).Newline()
	}
	b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	_ = s.Write(b.String())
	anyKey(s)
}

// champsSaisissables renvoie les colonnes à saisir (hors clé auto et auto-date).
func champsSaisissables(src content.SourceDonnees, dw *content.DataWindow) []string {
	var cols []string
	for _, col := range colonnesOrdre(src, dw) {
		cd := src.Colonnes[col]
		if (cd.ClePrimaire && cd.AutoIncrement) || cd.AutoDate {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}

// dwCreer saisit un nouvel enregistrement (boucle jusqu'à succès ou annulation).
func dwCreer(s *server.Session, eng *datawindow.Engine, src content.SourceDonnees, dw *content.DataWindow) {
	header(s, "NOUVEAU")
	cols := champsSaisissables(src, dw)
	for {
		champs := map[string]string{}
		annule := false
		for i, col := range cols {
			cd := src.Colonnes[col]
			lib := cd.Libelle
			if lib == "" {
				lib = col
			}
			val, err := prompt(s, lib)
			if err != nil {
				return
			}
			val = strings.TrimSpace(val)
			if i == 0 && val == "" {
				annule = true // premier champ vide = annulation
				break
			}
			champs[col] = val
		}
		if annule {
			return
		}
		if _, err := eng.Creer(src, champs); err != nil {
			writeErr(s, err.Error())
			continue // ressaisir
		}
		ok := oascii.New()
		ok.Ink(oascii.Green).Text(" Enregistrement cree.").Newline()
		_ = s.Write(ok.String())
		anyKey(s)
		return
	}
}

// dwEditer édite l'enregistrement sélectionné (RETURN vide = garder la valeur).
func dwEditer(s *server.Session, eng *datawindow.Engine, src content.SourceDonnees, dw *content.DataWindow, row map[string]string) {
	header(s, "EDITION")
	cle := row[dwClePrimaire(src)]
	for {
		champs := map[string]string{}
		for _, col := range champsSaisissables(src, dw) {
			cd := src.Colonnes[col]
			lib := cd.Libelle
			if lib == "" {
				lib = col
			}
			val, err := prompt(s, fmt.Sprintf("%s [%s]", lib, trunc(row[col], 16)))
			if err != nil {
				return
			}
			if v := strings.TrimSpace(val); v != "" {
				champs[col] = v
			}
		}
		if len(champs) == 0 {
			return // rien modifié
		}
		if _, err := eng.Modifier(src, cle, champs); err != nil {
			writeErr(s, err.Error())
			continue
		}
		ok := oascii.New()
		ok.Ink(oascii.Green).Text(" Enregistrement modifie.").Newline()
		_ = s.Write(ok.String())
		anyKey(s)
		return
	}
}

// dwSupprimer supprime l'enregistrement sélectionné après confirmation.
func dwSupprimer(s *server.Session, eng *datawindow.Engine, src content.SourceDonnees, row map[string]string) {
	val, err := prompt(s, "Supprimer cet enregistrement ? (O/N)")
	if err != nil {
		return
	}
	if v := strings.ToUpper(strings.TrimSpace(val)); v != "O" {
		return
	}
	if _, err := eng.Supprimer(src, row[dwClePrimaire(src)]); err != nil {
		writeErr(s, err.Error())
		anyKey(s)
	}
}

// dwClePrimaire renvoie le nom de la colonne clé primaire (ou "id").
func dwClePrimaire(src content.SourceDonnees) string {
	for nom, cd := range src.Colonnes {
		if cd.ClePrimaire {
			return nom
		}
	}
	return "id"
}

func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
