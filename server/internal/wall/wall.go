// Package wall persiste le « mur de messages » du BBS : de courts messages
// publics (une ligne) laissés par les appelants, à la manière du livre d'or
// historique d'un BBS.
//
// Le store suit le même patron que server/internal/user : verrou (1 goroutine
// par session), écriture atomique (fichier temporaire + rename) et horloge
// injectable (now) pour des tests déterministes. C'est le premier applet à
// écriture persistée ; le forum (Sprint 7 #1) réutilisera ce patron.
//
// Contrainte serveur public : les entrées sont bornées (taille du message,
// nombre de messages conservés) et nettoyées en ASCII imprimable (l'Oric
// n'affiche que de l'ASCII ; on écarte les octets de contrôle qui casseraient
// le rendu OASCII).
package wall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Bornes appliquées à chaque message et au mur entier.
const (
	// MaxText borne la longueur d'un message. 78 caractères tiennent sur deux
	// lignes de 40 colonnes Oric une fois l'attribut de couleur retiré.
	MaxText = 78
	// MinText impose un contenu non vide après nettoyage.
	MinText = 1
	// MaxMessages borne le nombre de messages conservés (les plus anciens sont
	// évincés). Garde-fou mémoire/disque pour un serveur public.
	MaxMessages = 200
)

// Message est une entrée du mur : un pseudo, un texte, une date.
type Message struct {
	Handle string    `json:"handle"` // pseudo de l'auteur (au moment de la publication)
	Text   string    `json:"text"`   // texte nettoyé (ASCII imprimable, borné)
	At     time.Time `json:"at"`     // date de publication
}

// Store persiste les messages du mur dans un fichier JSON.
type Store struct {
	mu   sync.Mutex
	path string
	msgs []Message
	now  func() time.Time
}

// Open charge le store depuis path. Si le fichier n'existe pas, le store
// démarre vide (créé à la première écriture). Un path vide donne un store
// purement en mémoire (utile pour les tests).
func Open(path string) (*Store, error) {
	s := &Store{path: path, now: time.Now}
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
	if err := json.Unmarshal(b, &s.msgs); err != nil {
		return nil, fmt.Errorf("JSON du mur invalide (%s) : %w", path, err)
	}
	// Applique le plafond même si le fichier en contenait davantage (borne dure).
	if len(s.msgs) > MaxMessages {
		s.msgs = s.msgs[len(s.msgs)-MaxMessages:]
	}
	return s, nil
}

// Sanitize nettoie un texte de message (ASCII imprimable, blancs compactés, bords
// coupés — cf. oascii.SanitizeText) puis borne sa longueur à MaxText.
func Sanitize(text string) string {
	out := oascii.SanitizeText(text)
	if len(out) > MaxText {
		out = strings.TrimSpace(out[:MaxText])
	}
	return out
}

// Post ajoute un message au mur. handle est le pseudo affiché de l'auteur ;
// text est nettoyé puis validé (non vide après nettoyage). Le message est
// persisté de façon atomique ; les plus anciens sont évincés au-delà de
// MaxMessages.
func (s *Store) Post(handle, text string) (Message, error) {
	clean := Sanitize(text)
	if len(clean) < MinText {
		return Message{}, fmt.Errorf("message vide")
	}
	h := Sanitize(handle)
	if h == "" {
		h = "Anonyme"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	m := Message{Handle: h, Text: clean, At: s.now()}
	s.msgs = append(s.msgs, m)
	if len(s.msgs) > MaxMessages {
		s.msgs = s.msgs[len(s.msgs)-MaxMessages:]
	}
	if err := s.saveLocked(); err != nil {
		s.msgs = s.msgs[:len(s.msgs)-1] // rollback mémoire si la persistance échoue
		return Message{}, err
	}
	return m, nil
}

// List renvoie jusqu'à n messages, du plus récent au plus ancien (copie). n<=0
// renvoie tout le mur.
func (s *Store) List(n int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := len(s.msgs)
	if n <= 0 || n > total {
		n = total
	}
	out := make([]Message, n)
	for i := 0; i < n; i++ {
		out[i] = s.msgs[total-1-i] // ordre antéchronologique
	}
	return out
}

// Count renvoie le nombre de messages conservés.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

// saveLocked écrit le store sur disque de façon atomique. À appeler avec le
// verrou détenu. No-op si le store est en mémoire (path vide).
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s.msgs, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation du mur : %w", err)
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".wall-*.json.tmp")
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
