// Package news persiste les actualités / annonces du BBS : de courtes entrées
// datées, publiées par le sysop et lues par tous. Même patron de store que le
// mur/forum/messagerie (server/internal/{wall,forum,pm}) : verrou, écriture
// atomique (internal/atomicfile), horloge injectable, entrées bornées et
// nettoyées en ASCII imprimable.
//
// La règle d'écriture (réservée à l'admin) est appliquée par l'applet, pas par
// le store : le store se contente de persister ce qu'on lui donne.
package news

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/internal/atomicfile"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Bornes (serveur public : tout est plafonné).
const (
	MaxTitle = 38  // titre d'une annonce (tient sur une ligne de 40 colonnes)
	MaxBody  = 400 // corps (s'affiche sur plusieurs lignes)
	MinBody  = 1   // corps non vide après nettoyage
	MaxItems = 200 // annonces conservées (la plus ancienne est évincée)
)

// Item est une annonce datée.
type Item struct {
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	Author string    `json:"author"`
	At     time.Time `json:"at"`
}

// Store persiste les annonces dans un fichier JSON.
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
	now   func() time.Time
}

// Open charge le store depuis path. Fichier absent → vide. path vide → mémoire.
func Open(path string) (*Store, error) {
	s := &Store{path: path, now: time.Now}
	if path == "" {
		return s, nil
	}
	if _, err := atomicfile.ReadJSON(path, &s.items); err != nil {
		return nil, err
	}
	if len(s.items) > MaxItems {
		s.items = s.items[len(s.items)-MaxItems:]
	}
	return s, nil
}

// sanitize nettoie puis borne un texte à max caractères.
func sanitize(text string, max int) string {
	out := oascii.SanitizeText(text)
	if len(out) > max {
		out = strings.TrimSpace(out[:max])
	}
	return out
}

// Post ajoute une annonce. Titre et corps sont nettoyés et bornés ; le corps
// doit être non vide (un titre vide est toléré et remplacé par « Annonce »).
func (s *Store) Post(author, title, body string) (Item, error) {
	b := sanitize(body, MaxBody)
	if len(b) < MinBody {
		return Item{}, fmt.Errorf("annonce vide")
	}
	t := sanitize(title, MaxTitle)
	if t == "" {
		t = "Annonce"
	}
	a := sanitize(author, 16)
	if a == "" {
		a = "sysop"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	it := Item{Title: t, Body: b, Author: a, At: s.now()}
	s.items = append(s.items, it)
	if len(s.items) > MaxItems {
		s.items = s.items[len(s.items)-MaxItems:]
	}
	if err := s.saveLocked(); err != nil {
		s.items = s.items[:len(s.items)-1] // rollback mémoire
		return Item{}, err
	}
	return it, nil
}

// List renvoie jusqu'à n annonces, de la plus récente à la plus ancienne (copie).
// n<=0 renvoie tout.
func (s *Store) List(n int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := len(s.items)
	if n <= 0 || n > total {
		n = total
	}
	out := make([]Item, n)
	for i := 0; i < n; i++ {
		out[i] = s.items[total-1-i] // antéchronologique
	}
	return out
}

// Count renvoie le nombre d'annonces conservées.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

// saveLocked écrit le store de façon atomique (verrou détenu). No-op en mémoire.
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	return atomicfile.WriteJSON(s.path, s.items)
}
