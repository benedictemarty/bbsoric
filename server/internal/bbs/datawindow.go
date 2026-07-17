package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/dwgrid"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/datawindow"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

func init() { Register("datawindow", dataWindowApplet) }

// Codes envoyés par les touches flèches du terminal Oric (cf. client/term.s,
// asciitab col 4). Choisis hors des octets ignorés par ReadKey ($0A/$0D/$00) et
// du backspace ($08). keyLeft/keyRight sont réservés au scroll horizontal.
const (
	keyUp    = 0x0B // flèche haut
	keyDown  = 0x0C // flèche bas
	keyLeft  = 0x0E // flèche gauche
	keyRight = 0x0F // flèche droite
)

// dataWindowApplet présente une source de données en grille paginée navigable
// au clavier, avec CRUD si la page est éditable. Touches :
//
//	+/-  bouger la sélection      S/R  page suivante/précédente
//	V    fiche détail             F    filtre LIKE   C  effacer le filtre
//	N/E/D créer/éditer/supprimer (si éditable et admin)   Q/ESC  quitter
//	X    télécharger le fichier de la ligne (si fichier_colonne défini) — catalogue
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

	parPage := dwgrid.GridLignesMax(dw)
	page, sel := 1, 0
	filtre := ""
	triEtat := 0 // 0 = tri par défaut ; sinon paires (colonne, ASC/DESC) cyclées par T
	var rows []map[string]string
	var total int

	// Filtre fixe éventuel de la page (vue par catégorie sans saisie utilisateur).
	var ff []content.FiltreFixe
	if dw.FiltreFixe != nil {
		ff = []content.FiltreFixe{*dw.FiltreFixe}
	}
	load := func() {
		tri := triString(dw, triEtat)
		var err error
		rows, total, err = eng.Lister(src, filtre, tri, page, parPage, ff...)
		if err == nil && len(rows) == 0 && total > 0 && page > 1 {
			page = 1
			rows, total, err = eng.Lister(src, filtre, tri, page, parPage, ff...)
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
	// L'écriture (CRUD) exige un compte administrateur : la lecture reste ouverte
	// à tous, mais seul un admin peut créer/éditer/supprimer (cf. ADR-0004, S11.5).
	editable := dw.Editable && ac.State.IsAdmin()
	// Catalogue : une colonne peut porter un nom de fichier téléchargeable (touche X).
	downloadable := dw.FichierColonne != "" && ac.State.Files != nil

	scr := oascii.NewScreen()
	draw := func() bool {
		dwgrid.RenderGrid(scr, dw, src, rows, sel, page, parPage, total, filtre, triLabel(src, dw, triEtat), editable, downloadable)
		return s.Write(string(scr.Render())) == nil
	}
	// Recharge complète de l'écran (après une saisie plein écran qui a brouillé
	// l'affichage côté terminal).
	redrawAll := func() bool { scr.Reset(); return draw() }

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
		case '+', keyDown: // descendre la sélection ('+' ou flèche bas)
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
		case '-', keyUp: // monter la sélection ('-' ou flèche haut)
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
		case 'T', 't': // cycler le tri : défaut → col0 ASC → col0 DESC → col1 ASC …
			triEtat = (triEtat + 1) % (2*len(dw.ColonnesAffichees) + 1)
			page, sel = 1, 0
			load()
			if !redrawAll() { // un tri réordonne la plupart des lignes
				return Outcome{Quit: true}
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
		case 'X', 'x': // télécharger le fichier de la ligne (catalogue)
			if downloadable && len(rows) > 0 {
				nom := strings.TrimSpace(rows[sel][dw.FichierColonne])
				if nom == "" {
					writeErr(s, "Aucun fichier pour cette entree.")
					anyKey(s)
				} else {
					sendFileDownload(s, ac.State.Files, nom)
				}
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

// dwDetail affiche une fiche détail plein écran (lecture seule). Les valeurs
// longues (ex. description) sont REPLIÉES sur plusieurs lignes (alignées sous la
// valeur) au lieu d'être tronquées à 22 colonnes.
func dwDetail(s *server.Session, src content.SourceDonnees, dw *content.DataWindow, row map[string]string) {
	const labW, valW, maxLines = 14, 22, 4
	header(s, "FICHE")
	b := oascii.New()
	indent := fmt.Sprintf(" %-*s ", labW, "") // alignement des lignes de suite
	for _, col := range colonnesOrdre(src, dw) {
		lib := col
		if cd, ok := src.Colonnes[col]; ok && cd.Libelle != "" {
			lib = cd.Libelle
		}
		lignes := wrapValeur(row[col], valW, maxLines)
		b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %-*s ", labW, trunc(lib, labW)))
		b.Ink(oascii.White).Text(lignes[0]).Newline()
		for _, suite := range lignes[1:] {
			b.Text(indent).Ink(oascii.White).Text(suite).Newline()
		}
	}
	b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	_ = s.Write(b.String())
	anyKey(s)
}

// wrapValeur replie une valeur en lignes d'au plus width caractères (au plus
// maxLines ; la dernière est marquée « ... » si la valeur est tronquée). Renvoie
// toujours au moins une ligne (vide si la valeur est vide).
func wrapValeur(v string, width, maxLines int) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return []string{""}
	}
	var out []string
	for len(v) > 0 && len(out) < maxLines {
		if len(v) <= width {
			out = append(out, v)
			v = ""
			break
		}
		out = append(out, v[:width])
		v = v[width:]
	}
	if v != "" { // reste tronqué : marquer la dernière ligne
		last := out[len(out)-1]
		if len(last) > 3 {
			out[len(out)-1] = last[:len(last)-3] + "..."
		}
	}
	return out
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

// triEtatColDir décode l'état de tri en (index de colonne, descendant). Renvoie
// ok=false pour l'état 0 (tri par défaut de la source).
func triEtatColDir(dw *content.DataWindow, etat int) (idx int, desc, ok bool) {
	if etat <= 0 || len(dw.ColonnesAffichees) == 0 {
		return 0, false, false
	}
	idx = (etat - 1) / 2
	desc = (etat-1)%2 == 1
	return idx, desc, true
}

// triString construit la clause de tri (« colonne ASC|DESC ») pour le moteur.
func triString(dw *content.DataWindow, etat int) string {
	idx, desc, ok := triEtatColDir(dw, etat)
	if !ok {
		return "" // le moteur applique TriDefaut
	}
	dir := "ASC"
	if desc {
		dir = "DESC"
	}
	return dw.ColonnesAffichees[idx] + " " + dir
}

// triLabel renvoie un libellé court du tri courant pour le pied de grille.
func triLabel(src content.SourceDonnees, dw *content.DataWindow, etat int) string {
	idx, desc, ok := triEtatColDir(dw, etat)
	if !ok {
		return ""
	}
	col := dw.ColonnesAffichees[idx]
	if cd, okc := src.Colonnes[col]; okc && cd.Libelle != "" {
		col = cd.Libelle
	}
	fleche := "+"
	if desc {
		fleche = "-"
	}
	return "tri " + col + fleche
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
