package datawindow

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/content"
)

func apiSource(url, racine string) content.SourceDonnees {
	return content.SourceDonnees{
		TypeSource: "api",
		TriDefaut:  "nom ASC",
		API:        &content.APIConfig{URL: url, Racine: racine, TTL: 60},
		Colonnes: map[string]content.ColonneDef{
			"id":   {Type: "INTEGER", Libelle: "ID", ClePrimaire: true},
			"nom":  {Type: "TEXT", Libelle: "Nom"},
			"note": {Type: "INTEGER", Libelle: "Note"},
		},
	}
}

func TestAPISourceListeFiltreTri(t *testing.T) {
	const body = `[
		{"id":1,"nom":"Charlie","note":4},
		{"id":2,"nom":"Alice","note":5},
		{"id":3,"nom":"Bob","note":3}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	e := testEngine(t)
	src := apiSource(srv.URL, "")

	// Tri par défaut nom ASC -> Alice, Bob, Charlie.
	rows, total, err := e.Lister(src, "", "", 1, 10)
	if err != nil {
		t.Fatalf("lister api: %v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Fatalf("total=%d len=%d (attendu 3)", total, len(rows))
	}
	if rows[0]["nom"] != "Alice" || rows[2]["nom"] != "Charlie" {
		t.Errorf("tri ASC inattendu : %v", []string{rows[0]["nom"], rows[1]["nom"], rows[2]["nom"]})
	}
	// id rendu en entier propre (pas 1.0) malgré le décodage JSON en float64.
	if rows[0]["id"] != "2" {
		t.Errorf("id Alice = %q, attendu 2", rows[0]["id"])
	}
	// Tri numérique DESC sur note -> 5,4,3.
	rows, _, _ = e.Lister(src, "", "note DESC", 1, 10)
	if rows[0]["note"] != "5" || rows[2]["note"] != "3" {
		t.Errorf("tri note DESC inattendu : %v", []string{rows[0]["note"], rows[1]["note"], rows[2]["note"]})
	}
	// Filtre sous-chaîne.
	rows, total, _ = e.Lister(src, "ali", "", 1, 10)
	if total != 1 || rows[0]["nom"] != "Alice" {
		t.Errorf("filtre 'ali' inattendu : total=%d %v", total, rows)
	}
	// Pagination.
	p1, total, _ := e.Lister(src, "", "nom ASC", 1, 2)
	p2, _, _ := e.Lister(src, "", "nom ASC", 2, 2)
	if total != 3 || len(p1) != 2 || len(p2) != 1 {
		t.Errorf("pagination api : total=%d p1=%d p2=%d", total, len(p1), len(p2))
	}
}

func TestAPISourceRacine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"results":[{"id":1,"nom":"X","note":1}]}`)
	}))
	defer srv.Close()
	e := testEngine(t)
	src := apiSource(srv.URL, "results")
	rows, total, err := e.Lister(src, "", "", 1, 10)
	if err != nil || total != 1 || rows[0]["nom"] != "X" {
		t.Fatalf("racine: err=%v total=%d rows=%v", err, total, rows)
	}
}

func TestAPISourceCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, `[{"id":1,"nom":"X"}]`)
	}))
	defer srv.Close()
	e := testEngine(t)
	src := apiSource(srv.URL, "")
	src.API.TTL = 60
	for i := 0; i < 3; i++ {
		if _, _, err := e.Lister(src, "", "", 1, 10); err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("cache TTL : %d requêtes HTTP, attendu 1", got)
	}
	// Avance l'horloge au-delà du TTL -> nouvelle requête.
	e.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, _, err := e.Lister(src, "", "", 1, 10); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("après TTL : %d requêtes, attendu 2", got)
	}
}

func TestAPISourceConsulterEtLectureSeule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"id":7,"nom":"Sept"}]`)
	}))
	defer srv.Close()
	e := testEngine(t)
	src := apiSource(srv.URL, "")

	row, err := e.Consulter(src, "7")
	if err != nil || row == nil || row["nom"] != "Sept" {
		t.Fatalf("consulter api: err=%v row=%v", err, row)
	}
	// Écritures refusées sur une source API.
	if _, err := e.Creer(src, map[string]string{"nom": "X"}); !errors.Is(err, errSourceLectureSeule) {
		t.Errorf("Creer api devrait être refusé : %v", err)
	}
	if _, err := e.Supprimer(src, "7"); !errors.Is(err, errSourceLectureSeule) {
		t.Errorf("Supprimer api devrait être refusé : %v", err)
	}
}

func TestAPISourceErreurHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	e := testEngine(t)
	if _, _, err := e.Lister(apiSource(srv.URL, ""), "", "", 1, 10); err == nil {
		t.Error("statut 500 devrait remonter une erreur")
	}
}

// InitialiserSource ignore les sources API (rien à créer en base).
func TestAPISourceInitNoOp(t *testing.T) {
	e := testEngine(t)
	if err := e.InitialiserSource(apiSource("http://x", "")); err != nil {
		t.Errorf("init source API : %v", err)
	}
}
