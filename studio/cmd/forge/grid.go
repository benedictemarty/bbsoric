package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/dwgrid"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// handleGrid : POST /api/grid?page=<pageId>&n=<pageNum>&sel=<sel>&filtre=<f>
// (corps = site JSON). Rend la grille DataWindow de la page à partir des données
// SEED de la source (aperçu interactif du studio) avec le MÊME rendu que le serveur
// (internal/dwgrid) et renvoie le buffer écran 40×28 brut (que le simulateur du
// studio affiche directement). Le vrai serveur pagine via SQLite ; ici on pagine
// les données seed en mémoire — suffisant pour un aperçu.
func (s *server) handleGrid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	body, err := readBody(r)
	if err != nil {
		http.Error(w, "lecture du corps : "+err.Error(), http.StatusBadRequest)
		return
	}
	var site content.Site
	if err := json.Unmarshal(body, &site); err != nil {
		http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
		return
	}
	p := site.Pages[r.URL.Query().Get("page")]
	if p == nil || p.DataWindow == nil {
		http.Error(w, "page grille introuvable", http.StatusBadRequest)
		return
	}
	dw := p.DataWindow
	src, ok := site.SourcesDonnees[dw.Source]
	if !ok {
		http.Error(w, "source « "+dw.Source+" » introuvable", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	pageNum := atoiDefault(q.Get("n"), 1)
	sel := atoiDefault(q.Get("sel"), 0)
	filtre := strings.TrimSpace(q.Get("filtre"))

	rows, total := previewLister(src, dw, filtre, pageNum)
	parPage := dwgrid.GridLignesMax(dw)
	if sel < 0 {
		sel = 0
	}
	if sel >= len(rows) {
		sel = len(rows) - 1
	}

	scr := oascii.NewScreen()
	dwgrid.RenderGrid(scr, dw, src, rows, sel, pageNum, parPage, total, filtre,
		"", dw.Editable, dw.FichierColonne != "")
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(scr.Buffer())
}

// previewLister applique en mémoire filtre fixe + filtre utilisateur (LIKE) + tri
// par défaut + pagination aux données seed de la source.
func previewLister(src content.SourceDonnees, dw *content.DataWindow, filtre string, pageNum int) (rows []map[string]string, total int) {
	parPage := dwgrid.GridLignesMax(dw)
	all := make([]map[string]string, 0, len(src.Donnees))
	for _, d := range src.Donnees {
		m := make(map[string]string, len(d))
		for k, v := range d {
			m[k] = fmt.Sprint(v)
		}
		all = append(all, m)
	}
	// Filtre fixe de la page (égalité sur une colonne).
	if dw.FiltreFixe != nil && dw.FiltreFixe.Colonne != "" {
		var kept []map[string]string
		for _, m := range all {
			if m[dw.FiltreFixe.Colonne] == dw.FiltreFixe.Valeur {
				kept = append(kept, m)
			}
		}
		all = kept
	}
	// Filtre utilisateur : sous-chaîne (insensible à la casse) sur colonnes TEXT.
	if filtre != "" {
		f := strings.ToLower(filtre)
		var kept []map[string]string
		for _, m := range all {
			for col, cd := range src.Colonnes {
				if cd.Type == "TEXT" || cd.Type == "" {
					if strings.Contains(strings.ToLower(m[col]), f) {
						kept = append(kept, m)
						break
					}
				}
			}
		}
		all = kept
	}
	// Tri par défaut (lexicographique — aperçu ; le serveur trie via SQLite).
	if champs := strings.Fields(src.TriDefaut); len(champs) >= 1 {
		col := champs[0]
		desc := len(champs) >= 2 && strings.EqualFold(champs[1], "DESC")
		sort.SliceStable(all, func(i, j int) bool {
			a, b := all[i][col], all[j][col]
			if desc {
				return a > b
			}
			return a < b
		})
	}
	total = len(all)
	if pageNum < 1 {
		pageNum = 1
	}
	start := (pageNum - 1) * parPage
	if start >= total {
		return nil, total
	}
	end := start + parPage
	if end > total {
		end = total
	}
	return all[start:end], total
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
