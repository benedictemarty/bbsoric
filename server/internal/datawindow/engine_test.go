package datawindow

import (
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	e := NewEngine(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(e.Close)
	return e
}

func repertoire() content.SourceDonnees {
	return content.SourceDonnees{
		Table:     "repertoire",
		TriDefaut: "nom ASC",
		Colonnes: map[string]content.ColonneDef{
			"id":    {Type: "INTEGER", Libelle: "ID", ClePrimaire: true, AutoIncrement: true},
			"nom":   {Type: "TEXT", Libelle: "Nom", Requis: true, LongueurMax: 20},
			"ville": {Type: "TEXT", Libelle: "Ville", LongueurMax: 15},
			"note":  {Type: "INTEGER", Libelle: "Note"},
		},
		Donnees: []map[string]any{
			{"nom": "Alice", "ville": "Lyon", "note": 5},
			{"nom": "Bob", "ville": "Paris", "note": 3},
			{"nom": "Charlie", "ville": "Lyon", "note": 4},
		},
	}
}

func TestInitialiserEtSeed(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	if err := e.InitialiserSource(src); err != nil {
		t.Fatalf("init: %v", err)
	}
	rows, total, err := e.Lister(src, "", "", 1, 10)
	if err != nil {
		t.Fatalf("lister: %v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Fatalf("3 lignes attendues, total=%d len=%d", total, len(rows))
	}
	// Idempotent : ré-init ne ré-importe pas (le total reste 3).
	if err := e.InitialiserSource(src); err != nil {
		t.Fatalf("ré-init: %v", err)
	}
	if _, total, _ := e.Lister(src, "", "", 1, 10); total != 3 {
		t.Errorf("ré-init a dupliqué les données : total=%d", total)
	}
}

func TestListerTriPaginationFiltre(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	if err := e.InitialiserSource(src); err != nil {
		t.Fatal(err)
	}
	// Tri par défaut nom ASC -> Alice en premier.
	rows, _, _ := e.Lister(src, "", "", 1, 10)
	if rows[0]["nom"] != "Alice" {
		t.Errorf("tri ASC : premier=%q, attendu Alice", rows[0]["nom"])
	}
	// Tri descendant.
	rows, _, _ = e.Lister(src, "", "nom DESC", 1, 10)
	if rows[0]["nom"] != "Charlie" {
		t.Errorf("tri DESC : premier=%q, attendu Charlie", rows[0]["nom"])
	}
	// Pagination : 2 par page.
	p1, total, _ := e.Lister(src, "", "nom ASC", 1, 2)
	p2, _, _ := e.Lister(src, "", "nom ASC", 2, 2)
	if total != 3 || len(p1) != 2 || len(p2) != 1 {
		t.Errorf("pagination : total=%d p1=%d p2=%d", total, len(p1), len(p2))
	}
	// Filtre LIKE global sur les colonnes TEXT.
	f, ftotal, _ := e.Lister(src, "Lyon", "nom ASC", 1, 10)
	if ftotal != 2 || len(f) != 2 {
		t.Errorf("filtre Lyon : total=%d len=%d (attendu 2)", ftotal, len(f))
	}
}

// TestListerFiltreFixe : le filtre fixe d'une page (égalité sur une colonne) ne
// montre que les lignes correspondantes, et se combine (AND) avec le filtre F (J3).
func TestListerFiltreFixe(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	if err := e.InitialiserSource(src); err != nil {
		t.Fatal(err)
	}
	ffLyon := content.FiltreFixe{Colonne: "ville", Valeur: "Lyon"}

	// Filtre fixe seul : Lyon -> Alice + Charlie (2).
	rows, total, _ := e.Lister(src, "", "nom ASC", 1, 10, ffLyon)
	if total != 2 || len(rows) != 2 {
		t.Fatalf("filtre fixe Lyon : total=%d len=%d (attendu 2)", total, len(rows))
	}
	// Combiné au filtre utilisateur (AND) : Lyon + "Alice" -> Alice seule.
	rows, total, _ = e.Lister(src, "Alice", "", 1, 10, ffLyon)
	if total != 1 || len(rows) != 1 || rows[0]["nom"] != "Alice" {
		t.Errorf("filtre fixe + recherche : total=%d rows=%v", total, rows)
	}
	// Le filtre fixe exclut bien les autres (Bob/Paris n'apparaît jamais).
	rows, _, _ = e.Lister(src, "Bob", "", 1, 10, ffLyon)
	if len(rows) != 0 {
		t.Errorf("Bob (Paris) ne doit pas passer le filtre fixe Lyon : %v", rows)
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	if err := e.InitialiserSource(src); err != nil {
		t.Fatal(err)
	}
	id, err := e.Creer(src, map[string]string{"nom": "Dora", "ville": "Nice", "note": "2"})
	if err != nil {
		t.Fatalf("creer: %v", err)
	}
	if id <= 0 {
		t.Fatalf("id attendu > 0, got %d", id)
	}
	cle := fmtInt(id)
	row, err := e.Consulter(src, cle)
	if err != nil || row == nil {
		t.Fatalf("consulter: %v row=%v", err, row)
	}
	if row["nom"] != "Dora" || row["ville"] != "Nice" {
		t.Errorf("consulter inattendu : %+v", row)
	}
	// cellString : la valeur TEXT (renvoyée en []byte par modernc) est propre.
	if row["ville"] != "Nice" {
		t.Errorf("cellString TEXT : %q", row["ville"])
	}
	ok, err := e.Modifier(src, cle, map[string]string{"ville": "Cannes"})
	if err != nil || !ok {
		t.Fatalf("modifier: %v ok=%v", err, ok)
	}
	row, _ = e.Consulter(src, cle)
	if row["ville"] != "Cannes" {
		t.Errorf("après modif : ville=%q", row["ville"])
	}
	ok, err = e.Supprimer(src, cle)
	if err != nil || !ok {
		t.Fatalf("supprimer: %v ok=%v", err, ok)
	}
	if row, _ := e.Consulter(src, cle); row != nil {
		t.Errorf("ligne toujours présente après suppression")
	}
}

func TestValider(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	cas := []struct {
		nom    string
		champs map[string]string
		veut   string // sous-chaîne attendue dans la 1re erreur ("" = aucune erreur)
	}{
		{"requis manquant", map[string]string{"ville": "X"}, "Nom requis"},
		{"trop long", map[string]string{"nom": "0123456789012345678901"}, "trop long"},
		{"entier invalide", map[string]string{"nom": "Z", "note": "abc"}, "nombre"},
		{"valide", map[string]string{"nom": "Z", "note": "7"}, ""},
	}
	for _, c := range cas {
		errs := e.Valider(src, c.champs, false)
		if c.veut == "" {
			if len(errs) != 0 {
				t.Errorf("%s : erreurs inattendues %v", c.nom, errs)
			}
			continue
		}
		if len(errs) == 0 || !contains(errs[0], c.veut) {
			t.Errorf("%s : attendu %q, got %v", c.nom, c.veut, errs)
		}
	}
}

func TestAutoMigration(t *testing.T) {
	e := testEngine(t)
	src := repertoire()
	if err := e.InitialiserSource(src); err != nil {
		t.Fatal(err)
	}
	// Ajoute une colonne et ré-initialise : ALTER TABLE l'ajoute sans perte.
	src.Colonnes["pays"] = content.ColonneDef{Type: "TEXT", Libelle: "Pays"}
	if err := e.InitialiserSource(src); err != nil {
		t.Fatalf("ré-init migration: %v", err)
	}
	rows, _, err := e.Lister(src, "", "nom ASC", 1, 10)
	if err != nil {
		t.Fatalf("lister après migration: %v", err)
	}
	if _, ok := rows[0]["pays"]; !ok {
		t.Errorf("colonne 'pays' absente après migration : %+v", rows[0])
	}
}

func TestSecuriteInjection(t *testing.T) {
	// Les gardes de content rejettent les identifiants dangereux.
	for _, mauvais := range []string{"a; DROP TABLE x", "a b", "`x`", "", "1col", "a-b"} {
		if content.ValiderNomSQL(mauvais) == nil {
			t.Errorf("ValiderNomSQL a accepté %q", mauvais)
		}
	}
	for _, mauvais := range []string{"TEXT; DROP", "VARCHAR(10)", "INT EGER"} {
		if content.ValiderTypeSQL(mauvais) == nil {
			t.Errorf("ValiderTypeSQL a accepté %q", mauvais)
		}
	}
	// Une source dont la table est malveillante est refusée par le moteur.
	e := testEngine(t)
	bad := content.SourceDonnees{Table: "x; DROP TABLE y", Colonnes: map[string]content.ColonneDef{"id": {Type: "INTEGER", ClePrimaire: true}}}
	if err := e.InitialiserSource(bad); err == nil {
		t.Error("InitialiserSource a accepté une table malveillante")
	}
}

func TestCellString(t *testing.T) {
	cas := map[string]struct {
		in   any
		want string
	}{
		"texte []byte": {[]byte("Nice"), "Nice"},
		"string":       {"abc", "abc"},
		"int64":        {int64(42), "42"},
		"float64":      {3.5, "3.5"},
		"nil":          {nil, ""},
		"bool":         {true, "1"},
	}
	for nom, c := range cas {
		if got := cellString(c.in); got != c.want {
			t.Errorf("%s : cellString=%q, attendu %q", nom, got, c.want)
		}
	}
}

// helpers de test
func fmtInt(i int64) string       { return strconv.FormatInt(i, 10) }
func contains(s, sub string) bool { return strings.Contains(s, sub) }
