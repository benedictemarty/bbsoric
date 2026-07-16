package datawindow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/benedictemarty/bbsoric/internal/content"
)

// errSourceLectureSeule est renvoyée quand on tente d'écrire dans une source API.
var errSourceLectureSeule = errors.New("source API en lecture seule")

// apiCacheEntry mémorise la dernière réponse d'une URL et l'instant de récupération.
type apiCacheEntry struct {
	rows    []map[string]string
	fetched time.Time
}

// fetchAPI récupère (avec cache TTL) les enregistrements d'une source REST. La
// réponse JSON est soit un tableau d'objets, soit un objet dont la clé Racine
// contient le tableau. Chaque champ est normalisé en chaîne (cellString).
func (e *Engine) fetchAPI(src content.SourceDonnees) ([]map[string]string, error) {
	url := src.API.URL
	ttl := time.Duration(src.API.TTL) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	e.apiMu.Lock()
	if ent, ok := e.apiCache[url]; ok && e.now().Sub(ent.fetched) < ttl {
		rows := ent.rows
		e.apiMu.Unlock()
		return rows, nil
	}
	e.apiMu.Unlock()

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("api GET: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api statut %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // borne 1 Mo
	if err != nil {
		return nil, fmt.Errorf("api lecture: %w", err)
	}

	tableau, err := extraireTableau(body, src.API.Racine)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(tableau))
	for _, obj := range tableau {
		row := make(map[string]string, len(obj))
		for k, v := range obj {
			row[k] = jsonValeurString(v)
		}
		rows = append(rows, row)
	}

	e.apiMu.Lock()
	e.apiCache[url] = apiCacheEntry{rows: rows, fetched: e.now()}
	e.apiMu.Unlock()
	return rows, nil
}

// extraireTableau décode le corps JSON en tableau d'objets, éventuellement niché
// sous la clé racine.
func extraireTableau(body []byte, racine string) ([]map[string]any, error) {
	if racine == "" {
		var arr []map[string]any
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("api JSON (tableau attendu): %w", err)
		}
		return arr, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, fmt.Errorf("api JSON (objet attendu): %w", err)
	}
	brut, ok := obj[racine]
	if !ok {
		return nil, fmt.Errorf("api: clé racine %q absente", racine)
	}
	var arr []map[string]any
	if err := json.Unmarshal(brut, &arr); err != nil {
		return nil, fmt.Errorf("api JSON (tableau sous %q): %w", racine, err)
	}
	return arr, nil
}

// jsonValeurString rend une valeur JSON décodée en chaîne propre.
func jsonValeurString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "1"
		}
		return "0"
	case float64:
		// Entier si pas de partie fractionnaire.
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// listerAPI applique recherche (sous-chaîne), tri et pagination côté client sur
// les enregistrements récupérés de l'API. Renvoie la page et le total filtré.
func (e *Engine) listerAPI(src content.SourceDonnees, recherche, tri string, page, parPage int, filtreFixe ...content.FiltreFixe) ([]map[string]string, int, error) {
	rows, err := e.fetchAPI(src)
	if err != nil {
		return nil, 0, err
	}

	// Filtre fixe de la page (égalité exacte sur une colonne) : appliqué en mémoire.
	if len(filtreFixe) > 0 && filtreFixe[0].Colonne != "" {
		col, val := filtreFixe[0].Colonne, filtreFixe[0].Valeur
		var gardees []map[string]string
		for _, r := range rows {
			if r[col] == val {
				gardees = append(gardees, r)
			}
		}
		rows = gardees
	}

	// Filtre : sous-chaîne (insensible à la casse) sur n'importe quelle colonne.
	if recherche != "" {
		rech := strings.ToLower(recherche)
		var filtre []map[string]string
		for _, r := range rows {
			for _, v := range r {
				if strings.Contains(strings.ToLower(v), rech) {
					filtre = append(filtre, r)
					break
				}
			}
		}
		rows = filtre
	}

	// Tri : "colonne ASC|DESC" (défaut TriDefaut). Comparaison numérique si les
	// deux valeurs sont des nombres, sinon lexicographique.
	triStr := tri
	if triStr == "" {
		triStr = src.TriDefaut
	}
	if champs := strings.Fields(triStr); len(champs) >= 1 {
		col := champs[0]
		desc := len(champs) >= 2 && strings.ToUpper(champs[1]) == "DESC"
		sort.SliceStable(rows, func(i, j int) bool {
			a, b := rows[i][col], rows[j][col]
			less := comparerCellule(a, b)
			if desc {
				return !less && a != b
			}
			return less
		})
	}

	total := len(rows)
	if page < 1 {
		page = 1
	}
	debut := (page - 1) * parPage
	if debut >= total {
		return nil, total, nil
	}
	fin := debut + parPage
	if fin > total {
		fin = total
	}
	return rows[debut:fin], total, nil
}

// comparerCellule renvoie true si a < b (numérique si possible, sinon lexicographique).
func comparerCellule(a, b string) bool {
	fa, ea := strconv.ParseFloat(a, 64)
	fb, eb := strconv.ParseFloat(b, 64)
	if ea == nil && eb == nil {
		return fa < fb
	}
	return a < b
}

// consulterAPI retrouve un enregistrement par sa clé primaire dans la source API.
func (e *Engine) consulterAPI(src content.SourceDonnees, cleValeur string) (map[string]string, error) {
	rows, err := e.fetchAPI(src)
	if err != nil {
		return nil, err
	}
	cle := trouverClePrimaire(src)
	for _, r := range rows {
		if r[cle] == cleValeur {
			return r, nil
		}
	}
	return nil, nil
}
