// Package files gère la bibliothèque de fichiers du BBS : un répertoire sur disque
// (la « mémoire de masse » côté serveur) d'où les utilisateurs téléchargent et
// vers lequel ils téléversent. Les noms sont validés pour empêcher toute sortie
// du répertoire (path traversal).
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Library est une bibliothèque de fichiers adossée à un répertoire.
type Library struct {
	dir     string
	maxSize int64 // taille max d'un téléversement (0 = pas de limite)
}

// Info décrit un fichier de la bibliothèque.
type Info struct {
	Name string
	Size int64
}

// Open ouvre (et crée si besoin) la bibliothèque dans dir. maxSize borne la
// taille d'un téléversement (0 = illimité).
func Open(dir string, maxSize int64) (*Library, error) {
	if dir == "" {
		return nil, fmt.Errorf("répertoire de fichiers vide")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("création %q : %w", dir, err)
	}
	return &Library{dir: dir, maxSize: maxSize}, nil
}

// validName n'autorise qu'un nom de fichier simple (pas de séparateur ni « .. »),
// non vide, longueur raisonnable.
func validName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	if name != filepath.Base(name) || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	return true
}

// List renvoie les fichiers réguliers de la bibliothèque, triés par nom.
func (l *Library) List() ([]Info, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Info{Name: e.Name(), Size: fi.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Read renvoie le contenu d'un fichier de la bibliothèque.
func (l *Library) Read(name string) ([]byte, error) {
	if !validName(name) {
		return nil, fmt.Errorf("nom de fichier invalide : %q", name)
	}
	return os.ReadFile(filepath.Join(l.dir, name))
}

// Write enregistre un fichier dans la bibliothèque (écriture atomique). Refuse un
// nom invalide ou un contenu dépassant la taille max.
func (l *Library) Write(name string, data []byte) error {
	if !validName(name) {
		return fmt.Errorf("nom de fichier invalide : %q", name)
	}
	if l.maxSize > 0 && int64(len(data)) > l.maxSize {
		return fmt.Errorf("fichier trop volumineux (%d > %d octets)", len(data), l.maxSize)
	}
	dst := filepath.Join(l.dir, name)
	tmp, err := os.CreateTemp(l.dir, ".upload-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op si le rename a réussi
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}
