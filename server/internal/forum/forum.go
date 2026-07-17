// Package forum persiste les fils de discussion du BBS : des sujets (threads)
// contenant des messages, lisibles et postables par les appelants.
//
// C'est le second applet à écriture persistée après le mur (server/internal/wall)
// et il en reprend le patron : store JSON à verrou, écriture atomique
// (temp+rename), horloge injectable, entrées bornées et nettoyées en ASCII
// imprimable (oascii.SanitizeText). La différence est la structure à deux
// niveaux (fil → messages) et la pagination côté applet.
package forum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Bornes (serveur public : tout est plafonné).
const (
	MaxTitle          = 38  // titre d'un fil : tient sur une ligne de 40 colonnes
	MaxText           = 200 // corps d'un message (s'affiche sur plusieurs lignes)
	MinText           = 1   // message non vide après nettoyage
	MaxThreads        = 100 // fils conservés (le moins actif est évincé au-delà)
	MaxPostsPerThread = 500 // messages par fil
)

// Post est un message dans un fil.
type Post struct {
	Author string    `json:"author"`
	Text   string    `json:"text"`
	At     time.Time `json:"at"`
}

// Thread est un fil de discussion : un titre, un auteur, ses messages.
type Thread struct {
	ID      uint64    `json:"id"`
	Title   string    `json:"title"`
	Author  string    `json:"author"`
	Created time.Time `json:"created"`
	Posts   []Post    `json:"posts"`
}

// LastActivity renvoie la date du dernier message (ou la création si aucun).
func (t *Thread) LastActivity() time.Time {
	if n := len(t.Posts); n > 0 {
		return t.Posts[n-1].At
	}
	return t.Created
}

// Info est un résumé de fil pour la liste (sans les messages).
type Info struct {
	ID     uint64
	Title  string
	Author string
	Posts  int
	LastAt time.Time
}

// Store persiste les fils dans un fichier JSON.
type Store struct {
	mu      sync.Mutex
	path    string
	threads []*Thread
	nextID  uint64
	now     func() time.Time
}

// persisted est la forme sérialisée sur disque (compteur d'ID + fils).
type persisted struct {
	NextID  uint64    `json:"next_id"`
	Threads []*Thread `json:"threads"`
}

// Open charge le store depuis path. Fichier absent → store vide. path vide →
// store en mémoire (tests).
func Open(path string) (*Store, error) {
	s := &Store{path: path, nextID: 1, now: time.Now}
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
	var p persisted
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("JSON du forum invalide (%s) : %w", path, err)
	}
	s.threads = p.Threads
	s.nextID = p.NextID
	if s.nextID == 0 {
		s.nextID = 1
	}
	// Garde-fous : borne le nombre de fils et de messages même si le fichier en
	// contenait davantage.
	s.enforceCapsLocked()
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

// NewThread crée un fil avec son premier message. Titre et texte sont nettoyés
// et bornés ; tous deux doivent être non vides après nettoyage.
func (s *Store) NewThread(author, title, text string) (Thread, error) {
	t := sanitize(title, MaxTitle)
	if t == "" {
		return Thread{}, fmt.Errorf("titre vide")
	}
	body := sanitize(text, MaxText)
	if len(body) < MinText {
		return Thread{}, fmt.Errorf("message vide")
	}
	a := authorOr(author)

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	th := &Thread{
		ID:      s.nextID,
		Title:   t,
		Author:  a,
		Created: now,
		Posts:   []Post{{Author: a, Text: body, At: now}},
	}
	s.nextID++
	s.threads = append(s.threads, th)
	s.enforceCapsLocked()
	if err := s.saveLocked(); err != nil {
		return Thread{}, err
	}
	return *th, nil
}

// Reply ajoute un message à un fil existant. Erreur si le fil est introuvable ou
// le texte vide après nettoyage.
func (s *Store) Reply(id uint64, author, text string) (Post, error) {
	body := sanitize(text, MaxText)
	if len(body) < MinText {
		return Post{}, fmt.Errorf("message vide")
	}
	a := authorOr(author)

	s.mu.Lock()
	defer s.mu.Unlock()
	th := s.findLocked(id)
	if th == nil {
		return Post{}, fmt.Errorf("fil introuvable")
	}
	p := Post{Author: a, Text: body, At: s.now()}
	th.Posts = append(th.Posts, p)
	if len(th.Posts) > MaxPostsPerThread {
		th.Posts = th.Posts[len(th.Posts)-MaxPostsPerThread:]
	}
	if err := s.saveLocked(); err != nil {
		th.Posts = th.Posts[:len(th.Posts)-1] // rollback mémoire
		return Post{}, err
	}
	return p, nil
}

// List renvoie les résumés de fils, du plus récemment actif au plus ancien.
func (s *Store) List() []Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Info, len(s.threads))
	for i, t := range s.threads {
		out[i] = Info{ID: t.ID, Title: t.Title, Author: t.Author, Posts: len(t.Posts), LastAt: t.LastActivity()}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].LastAt.After(out[j].LastAt) })
	return out
}

// Thread renvoie une copie du fil (messages inclus), ou false s'il n'existe pas.
func (s *Store) Thread(id uint64) (Thread, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	th := s.findLocked(id)
	if th == nil {
		return Thread{}, false
	}
	cp := *th
	cp.Posts = append([]Post(nil), th.Posts...)
	return cp, true
}

// Count renvoie le nombre de fils.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.threads)
}

// findLocked cherche un fil par ID (verrou détenu).
func (s *Store) findLocked(id uint64) *Thread {
	for _, t := range s.threads {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// enforceCapsLocked applique les plafonds (verrou détenu) : borne les messages
// par fil, puis évince les fils les moins récemment actifs au-delà de MaxThreads.
func (s *Store) enforceCapsLocked() {
	for _, t := range s.threads {
		if len(t.Posts) > MaxPostsPerThread {
			t.Posts = t.Posts[len(t.Posts)-MaxPostsPerThread:]
		}
	}
	if len(s.threads) <= MaxThreads {
		return
	}
	// Trie par activité décroissante et ne garde que les MaxThreads premiers.
	sort.SliceStable(s.threads, func(i, j int) bool {
		return s.threads[i].LastActivity().After(s.threads[j].LastActivity())
	})
	s.threads = s.threads[:MaxThreads]
}

// authorOr renvoie un pseudo nettoyé, ou « Anonyme » à défaut.
func authorOr(author string) string {
	a := oascii.SanitizeText(author)
	if a == "" {
		return "Anonyme"
	}
	if len(a) > 16 {
		a = a[:16]
	}
	return a
}

// saveLocked écrit le store de façon atomique (verrou détenu). No-op en mémoire.
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(persisted{NextID: s.nextID, Threads: s.threads}, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation du forum : %w", err)
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".forum-*.json.tmp")
	if err != nil {
		return fmt.Errorf("fichier temporaire : %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

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
