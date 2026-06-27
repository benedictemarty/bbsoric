// Package datawindow fournit le moteur de données « DataWindow » : des tables
// SQLite typées (colonnes, validation, données initiales) exposées en CRUD
// paginé/filtré/trié. Port du moteur telenet (serveur-go/datawindow.go), adapté
// à bbsoric : types de modèle dans internal/content, journalisation slog, lignes
// renvoyées en map[string]string (cellString gère le TEXT []byte de modernc).
//
// Mono-site : une seule base SQLite (DBName) dans le répertoire fourni. Le code
// de pool/verrou par base est conservé tel quel (éprouvé) ; on l'appelle toujours
// avec la même clé. Thread-safe : un verrou par base, pool de connexions.
package datawindow

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/internal/content"

	_ "modernc.org/sqlite"
)

// DBName est la clé de base unique (mono-site).
const DBName = "bbsoric"

// Engine est le moteur DataWindow (sources SQLite + sources API REST).
type Engine struct {
	dataDir string
	log     *slog.Logger
	locks   map[string]*sync.Mutex
	mu      sync.Mutex
	dbPool  map[string]*sql.DB
	poolMu  sync.RWMutex

	httpClient *http.Client                // pour les sources type_source="api"
	apiCache   map[string]apiCacheEntry    // URL -> réponse mise en cache
	apiMu      sync.Mutex
	now        func() time.Time            // horloge (injectable pour les tests de TTL)
}

// NewEngine crée le moteur ; les bases sont stockées dans dataDir.
func NewEngine(dataDir string, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Warn("datawindow: MkdirAll", "dir", dataDir, "err", err)
	}
	return &Engine{
		dataDir:    dataDir,
		log:        log,
		locks:      make(map[string]*sync.Mutex),
		dbPool:     make(map[string]*sql.DB),
		httpClient: &http.Client{Timeout: 8 * time.Second},
		apiCache:   make(map[string]apiCacheEntry),
		now:        time.Now,
	}
}

func (e *Engine) getLock(nomBase string) *sync.Mutex {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.locks[nomBase]; !ok {
		e.locks[nomBase] = &sync.Mutex{}
	}
	return e.locks[nomBase]
}

func (e *Engine) dbPath(nomBase string) string {
	return filepath.Join(e.dataDir, nomBase+".db")
}

// connect retourne une connexion depuis le pool, ou en ouvre une nouvelle.
func (e *Engine) connect(nomBase string) (*sql.DB, error) {
	e.poolMu.RLock()
	db, ok := e.dbPool[nomBase]
	e.poolMu.RUnlock()
	if ok {
		if err := db.Ping(); err == nil {
			return db, nil
		}
	}

	e.poolMu.Lock()
	defer e.poolMu.Unlock()
	if db, ok := e.dbPool[nomBase]; ok {
		if err := db.Ping(); err == nil {
			return db, nil
		}
		db.Close()
		delete(e.dbPool, nomBase)
	}

	db, err := sql.Open("sqlite", e.dbPath(nomBase))
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		e.log.Warn("datawindow: PRAGMA journal_mode=WAL", "err", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		e.log.Warn("datawindow: PRAGMA foreign_keys=ON", "err", err)
	}
	e.dbPool[nomBase] = db
	return db, nil
}

// Close ferme toutes les connexions du pool.
func (e *Engine) Close() {
	e.poolMu.Lock()
	defer e.poolMu.Unlock()
	for nom, db := range e.dbPool {
		if err := db.Close(); err != nil {
			e.log.Warn("datawindow: fermeture pool", "base", nom, "err", err)
		}
		delete(e.dbPool, nom)
	}
}

// echapperValeurDefaut échappe une valeur DEFAULT pour SQL (anti-injection).
func echapperValeurDefaut(v any) string {
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// convertirValeur convertit une valeur texte vers le type SQL cible.
func convertirValeur(val, typSQL string) any {
	switch typSQL {
	case "INTEGER":
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			return v
		}
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			return int64(v)
		}
		return val
	case "REAL":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			return v
		}
		return val
	default:
		return val
	}
}

// cellString rend une valeur scannée depuis SQLite en chaîne propre. modernc
// renvoie le TEXT en []byte : sans cette conversion la grille afficherait
// « [104 101 …] ». Centralise la normalisation des types.
func cellString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(x)
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", x)
	}
}

// scanRows lit toutes les lignes d'un *sql.Rows en map[string]string (cellString).
func scanRows(rows *sql.Rows) ([]map[string]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}
	var out []map[string]string
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		row := make(map[string]string, len(cols))
		for i, col := range cols {
			row[col] = cellString(values[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// InitialiserSource crée ou met à jour la table d'une source (idempotent), et
// importe les données initiales si la table est vide.
func (e *Engine) InitialiserSource(srcDef content.SourceDonnees) error {
	if srcDef.EstAPI() {
		return nil // source REST : rien à créer en base
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return err
	}

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // no-op si commit réussi

	var colsSQL []string
	for nomCol, colDef := range srcDef.Colonnes {
		if err := content.ValiderNomSQL(nomCol); err != nil {
			return err
		}
		typSQL := colDef.Type
		if typSQL == "" {
			typSQL = "TEXT"
		}
		if err := content.ValiderTypeSQL(typSQL); err != nil {
			return err
		}
		parts := []string{fmt.Sprintf(`"%s" %s`, nomCol, typSQL)}
		if colDef.ClePrimaire {
			parts = append(parts, "PRIMARY KEY")
			if colDef.AutoIncrement && typSQL == "INTEGER" {
				parts = append(parts, "AUTOINCREMENT")
			}
		}
		if colDef.Requis && !colDef.ClePrimaire {
			parts = append(parts, "NOT NULL")
		}
		if colDef.ValeurDefaut != nil {
			parts = append(parts, "DEFAULT "+echapperValeurDefaut(colDef.ValeurDefaut))
		}
		colsSQL = append(colsSQL, strings.Join(parts, " "))
	}

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s)`, table, strings.Join(colsSQL, ", "))
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Auto-migration : ajouter les colonnes manquantes.
	rows, err := tx.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
	if err != nil {
		return err
	}
	existantes := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			e.log.Warn("datawindow: scan table_info", "err", err)
			continue
		}
		existantes[name] = true
	}
	rows.Close()

	for nomCol, colDef := range srcDef.Colonnes {
		if existantes[nomCol] {
			continue
		}
		typSQL := colDef.Type
		if typSQL == "" {
			typSQL = "TEXT"
		}
		alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, table, nomCol, typSQL)
		if colDef.ValeurDefaut != nil {
			alterSQL += " DEFAULT " + echapperValeurDefaut(colDef.ValeurDefaut)
		}
		if _, err := tx.Exec(alterSQL); err != nil {
			e.log.Warn("datawindow: ALTER TABLE", "table", table, "col", nomCol, "err", err)
		} else {
			e.log.Info("datawindow: colonne ajoutée", "table", table, "col", nomCol)
		}
	}

	// Import initial si la table est vide.
	if len(srcDef.Donnees) > 0 {
		var count int
		if err := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)).Scan(&count); err != nil {
			e.log.Warn("datawindow: COUNT", "table", table, "err", err)
		}
		if count == 0 {
			if err := e.importerDonnees(tx, srcDef, srcDef.Donnees); err != nil {
				return fmt.Errorf("import initial: %w", err)
			}
			e.log.Info("datawindow: import initial", "table", table, "lignes", len(srcDef.Donnees))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// trouverClePrimaire renvoie la colonne clé primaire (ou "id" par défaut).
func trouverClePrimaire(srcDef content.SourceDonnees) string {
	for nomCol, colDef := range srcDef.Colonnes {
		if colDef.ClePrimaire {
			return nomCol
		}
	}
	return "id"
}

// importerDonnees insère plusieurs enregistrements via un prepared statement.
func (e *Engine) importerDonnees(tx *sql.Tx, srcDef content.SourceDonnees, donnees []map[string]any) error {
	table := srcDef.Table
	var cols, colNames []string
	for nomCol, colDef := range srcDef.Colonnes {
		if colDef.ClePrimaire && colDef.AutoIncrement {
			continue
		}
		cols = append(cols, fmt.Sprintf(`"%s"`, nomCol))
		colNames = append(colNames, nomCol)
	}
	if len(cols) == 0 {
		return nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	sqlStr := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`, table, strings.Join(cols, ","), placeholders)
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range donnees {
		champs := make(map[string]string, len(row))
		for k, v := range row {
			champs[k] = fmt.Sprintf("%v", v)
		}
		vals := make([]any, len(colNames))
		for i, nomCol := range colNames {
			colDef := srcDef.Colonnes[nomCol]
			switch {
			case colDef.AutoDate:
				vals[i] = time.Now().Format("2006-01-02 15:04:05")
			case champs[nomCol] != "":
				vals[i] = convertirValeur(champs[nomCol], colDef.Type)
			default:
				vals[i] = nil
			}
		}
		if _, err := stmt.Exec(vals...); err != nil {
			e.log.Warn("datawindow: insert initial", "table", table, "err", err)
		}
	}
	return nil
}

// Lister retourne les enregistrements (map colonne→texte), avec pagination,
// recherche globale LIKE (sur les colonnes TEXT) et tri, plus le total.
func (e *Engine) Lister(srcDef content.SourceDonnees, recherche, tri string, page, parPage int) ([]map[string]string, int, error) {
	if srcDef.EstAPI() {
		return e.listerAPI(srcDef, recherche, tri, page, parPage)
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return nil, 0, err
	}

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return nil, 0, err
	}

	var whereParts []string
	var params []any
	if recherche != "" {
		for nomCol, colDef := range srcDef.Colonnes {
			if colDef.Type == "TEXT" || colDef.Type == "" {
				whereParts = append(whereParts, fmt.Sprintf(`"%s" LIKE ?`, nomCol))
				params = append(params, "%"+recherche+"%")
			}
		}
	}
	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = "WHERE " + strings.Join(whereParts, " OR ")
	}

	var total int
	if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s" %s`, table, whereSQL), params...).Scan(&total); err != nil {
		e.log.Warn("datawindow: COUNT", "table", table, "err", err)
	}

	orderSQL := ""
	triStr := tri
	if triStr == "" {
		triStr = srcDef.TriDefaut
	}
	if triStr != "" {
		triParts := strings.Fields(triStr)
		if len(triParts) >= 1 && content.ValiderNomSQL(triParts[0]) == nil {
			dir := "ASC"
			if len(triParts) >= 2 {
				if d := strings.ToUpper(triParts[1]); d == "ASC" || d == "DESC" {
					dir = d
				}
			}
			orderSQL = fmt.Sprintf(`ORDER BY "%s" %s`, triParts[0], dir)
		}
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * parPage
	selectSQL := fmt.Sprintf(`SELECT * FROM "%s" %s %s LIMIT ? OFFSET ?`, table, whereSQL, orderSQL)
	params = append(params, parPage, offset)

	rows, err := db.Query(selectSQL, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	res, err := scanRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// Consulter retourne un enregistrement par sa clé primaire (nil si absent).
func (e *Engine) Consulter(srcDef content.SourceDonnees, cleValeur string) (map[string]string, error) {
	if srcDef.EstAPI() {
		return e.consulterAPI(srcDef, cleValeur)
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return nil, err
	}
	cleCol := trouverClePrimaire(srcDef)

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM "%s" WHERE "%s" = ?`, table, cleCol), cleValeur)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	return res[0], nil
}

// Creer insère un enregistrement (après validation) et renvoie son id.
func (e *Engine) Creer(srcDef content.SourceDonnees, champs map[string]string) (int64, error) {
	if srcDef.EstAPI() {
		return 0, errSourceLectureSeule
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return 0, err
	}

	champsComplets := make(map[string]string, len(champs))
	for k, v := range champs {
		champsComplets[k] = v
	}
	for nomCol, colDef := range srcDef.Colonnes {
		if colDef.AutoDate {
			champsComplets[nomCol] = time.Now().Format("2006-01-02 15:04:05")
		}
	}

	if erreurs := e.Valider(srcDef, champsComplets, false); len(erreurs) > 0 {
		return 0, fmt.Errorf("%s", erreurs[0])
	}

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return 0, err
	}

	var cols []string
	var vals []any
	for nomCol, colDef := range srcDef.Colonnes {
		if colDef.ClePrimaire && colDef.AutoIncrement {
			continue
		}
		if v, ok := champsComplets[nomCol]; ok {
			cols = append(cols, fmt.Sprintf(`"%s"`, nomCol))
			vals = append(vals, convertirValeur(v, colDef.Type))
		}
	}
	if len(cols) == 0 {
		return 0, fmt.Errorf("aucune valeur à insérer")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	sqlStr := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`, table, strings.Join(cols, ","), placeholders)
	result, err := db.Exec(sqlStr, vals...)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId() // SQLite fournit toujours LastInsertId
	return id, nil
}

// Modifier met à jour un enregistrement par sa clé primaire.
func (e *Engine) Modifier(srcDef content.SourceDonnees, cleValeur string, champs map[string]string) (bool, error) {
	if srcDef.EstAPI() {
		return false, errSourceLectureSeule
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return false, err
	}
	cleCol := trouverClePrimaire(srcDef)

	if erreurs := e.Valider(srcDef, champs, true); len(erreurs) > 0 {
		return false, fmt.Errorf("%s", erreurs[0])
	}

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return false, err
	}

	var setParts []string
	var params []any
	for nomCol, colDef := range srcDef.Colonnes {
		if nomCol == cleCol {
			continue
		}
		if v, ok := champs[nomCol]; ok {
			setParts = append(setParts, fmt.Sprintf(`"%s" = ?`, nomCol))
			params = append(params, convertirValeur(v, colDef.Type))
		}
	}
	if len(setParts) == 0 {
		return false, nil
	}
	params = append(params, cleValeur)

	sqlStr := fmt.Sprintf(`UPDATE "%s" SET %s WHERE "%s" = ?`, table, strings.Join(setParts, ", "), cleCol)
	result, err := db.Exec(sqlStr, params...)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// Supprimer supprime un enregistrement par sa clé primaire.
func (e *Engine) Supprimer(srcDef content.SourceDonnees, cleValeur string) (bool, error) {
	if srcDef.EstAPI() {
		return false, errSourceLectureSeule
	}
	table := srcDef.Table
	if err := content.ValiderNomSQL(table); err != nil {
		return false, err
	}
	cleCol := trouverClePrimaire(srcDef)

	lock := e.getLock(DBName)
	lock.Lock()
	defer lock.Unlock()

	db, err := e.connect(DBName)
	if err != nil {
		return false, err
	}

	result, err := db.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE "%s" = ?`, table, cleCol), cleValeur)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// Valider valide les champs avant écriture. partiel=true ignore les champs
// absents (mise à jour partielle). Renvoie la liste des messages d'erreur.
func (e *Engine) Valider(srcDef content.SourceDonnees, champs map[string]string, partiel bool) []string {
	var erreurs []string
	for nomCol, colDef := range srcDef.Colonnes {
		if (colDef.ClePrimaire && colDef.AutoIncrement) || colDef.AutoDate {
			continue
		}
		valeur, present := champs[nomCol]
		libelle := colDef.Libelle
		if libelle == "" {
			libelle = nomCol
		}
		if partiel && !present {
			continue
		}
		if colDef.Requis {
			if !present || strings.TrimSpace(valeur) == "" {
				erreurs = append(erreurs, libelle+" requis")
				continue
			}
		}
		if !present || valeur == "" {
			continue
		}
		if colDef.LongueurMax > 0 && len(valeur) > colDef.LongueurMax {
			erreurs = append(erreurs, fmt.Sprintf("%s trop long (%d max)", libelle, colDef.LongueurMax))
		}
		if colDef.Pattern != "" {
			if matched, _ := regexp.MatchString(colDef.Pattern, valeur); !matched {
				erreurs = append(erreurs, libelle+" format invalide")
			}
		}
		switch colDef.Type {
		case "INTEGER":
			if _, err := strconv.ParseInt(valeur, 10, 64); err != nil {
				erreurs = append(erreurs, libelle+" doit etre un nombre")
			}
		case "REAL":
			if _, err := strconv.ParseFloat(valeur, 64); err != nil {
				erreurs = append(erreurs, libelle+" doit etre un nombre")
			}
		}
	}
	return erreurs
}
