package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const gridSite = `{
  "start": "g",
  "sources_donnees": {
    "t": {
      "table": "t", "tri_defaut": "nom ASC",
      "colonnes": {
        "id":    {"type":"INTEGER","cle_primaire":true,"auto_increment":true},
        "nom":   {"type":"TEXT","libelle":"Nom","longueur_max":16},
        "ville": {"type":"TEXT","libelle":"Ville","longueur_max":10}
      },
      "donnees": [ {"nom":"Alice","ville":"Lyon"}, {"nom":"Bob","ville":"Paris"}, {"nom":"Carla","ville":"Nice"} ]
    }
  },
  "pages": { "g": {"title":"G","datawindow":{"source":"t","colonnes_affichees":["nom","ville"],"largeurs":[16,10]}} }
}`

// bufText masque le bit 7 (vidéo inverse) et ne garde que l'ASCII imprimable.
func bufText(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		c &= 0x7F
		if c >= 0x20 && c < 0x7F {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

func TestHandleGrid(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleGrid(rec, httptest.NewRequest("POST", "/api/grid?page=g&n=1&sel=0", strings.NewReader(gridSite)))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d, attendu 200 ; corps : %s", rec.Code, rec.Body.String())
	}
	if n := rec.Body.Len(); n != 40*28 {
		t.Fatalf("buffer de %d octets, attendu %d (40x28)", n, 40*28)
	}
	txt := bufText(rec.Body.Bytes())
	for _, want := range []string{"Nom", "Ville", "Alice", "Lyon", "3 enreg."} {
		if !strings.Contains(txt, want) {
			t.Errorf("grille ne contient pas %q ; vu : %q", want, txt)
		}
	}
}

func TestHandleGridFilter(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	// filtre "Paris" -> seule Bob (Paris) reste (1 enreg.), Alice/Lyon absente.
	s.handleGrid(rec, httptest.NewRequest("POST", "/api/grid?page=g&n=1&filtre=Paris", strings.NewReader(gridSite)))
	txt := bufText(rec.Body.Bytes())
	if !strings.Contains(txt, "1 enreg.") || !strings.Contains(txt, "Bob") {
		t.Errorf("filtre Paris : attendu 1 enreg. + Bob ; vu : %q", txt)
	}
	if strings.Contains(txt, "Alice") {
		t.Errorf("filtre Paris : Alice ne devrait pas apparaître ; vu : %q", txt)
	}
}

func TestHandleGridRequiresPost(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleGrid(rec, httptest.NewRequest("GET", "/api/grid?page=g", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET sur /api/grid : code %d, attendu 405", rec.Code)
	}
}
