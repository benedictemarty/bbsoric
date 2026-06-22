// Commande forge : le studio web du BBS Oric.
//
// Outil de DÉVELOPPEMENT LOCAL : édite le(s) site*.json (pages menu/page/applet),
// aperçu OASCII couleur, validation par le même paquet que le serveur
// (internal/content). Bind 127.0.0.1 par défaut (non exposé, pas d'auth).
//
//	forge -addr 127.0.0.1:8080 -content-dir content
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/render"
	"github.com/benedictemarty/bbsoric/studio/internal/deploy"
	"github.com/benedictemarty/bbsoric/studio/internal/store"
	"github.com/benedictemarty/bbsoric/studio/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "adresse d'écoute (locale)")
	dir := flag.String("content-dir", "content", "répertoire des site*.json")
	profilesDir := flag.String("profiles-dir", "deploy/profiles", "répertoire des profils de déploiement")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	srv := &server{store: store.New(*dir), profilesDir: *profilesDir, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/sites", srv.handleSites)
	mux.HandleFunc("/api/site", srv.handleSite)
	mux.HandleFunc("/api/validate", srv.handleValidate)
	mux.HandleFunc("/api/save", srv.handleSave)
	mux.HandleFunc("/api/screen", srv.handleScreen)
	mux.HandleFunc("/api/profiles", srv.handleProfiles)
	mux.HandleFunc("/api/profile", srv.handleProfile)
	mux.HandleFunc("/api/deploy", srv.handleDeploy)
	mux.Handle("/", http.FileServer(http.FS(web.FS)))

	log.Info("studio forge en écoute", "addr", *addr, "content-dir", *dir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

type server struct {
	store       *store.Store
	profilesDir string
	log         *slog.Logger
}

// writeJSON sérialise v en JSON avec le code de statut donné.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// handleSites : GET /api/sites -> liste des fichiers .json.
func (s *server) handleSites(w http.ResponseWriter, r *http.Request) {
	names, err := s.store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, http.StatusOK, names)
}

// handleSite : GET /api/site?name= -> contenu brut du site.
func (s *server) handleSite(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Load(r.URL.Query().Get("name"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

// handleValidate : POST /api/validate (corps = site JSON) -> {ok, error}.
func (s *server) handleValidate(w http.ResponseWriter, r *http.Request) {
	body, _ := readBody(r)
	if _, err := content.Parse(body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSave : POST /api/save?name= (corps = site JSON) -> {ok, error}.
// Valide avant d'écrire (refuse un contenu invalide).
func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	body, _ := readBody(r)
	if err := s.store.Save(r.URL.Query().Get("name"), body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleScreen : POST /api/screen?page= (corps = site JSON) -> flux d'octets
// OASCII de l'écran de la page (le MÊME que celui servi par le serveur, via
// internal/render). Lecture tolérante (pas de validation des cibles) pour
// prévisualiser en cours d'édition. Le client (simulateur ULA) rend ces octets.
func (s *server) handleScreen(w http.ResponseWriter, r *http.Request) {
	body, _ := readBody(r)
	var site content.Site
	if err := json.Unmarshal(body, &site); err != nil {
		http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
		return
	}
	p := site.Pages[r.URL.Query().Get("page")]
	if p == nil {
		http.Error(w, "page introuvable", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(render.Screen(p))
}

// handleProfiles : GET /api/profiles?site= -> profils (dev/int/prod) DU SITE.
func (s *server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	profs, err := deploy.LoadSiteProfiles(s.profilesDir, r.URL.Query().Get("site"))
	if err != nil {
		writeJSON(w, http.StatusOK, []string{}) // site invalide / pas de profils
		return
	}
	writeJSON(w, http.StatusOK, deploy.Names(profs))
}

// handleProfile : lit (GET) ou enregistre (POST) un profil d'un site.
//
//	GET  /api/profile?site=&env=  -> champs du profil (valeurs par défaut si absent)
//	POST /api/profile?site=&env=  (corps = Profile JSON) -> écrit <site>/<env>.conf
func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	env := r.URL.Query().Get("profile")
	if env == "" {
		env = r.URL.Query().Get("env")
	}
	if r.Method == http.MethodPost {
		body, _ := readBody(r)
		var p deploy.Profile
		if err := json.Unmarshal(body, &p); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "JSON invalide : " + err.Error()})
			return
		}
		if err := deploy.SaveProfile(s.profilesDir, site, env, &p); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	// GET : renvoie le profil existant, ou des valeurs par défaut.
	profs, _ := deploy.LoadSiteProfiles(s.profilesDir, site)
	if p := profs[env]; p != nil {
		writeJSON(w, http.StatusOK, p)
		return
	}
	writeJSON(w, http.StatusOK, deploy.Profile{Name: env, Port: "22", Reload: "none"})
}

// handleDeploy : POST /api/deploy?site=&profile=&dryRun= (corps = site JSON).
// Le profil est résolu DANS le site (dev/int/prod propres à ce site).
// dryRun par défaut TRUE ; passer dryRun=false pour exécuter réellement.
func (s *server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	body, _ := readBody(r)
	site := r.URL.Query().Get("site")
	env := r.URL.Query().Get("profile")
	profs, err := deploy.LoadSiteProfiles(s.profilesDir, site)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "log": []string{"profils illisibles : " + err.Error()}})
		return
	}
	dryRun := r.URL.Query().Get("dryRun") != "false"
	stamp := time.Now().Format("20060102-150405")
	res, err := deploy.Deploy(profs[env], body, dryRun, stamp)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "log": []string{err.Error()}})
		return
	}
	res.OK = true
	s.log.Info("déploiement", "site", site, "profil", env, "dryRun", dryRun)
	writeJSON(w, http.StatusOK, res)
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	const max = 1 << 20 // 1 Mio
	return io.ReadAll(io.LimitReader(r.Body, max))
}
