package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleCreateBackupRestore : cycle F3 via les handlers HTTP — créer un
// site, le sauvegarder, lister les sauvegardes, restaurer.
func TestHandleCreateBackupRestore(t *testing.T) {
	s, _ := newServer(t)

	// Create.
	rec := httptest.NewRecorder()
	s.handleCreate(rec, httptest.NewRequest("POST", "/api/site/create?name=neuf.json", strings.NewReader(validSite)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("handleCreate: %d %s", rec.Code, rec.Body.String())
	}

	// Backup.
	rec = httptest.NewRecorder()
	s.handleBackup(rec, httptest.NewRequest("POST", "/api/backup?name=neuf.json", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"backup":"neuf.`) {
		t.Fatalf("handleBackup: %d %s", rec.Code, rec.Body.String())
	}

	// Backups (liste non vide).
	rec = httptest.NewRecorder()
	s.handleBackups(rec, httptest.NewRequest("GET", "/api/backups?name=neuf.json", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "neuf.") {
		t.Fatalf("handleBackups: %d %s", rec.Code, rec.Body.String())
	}
	// Récupère un nom de sauvegarde du corps JSON (["neuf.<stamp>.json"]).
	body := rec.Body.String()
	i, j := strings.Index(body, `"`), strings.LastIndex(body, `"`)
	if i < 0 || j <= i {
		t.Fatalf("liste de sauvegardes inattendue : %s", body)
	}
	bak := body[i+1 : j]

	// Restore.
	rec = httptest.NewRecorder()
	s.handleRestore(rec, httptest.NewRequest("POST", "/api/restore?name=neuf.json&backup="+bak, nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("handleRestore: %d %s", rec.Code, rec.Body.String())
	}
}

// TestHandleCreateRejectsDuplicate : créer un site déjà existant échoue (400).
func TestHandleCreateRejectsDuplicate(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleCreate(rec, httptest.NewRequest("POST", "/api/site/create?name=site.json", strings.NewReader(validSite)))
	if rec.Code != 400 {
		t.Fatalf("créer un site existant devrait renvoyer 400, a renvoyé %d", rec.Code)
	}
}

// TestHandleBackupRequiresPost : les endpoints mutants exigent POST.
func TestHandleBackupRequiresPost(t *testing.T) {
	s, _ := newServer(t)
	rec := httptest.NewRecorder()
	s.handleBackup(rec, httptest.NewRequest("GET", "/api/backup?name=site.json", nil))
	if rec.Code != 405 {
		t.Fatalf("GET sur /api/backup devrait renvoyer 405, a renvoyé %d", rec.Code)
	}
}
