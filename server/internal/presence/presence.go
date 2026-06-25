// Package presence tient le registre des appelants connectes (« qui est en
// ligne ») et un petit relais de discussion (chat) entre sessions.
//
// Tout est en memoire et protege par un mutex ; rien n'est persiste : l'etat
// disparait a l'arret du serveur, comme la liste des connectes elle-meme. Le
// relais de chat est un pub/sub a diffusion non bloquante (un abonne lent ne
// fige jamais l'emetteur), avec un petit rappel des messages recents pour les
// arrivants.
package presence

import (
	"sort"
	"sync"
	"time"
)

// backlogSize borne le nombre de messages recents conserves pour le rappel.
const backlogSize = 12

// subBuffer dimensionne le tampon par abonne (au-dela, les messages sont sautes).
const subBuffer = 32

// Member est l'instantane public d'un appelant connecte.
type Member struct {
	ID     uint64
	Handle string
	Since  time.Time
}

// Message est une ligne de discussion diffusee a tous les abonnes du chat.
type Message struct {
	FromID uint64    // identifiant de l'emetteur (0 pour le systeme)
	From   string    // pseudo affiche
	Text   string    // contenu
	At     time.Time // horodatage
	System bool       // message de service (arrivee/depart) plutot qu'un appelant
}

// member est l'etat interne d'une session connectee.
type member struct {
	Member
	ip  string
	sub chan Message // non-nil si la session est abonnee au chat
}

// Registry est le registre central, sur pour un usage concurrent.
type Registry struct {
	mu      sync.Mutex
	seq     uint64
	members map[uint64]*member
	backlog []Message
	now     func() time.Time // injectable pour les tests
}

// New cree un registre vide.
func New() *Registry {
	return &Registry{members: map[uint64]*member{}, now: time.Now}
}

// Join enregistre une session connectee et renvoie son identifiant interne.
// handle est le pseudo initial (souvent provisoire avant identification).
func (r *Registry) Join(handle, ip string) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	id := r.seq
	r.members[id] = &member{Member: Member{ID: id, Handle: handle, Since: r.now()}, ip: ip}
	return id
}

// SetHandle met a jour le pseudo affiche d'une session (apres login/invite).
func (r *Registry) SetHandle(id uint64, handle string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m := r.members[id]; m != nil {
		m.Handle = handle
	}
}

// Leave retire une session du registre (et la desabonne du chat si besoin).
func (r *Registry) Leave(id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, id)
}

// List renvoie l'instantane des connectes, trie par anciennete de connexion.
func (r *Registry) List() []Member {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Member, 0, len(r.members))
	for _, m := range r.members {
		out = append(out, m.Member)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Since.Equal(out[j].Since) {
			return out[i].ID < out[j].ID
		}
		return out[i].Since.Before(out[j].Since)
	})
	return out
}

// Count renvoie le nombre de connectes.
func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.members)
}

// Subscribe abonne une session au chat et renvoie son canal de reception ainsi
// que le rappel des messages recents. Un nouvel abonnement remplace le precedent.
func (r *Registry) Subscribe(id uint64) (<-chan Message, []Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan Message, subBuffer)
	if m := r.members[id]; m != nil {
		m.sub = ch
	}
	backlog := append([]Message(nil), r.backlog...)
	return ch, backlog
}

// Unsubscribe retire l'abonnement chat d'une session (sans la deconnecter). Le
// canal n'est pas ferme : Publish, sous verrou, ne touche plus qu'aux abonnes
// actifs, donc aucun envoi sur canal ferme n'est possible.
func (r *Registry) Unsubscribe(id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m := r.members[id]; m != nil {
		m.sub = nil
	}
}

// Publish diffuse un message a tous les abonnes du chat et l'ajoute au rappel.
// L'envoi est non bloquant : un abonne dont le tampon est plein perd le message
// (le serveur ne se fige jamais a cause d'une session lente).
func (r *Registry) Publish(msg Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg.At.IsZero() {
		msg.At = r.now()
	}
	r.backlog = append(r.backlog, msg)
	if len(r.backlog) > backlogSize {
		r.backlog = r.backlog[len(r.backlog)-backlogSize:]
	}
	for _, m := range r.members {
		if m.sub == nil {
			continue
		}
		select {
		case m.sub <- msg:
		default: // tampon plein : on saute (jamais bloquant)
		}
	}
}
