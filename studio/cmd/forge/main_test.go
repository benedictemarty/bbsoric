package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/studio/internal/store"
)

const validSite = `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"Q","label":"Quitter","target":"__quit__"}]}}}`

func newServer(t *testing.T) (*server, string) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "site.json"), []byte(validSite), 0o644)
	return &server{store: store.New(dir), log: slog.New(slog.NewTextHandler(io.Discard, nil))}, dir
}

func TestHandleSites(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleSites(rec, httptest.NewRequest("GET", "/api/sites", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "site.json") {
		t.Fatalf("handleSites: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSite(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleSite(rec, httptest.NewRequest("GET", "/api/site?name=site.json", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"start":"main"`) {
		t.Fatalf("handleSite: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidate(t *testing.T) {
	s, _ := newServer(t)
	// valide
	rec := httptest.NewRecorder()
	s.handleValidate(rec, httptest.NewRequest("POST", "/api/validate", strings.NewReader(validSite)))
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("site valide: %s", rec.Body.String())
	}
	// invalide
	rec = httptest.NewRecorder()
	bad := `{"start":"x","pages":{}}`
	s.handleValidate(rec, httptest.NewRequest("POST", "/api/validate", strings.NewReader(bad)))
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Errorf("site invalide non détecté: %s", rec.Body.String())
	}
}

func TestHandleSaveThenReload(t *testing.T) {
	s, dir := newServer(t)
	rec := httptest.NewRecorder()
	s.handleSave(rec, httptest.NewRequest("POST", "/api/save?name=new.json", strings.NewReader(validSite)))
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("save: %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "new.json")); err != nil {
		t.Errorf("le fichier doit être écrit: %v", err)
	}
	// save d'un JSON invalide -> ok:false
	rec = httptest.NewRecorder()
	s.handleSave(rec, httptest.NewRequest("POST", "/api/save?name=bad.json", strings.NewReader(`{"pages":{}}`)))
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Errorf("save invalide doit échouer: %s", rec.Body.String())
	}
}

// TestMutatingEndpointsRequirePOST : les endpoints mutants refusent une méthode
// autre que POST avec 405 + en-tête Allow (S11.8).
func TestMutatingEndpointsRequirePOST(t *testing.T) {
	s, _ := newServer(t)
	cases := []struct {
		name string
		h    http.HandlerFunc
		path string
	}{
		{"validate", s.handleValidate, "/api/validate"},
		{"save", s.handleSave, "/api/save?name=x.json"},
		{"screen", s.handleScreen, "/api/screen?page=main"},
		{"deploy", s.handleDeploy, "/api/deploy?site=site.json&profile=dev"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		c.h(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s en GET: code %d, attendu 405", c.name, rec.Code)
		}
		if got := rec.Header().Get("Allow"); got != "POST" {
			t.Errorf("%s: en-tête Allow = %q, attendu POST", c.name, got)
		}
	}
}

// TestHandleSaveInvalidReturns400 : une sauvegarde de contenu invalide renvoie 400
// (et non plus 200), tout en portant le détail dans le corps (S11.8).
func TestHandleSaveInvalidReturns400(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleSave(rec, httptest.NewRequest("POST", "/api/save?name=bad.json", strings.NewReader(`{"pages":{}}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("save invalide: code %d, attendu 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Errorf("le corps doit porter l'erreur: %s", rec.Body.String())
	}
}

func TestHandleScreen(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleScreen(rec, httptest.NewRequest("POST", "/api/screen?page=main", strings.NewReader(validSite)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Quitter") {
		t.Fatalf("screen: %d %s", rec.Code, rec.Body.String())
	}
}
