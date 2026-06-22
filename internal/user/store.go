package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store persiste les comptes dans un fichier JSON, avec verrou (accès
// concurrents : 1 goroutine par session) et écriture atomique (fichier
// temporaire + rename), symétrique au choix retenu pour le contenu.
//
// L'horloge est injectable (now) pour des tests déterministes.
type Store struct {
	mu    sync.Mutex
	path  string
	users map[string]*User // clé = handle normalisé
	now   func() time.Time
}

// Open charge le store depuis path. Si le fichier n'existe pas, le store
// démarre vide (il sera créé à la première écriture). Un path vide donne un
// store purement en mémoire (utile pour les tests).
func Open(path string) (*Store, error) {
	s := &Store{
		path:  path,
		users: make(map[string]*User),
		now:   time.Now,
	}
	if path == "" {
		return s, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	var list []*User
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("JSON utilisateurs invalide (%s) : %w", path, err)
	}
	for _, u := range list {
		s.users[NormalizeHandle(u.Handle)] = u
	}
	return s, nil
}

// Count renvoie le nombre de comptes enregistrés.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.users)
}

// Get renvoie une copie du compte (insensible à la casse), ou false s'il
// n'existe pas. On renvoie une copie pour éviter toute mutation hors verrou.
func (s *Store) Get(handle string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[NormalizeHandle(handle)]
	if !ok {
		return User{}, false
	}
	return *u, true
}

// Register crée un compte. Erreur si le pseudo/mot de passe est invalide ou si
// le pseudo est déjà pris (insensible à la casse).
func (s *Store) Register(handle, password string) (User, error) {
	clean, err := ValidateHandle(handle)
	if err != nil {
		return User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return User{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := NormalizeHandle(clean)
	if _, exists := s.users[key]; exists {
		return User{}, fmt.Errorf("le pseudo %q est deja pris", clean)
	}
	u := &User{Handle: clean, PassHash: hash, Created: s.now()}
	s.users[key] = u
	if err := s.saveLocked(); err != nil {
		delete(s.users, key) // rollback mémoire si la persistance échoue
		return User{}, err
	}
	return *u, nil
}

// Authenticate vérifie le couple (pseudo, mot de passe). En cas de succès,
// incrémente le compteur d'appels, met à jour LastLogin, persiste, et renvoie
// le compte. En cas d'échec, renvoie une erreur générique (on ne révèle pas si
// c'est le pseudo ou le mot de passe qui est faux).
func (s *Store) Authenticate(handle, password string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[NormalizeHandle(handle)]
	if !ok || !VerifyPassword(u.PassHash, password) {
		return User{}, fmt.Errorf("pseudo ou mot de passe incorrect")
	}
	u.Calls++
	u.LastLogin = s.now()
	if err := s.saveLocked(); err != nil {
		return User{}, err
	}
	return *u, nil
}

// saveLocked écrit le store sur disque de façon atomique. À appeler avec le
// verrou détenu. No-op si le store est en mémoire (path vide).
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	// Ordonner par pseudo normalisé pour une sortie stable (diffs lisibles).
	list := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	sort.Slice(list, func(i, j int) bool {
		return NormalizeHandle(list[i].Handle) < NormalizeHandle(list[j].Handle)
	})
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation utilisateurs : %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".users-*.json.tmp")
	if err != nil {
		return fmt.Errorf("fichier temporaire : %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op après rename réussi

	if _, err := tmp.Write(b); err != nil {
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
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename atomique : %w", err)
	}
	return nil
}
