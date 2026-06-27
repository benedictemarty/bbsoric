package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// --- Modèle DataWindow (porté de telenet/serveur-go/structures.go) ---
//
// Une SourceDonnees décrit une table (colonnes typées, données initiales) ; un
// descripteur DataWindow attaché à une page la présente sous forme de grille
// paginée navigable (CRUD côté applet). Le moteur SQLite (server/internal/
// datawindow) consomme ces types ; ils vivent ici pour que Site.Validate() les
// vérifie au chargement (et que le studio puisse les éditer plus tard).

// ColonneDef définit une colonne d'une source de données.
type ColonneDef struct {
	Type          string `json:"type"`                    // type SQL (TEXT par défaut)
	Libelle       string `json:"libelle"`                 // libellé affiché
	ClePrimaire   bool   `json:"cle_primaire"`            // clé primaire
	Requis        bool   `json:"requis"`                  // NOT NULL / validation requise
	LongueurMax   int    `json:"longueur_max"`            // longueur max (0 = illimité)
	AutoIncrement bool   `json:"auto_increment,omitempty"` // AUTOINCREMENT (INTEGER PK)
	Pattern       string `json:"pattern,omitempty"`       // regex de validation
	ValeurDefaut  any    `json:"valeur_defaut,omitempty"` // DEFAULT SQL
	AutoDate      bool   `json:"auto_date,omitempty"`     // rempli à la date courante
	Masque        string `json:"masque,omitempty"`        // masque de saisie (réservé)
}

// SourceDonnees définit une source de données. Par défaut une table SQLite
// (colonnes + données initiales) ; si TypeSource == "api", une source REST en
// lecture seule (API) dont les enregistrements proviennent d'un endpoint JSON.
type SourceDonnees struct {
	Table         string                `json:"table"`
	Colonnes      map[string]ColonneDef `json:"colonnes"`
	TriDefaut     string                `json:"tri_defaut"`
	LignesParPage int                   `json:"lignes_par_page"`
	Donnees       []map[string]any      `json:"donnees,omitempty"`     // seed SQLite (importé si table vide)
	TypeSource    string                `json:"type_source,omitempty"` // "" = sqlite (défaut), "api" = REST
	API           *APIConfig            `json:"api,omitempty"`         // config si TypeSource == "api"
}

// APIConfig décrit une source REST en lecture seule. L'endpoint renvoie un JSON ;
// Racine (optionnel) est la clé contenant le tableau d'objets (sinon le JSON est
// lui-même un tableau). Chaque objet mappe ses champs sur les colonnes par nom.
type APIConfig struct {
	URL    string `json:"url"`
	Racine string `json:"racine,omitempty"`    // clé du tableau dans la réponse (ex. "results")
	TTL    int    `json:"ttl_sec,omitempty"`   // durée de cache en secondes (défaut 60)
}

// EstAPI indique si la source est une source REST (lecture seule).
func (src SourceDonnees) EstAPI() bool { return src.TypeSource == "api" }

// DataWindow est le descripteur de page qui présente une source en grille.
type DataWindow struct {
	Source            string   `json:"source"`              // clé dans Site.SourcesDonnees
	ColonnesAffichees []string `json:"colonnes_affichees"`  // colonnes montrées dans la grille
	Largeurs          []int    `json:"largeurs,omitempty"`  // largeur de chaque colonne (en cases)
	CouleurEntete     string   `json:"couleur_entete,omitempty"`
	CouleurLignes     string   `json:"couleur_lignes,omitempty"`
	CouleurSelection  string   `json:"couleur_selection,omitempty"`
	LignesMax         int      `json:"lignes_max,omitempty"` // lignes de données par écran
	Editable          bool     `json:"editable,omitempty"`   // autorise N/E/D (créer/éditer/supprimer)
}

// --- Gardes anti-injection SQL (partagées avec le moteur) ---

// ErrValidationSQL signale un identifiant SQL invalide.
var ErrValidationSQL = errors.New("identifiant SQL invalide")

var nomSQLValide = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// typesSQL : seuls les types SQLite standards sont autorisés.
var typesSQL = map[string]bool{
	"TEXT": true, "INTEGER": true, "REAL": true, "BLOB": true,
	"NUMERIC": true, "BOOLEAN": true, "DATE": true, "DATETIME": true,
}

// ValiderNomSQL vérifie qu'un nom de table/colonne est un identifiant sûr
// (jamais interpolé en SQL sans cette garde).
func ValiderNomSQL(nom string) error {
	if nom == "" || !nomSQLValide.MatchString(nom) {
		return fmt.Errorf("%q : %w", nom, ErrValidationSQL)
	}
	return nil
}

// ValiderTypeSQL vérifie qu'un type est dans la liste blanche.
func ValiderTypeSQL(typ string) error {
	if typ == "" {
		return nil // défaut TEXT
	}
	if !typesSQL[strings.ToUpper(strings.TrimSpace(typ))] {
		return fmt.Errorf("type SQL non autorisé : %q", typ)
	}
	return nil
}

// GridIndexWidth est la largeur de la colonne d'index ("NN ") en tête de ligne.
const GridIndexWidth = 3

// validate vérifie une source de données. Source SQLite : noms et types sûrs.
// Source API : URL requise (lecture seule, pas de table SQL).
func (src SourceDonnees) validate(nomSrc string) error {
	if len(src.Colonnes) == 0 {
		return fmt.Errorf("source %q : aucune colonne", nomSrc)
	}
	if src.EstAPI() {
		if src.API == nil || src.API.URL == "" {
			return fmt.Errorf("source %q : type_source=api exige api.url", nomSrc)
		}
		return nil
	}
	if err := ValiderNomSQL(src.Table); err != nil {
		return fmt.Errorf("source %q : table %w", nomSrc, err)
	}
	for nomCol, col := range src.Colonnes {
		if err := ValiderNomSQL(nomCol); err != nil {
			return fmt.Errorf("source %q : colonne %w", nomSrc, err)
		}
		if err := ValiderTypeSQL(col.Type); err != nil {
			return fmt.Errorf("source %q.%s : %w", nomSrc, nomCol, err)
		}
	}
	return nil
}

// validate vérifie un descripteur DataWindow de page : source existante,
// colonnes affichées déclarées, largeurs cohérentes, et budget 40 colonnes.
func (dw *DataWindow) validate(pageID string, s *Site) error {
	src, ok := s.SourcesDonnees[dw.Source]
	if !ok {
		return fmt.Errorf("page %q : source DataWindow %q introuvable", pageID, dw.Source)
	}
	if len(dw.ColonnesAffichees) == 0 {
		return fmt.Errorf("page %q : datawindow sans 'colonnes_affichees'", pageID)
	}
	for _, col := range dw.ColonnesAffichees {
		if _, ok := src.Colonnes[col]; !ok {
			return fmt.Errorf("page %q : colonne affichée %q absente de la source %q", pageID, col, dw.Source)
		}
	}
	if len(dw.Largeurs) != 0 && len(dw.Largeurs) != len(dw.ColonnesAffichees) {
		return fmt.Errorf("page %q : 'largeurs' (%d) doit correspondre à 'colonnes_affichees' (%d)",
			pageID, len(dw.Largeurs), len(dw.ColonnesAffichees))
	}
	// Budget de largeur : col 0 (attribut couleur) + index + Σ(largeur+1) ≤ 40.
	total := 1 + GridIndexWidth
	for i := range dw.ColonnesAffichees {
		largeur := 8 // défaut si non précisé
		if i < len(dw.Largeurs) {
			largeur = dw.Largeurs[i]
		}
		if largeur < 1 {
			return fmt.Errorf("page %q : largeur de colonne %d invalide (%d)", pageID, i, largeur)
		}
		total += largeur + 1
	}
	if total > 40 {
		return fmt.Errorf("page %q : grille trop large (%d > 40 colonnes)", pageID, total)
	}
	return nil
}
