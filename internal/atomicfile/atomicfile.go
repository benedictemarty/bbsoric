// Package atomicfile fournit l'écriture atomique d'un fichier : on écrit dans
// un fichier temporaire du même répertoire, on le synchronise sur disque, puis
// on le renomme (rename atomique sur le même système de fichiers). Un lecteur
// concurrent voit soit l'ancien contenu, soit le nouveau, jamais un fichier
// partiel.
//
// C'est la primitive de persistance partagée par les stores JSON du serveur
// (comptes, mur, forum, messagerie) : une seule source de vérité pour ce
// motif, au lieu de le recopier dans chaque store.
package atomicfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write écrit data dans path de façon atomique (temp + sync + rename).
func Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".atomic-*.tmp")
	if err != nil {
		return fmt.Errorf("fichier temporaire : %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op après un rename réussi

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("ecriture temporaire : %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporaire : %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fermeture temporaire : %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename atomique : %w", err)
	}
	return nil
}

// WriteJSON sérialise v en JSON indenté (2 espaces) et l'écrit atomiquement.
func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation JSON : %w", err)
	}
	return Write(path, b)
}

// ReadJSON lit path et le désérialise dans v. Renvoie (false, nil) si le fichier
// n'existe pas (cas d'un store neuf : v reste inchangé), (true, nil) en cas de
// succès, (true, err) si la lecture ou le JSON échoue. Symétrique de WriteJSON.
func ReadJSON(path string, v any) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, fmt.Errorf("lecture %s : %w", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return true, fmt.Errorf("JSON invalide (%s) : %w", path, err)
	}
	return true, nil
}
