package main

import (
	"net/http"
	"time"
)

// handleCreate : POST /api/site/create?name= (corps = site JSON) -> {ok, error}.
// Crée un NOUVEAU site (échoue s'il existe). Bouton « Nouveau site » (F3).
func (s *server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	body, ok := bodyOrError(w, r)
	if !ok {
		return
	}
	if err := s.store.Create(r.URL.Query().Get("name"), body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleBackup : POST /api/backup?name= -> {ok, backup}. Sauvegarde horodatée
// de l'état courant du site.
func (s *server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	name, err := s.store.Backup(r.URL.Query().Get("name"), time.Now())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "backup": name})
}

// handleBackups : GET /api/backups?name= -> liste des sauvegardes (récentes en tête).
func (s *server) handleBackups(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.Backups(r.URL.Query().Get("name"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleRestore : POST /api/restore?name=&backup= -> {ok, error}. Restaure une
// sauvegarde (après avoir sauvegardé l'état courant).
func (s *server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if !requirePOST(w, r) {
		return
	}
	q := r.URL.Query()
	if err := s.store.Restore(q.Get("name"), q.Get("backup"), time.Now()); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
