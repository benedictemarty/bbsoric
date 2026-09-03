// Package userfiles gère les espaces de fichiers PERSONNELS du BBS : un
// répertoire privé PAR utilisateur identifié, distinct de la bibliothèque
// publique (`internal/files`) qui sert le catalogue. Chaque compte a son
// sous-répertoire `<root>/<pseudo-normalisé>/`, avec un quota (nombre de fichiers
// + octets) pour protéger le disque d'un serveur public. Cf. ADR-0006.
package userfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/benedictemarty/bbsoric/server/internal/files"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// safeKey borne défensivement le nom de répertoire dérivé d'un pseudo. Les pseudos
// sont déjà validés à l'inscription (ASCII [A-Za-z0-9_-], 2..16), et NormalizeHandle
// les met en minuscules ; ce garde-fou refuse malgré tout tout ce qui pourrait sortir
// du répertoire racine (défense en profondeur : aucun '/', '.', '..').
var safeKey = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// Store est la racine des espaces personnels, avec les quotas appliqués.
type Store struct {
	root      string
	maxFiles  int   // nombre max de fichiers par utilisateur (0 = illimité)
	maxBytes  int64 // taille totale max par utilisateur en octets (0 = illimité)
	maxUpload int64 // taille max d'un fichier (transmise à chaque bibliothèque)
}

// Open ouvre (et crée si besoin) la racine des espaces personnels. maxFiles et
// maxBytes fixent le quota par utilisateur (0 = illimité) ; maxUpload borne la
// taille d'un fichier (comme la bibliothèque publique).
func Open(root string, maxFiles int, maxBytes, maxUpload int64) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("répertoire des fichiers personnels vide")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("création %q : %w", root, err)
	}
	return &Store{root: root, maxFiles: maxFiles, maxBytes: maxBytes, maxUpload: maxUpload}, nil
}

// For renvoie la bibliothèque privée de l'utilisateur handle (créée si besoin),
// rootée sur `<root>/<pseudo-normalisé>/`. Réutilise `files.Library` (validation
// des noms, écriture atomique, lecture bornée). Erreur si le pseudo est invalide.
func (s *Store) For(handle string) (*files.Library, error) {
	key := user.NormalizeHandle(handle)
	if !safeKey.MatchString(key) {
		return nil, fmt.Errorf("pseudo invalide pour un espace personnel : %q", handle)
	}
	return files.Open(filepath.Join(s.root, key), s.maxUpload)
}

// Usage renvoie le nombre de fichiers et le total d'octets de l'espace de handle.
func (s *Store) Usage(handle string) (int, int64, error) {
	lib, err := s.For(handle)
	if err != nil {
		return 0, 0, err
	}
	list, err := lib.List()
	if err != nil {
		return 0, 0, err
	}
	var total int64
	for _, f := range list {
		total += f.Size
	}
	return len(list), total, nil
}

// Write enregistre data sous name dans l'espace de handle, en respectant le QUOTA
// (nombre de fichiers + octets). Un fichier de même nom est remplacé (compté une
// seule fois, sa taille étant déduite). Renvoie une erreur explicite — sans écrire —
// si le quota serait dépassé ; la borne par fichier reste gérée par la bibliothèque.
func (s *Store) Write(handle, name string, data []byte) error {
	lib, err := s.For(handle)
	if err != nil {
		return err
	}
	list, err := lib.List()
	if err != nil {
		return err
	}
	var total int64
	existing := int64(-1) // taille d'un fichier de même nom déjà présent (-1 = aucun)
	for _, f := range list {
		total += f.Size
		if f.Name == name {
			existing = f.Size
		}
	}
	newCount := len(list)
	if existing < 0 {
		newCount++ // nouveau fichier
	}
	if s.maxFiles > 0 && newCount > s.maxFiles {
		return fmt.Errorf("quota atteint : %d fichiers maximum", s.maxFiles)
	}
	newTotal := total + int64(len(data))
	if existing >= 0 {
		newTotal -= existing // remplacement : on retire l'ancienne taille
	}
	if s.maxBytes > 0 && newTotal > s.maxBytes {
		return fmt.Errorf("quota atteint : %d octets maximum (deja %do)", s.maxBytes, total)
	}
	return lib.Write(name, data)
}

// Delete supprime un fichier de l'espace de handle.
func (s *Store) Delete(handle, name string) error {
	lib, err := s.For(handle)
	if err != nil {
		return err
	}
	return lib.Delete(name)
}

// Rename renomme un fichier de l'espace de handle (oldName -> newName). Ne change
// pas l'usage du quota ; refuse si la cible existe déjà.
func (s *Store) Rename(handle, oldName, newName string) error {
	lib, err := s.For(handle)
	if err != nil {
		return err
	}
	return lib.Rename(oldName, newName)
}
