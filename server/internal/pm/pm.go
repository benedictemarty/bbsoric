// Package pm persiste la messagerie privée du BBS : des messages point à point
// entre comptes identifiés (pas les invités), avec boîte de réception et
// marqueur « lu / non lu ».
//
// Troisième applet à écriture persistée après le mur (server/internal/wall) et
// le forum (server/internal/forum) ; même patron : store JSON à verrou,
// écriture atomique (temp+rename), horloge injectable, entrées bornées et
// nettoyées ASCII. La correspondance destinataire est insensible à la casse
// (user.NormalizeHandle, comme les comptes).
package pm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// Bornes (serveur public : tout est plafonné).
const (
	MaxText     = 200  // corps d'un message privé
	MinText     = 1    // non vide après nettoyage
	MaxMessages = 1000 // messages conservés au total (le plus ancien est évincé)
)

// Message est un message privé.
type Message struct {
	ID   uint64    `json:"id"`
	From string    `json:"from"` // pseudo canonique de l'expéditeur
	To   string    `json:"to"`   // pseudo canonique du destinataire
	Text string    `json:"text"`
	At   time.Time `json:"at"`
	Read bool      `json:"read"` // lu par le destinataire
}

// Store persiste les messages privés dans un fichier JSON.
type Store struct {
	mu     sync.Mutex
	path   string
	msgs   []Message
	nextID uint64
	now    func() time.Time
}

type persisted struct {
	NextID   uint64    `json:"next_id"`
	Messages []Message `json:"messages"`
}

// Open charge le store depuis path. Fichier absent → vide. path vide → mémoire.
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
		return nil, fmt.Errorf("JSON de la messagerie invalide (%s) : %w", path, err)
	}
	s.msgs = p.Messages
	s.nextID = p.NextID
	if s.nextID == 0 {
		s.nextID = 1
	}
	if len(s.msgs) > MaxMessages {
		s.msgs = s.msgs[len(s.msgs)-MaxMessages:]
	}
	return s, nil
}

// Send enregistre un message de from vers to. from et to sont des pseudos
// canoniques (le validateur d'existence du destinataire est à l'appelant). Le
// texte est nettoyé et borné ; il doit être non vide.
func (s *Store) Send(from, to, text string) (Message, error) {
	body := oascii.SanitizeText(text)
	if len(body) > MaxText {
		body = strings.TrimSpace(body[:MaxText])
	}
	if len(body) < MinText {
		return Message{}, fmt.Errorf("message vide")
	}
	if strings.TrimSpace(to) == "" {
		return Message{}, fmt.Errorf("destinataire manquant")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	m := Message{ID: s.nextID, From: from, To: to, Text: body, At: s.now()}
	s.nextID++
	s.msgs = append(s.msgs, m)
	if len(s.msgs) > MaxMessages {
		s.msgs = s.msgs[len(s.msgs)-MaxMessages:]
	}
	if err := s.saveLocked(); err != nil {
		s.msgs = s.msgs[:len(s.msgs)-1] // rollback mémoire
		return Message{}, err
	}
	return m, nil
}

// Inbox renvoie les messages reçus par handle (insensible à la casse), du plus
// récent au plus ancien (copie).
func (s *Store) Inbox(handle string) []Message {
	key := user.NormalizeHandle(handle)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Message
	for i := len(s.msgs) - 1; i >= 0; i-- {
		if user.NormalizeHandle(s.msgs[i].To) == key {
			out = append(out, s.msgs[i])
		}
	}
	return out
}

// Unread compte les messages non lus reçus par handle.
func (s *Store) Unread(handle string) int {
	key := user.NormalizeHandle(handle)
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for i := range s.msgs {
		if !s.msgs[i].Read && user.NormalizeHandle(s.msgs[i].To) == key {
			n++
		}
	}
	return n
}

// MarkRead marque comme lu le message id s'il est bien adressé à handle. No-op
// (sans erreur) si déjà lu ; erreur si le message n'existe pas ou n'appartient
// pas au destinataire.
func (s *Store) MarkRead(handle string, id uint64) error {
	key := user.NormalizeHandle(handle)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.msgs {
		if s.msgs[i].ID == id && user.NormalizeHandle(s.msgs[i].To) == key {
			if s.msgs[i].Read {
				return nil
			}
			s.msgs[i].Read = true
			return s.saveLocked()
		}
	}
	return fmt.Errorf("message introuvable")
}

// Count renvoie le nombre total de messages conservés.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

// saveLocked écrit le store de façon atomique (verrou détenu). No-op en mémoire.
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(persisted{NextID: s.nextID, Messages: s.msgs}, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation de la messagerie : %w", err)
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".pm-*.json.tmp")
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
