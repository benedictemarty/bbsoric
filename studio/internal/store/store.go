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

	"github.com/benedictemarty/bbsoric/internal/content"
)

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
	data = pretty.Bytes()

	tmp, err := os.CreateTemp(s.dir, ".site-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
