// Package store gère les fichiers de contenu (site*.json) manipulés par le
// studio : lister, charger, et enregistrer après validation par le MÊME paquet
// que le serveur (internal/content) — aucune divergence de format.
package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/benedictemarty/bbsoric/internal/atomicfile"
	"github.com/benedictemarty/bbsoric/internal/content"
)

// backupsDir est le sous-répertoire (du répertoire de contenu) où sont rangées
// les sauvegardes horodatées, hors de la liste des sites éditables.
const backupsDir = "backups"

// Store expose les sites JSON d'un répertoire de contenu.
type Store struct {
	dir string
}

// New crée un Store sur le répertoire donné.
func New(dir string) *Store { return &Store{dir: dir} }

// Dir renvoie le répertoire de contenu.
func (s *Store) Dir() string { return s.dir }

// List renvoie les noms (base) des fichiers .json du répertoire, triés.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// safePath valide un nom de fichier (pas de traversée de répertoire) et renvoie
// son chemin absolu dans le répertoire du store.
func (s *Store) safePath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("nom de fichier invalide : %q", name)
	}
	if !strings.HasSuffix(name, ".json") {
		return "", fmt.Errorf("le nom doit finir par .json : %q", name)
	}
	return filepath.Join(s.dir, name), nil
}

// Load renvoie le contenu brut d'un site.
func (s *Store) Load(name string) ([]byte, error) {
	path, err := s.safePath(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Save valide le JSON (via content.Parse, la même validation que le serveur)
// puis l'écrit de façon atomique (fichier temporaire + rename).
func (s *Store) Save(name string, data []byte) error {
	path, err := s.safePath(name)
	if err != nil {
		return err
	}
	if _, err := content.Parse(data); err != nil {
		return fmt.Errorf("contenu invalide : %w", err)
	}
	// Ré-indente pour un fichier lisible et des diffs git stables (préserve
	// toutes les clés, y compris _comment).
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		return fmt.Errorf("formatage : %w", err)
	}
	pretty.WriteByte('\n')
	return atomicfile.Write(path, pretty.Bytes())
}

// Create crée un NOUVEAU site (échoue s'il existe déjà). data est validé et
// écrit comme par Save. Sert au bouton « Nouveau site » du studio (F3).
func (s *Store) Create(name string, data []byte) error {
	path, err := s.safePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("le site %q existe déjà", name)
	}
	return s.Save(name, data)
}

// Backup copie l'état courant d'un site vers une sauvegarde horodatée dans le
// sous-répertoire backups/. Renvoie le nom de fichier de la sauvegarde. now est
// injectable pour des tests déterministes.
func (s *Store) Backup(name string, now time.Time) (string, error) {
	data, err := s.Load(name) // valide aussi le nom (safePath)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s.dir, backupsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("répertoire de sauvegardes : %w", err)
	}
	stamp := now.Format("20060102-150405")
	base := strings.TrimSuffix(name, ".json") + "." + stamp + ".json"
	if err := atomicfile.Write(filepath.Join(dir, base), data); err != nil {
		return "", err
	}
	return base, nil
}

// Backups liste les sauvegardes d'un site (plus récentes en tête), par nom de
// fichier dans backups/ commençant par « <site>. ».
func (s *Store) Backups(name string) ([]string, error) {
	if _, err := s.safePath(name); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(s.dir, backupsDir))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	prefix := strings.TrimSuffix(name, ".json") + "."
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out))) // horodatage décroissant
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// Restore restaure une sauvegarde sur le site. Le contenu est validé et une
// sauvegarde de l'état courant est prise avant d'écraser (sécurité). backup est
// un nom de fichier renvoyé par Backups.
func (s *Store) Restore(name, backup string, now time.Time) error {
	if _, err := s.safePath(name); err != nil {
		return err
	}
	if backup == "" || strings.ContainsAny(backup, `/\`) || strings.Contains(backup, "..") ||
		!strings.HasSuffix(backup, ".json") {
		return fmt.Errorf("nom de sauvegarde invalide : %q", backup)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, backupsDir, backup))
	if err != nil {
		return err
	}
	// Sauvegarde de sécurité de l'état courant (ignore l'absence du site courant).
	if _, err := os.Stat(filepath.Join(s.dir, name)); err == nil {
		if _, err := s.Backup(name, now); err != nil {
			return err
		}
	}
	return s.Save(name, data) // valide + écrit atomiquement
}
