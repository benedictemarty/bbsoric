// Package throttle limite le débit d'événements par clé — typiquement les
// tentatives d'authentification par adresse IP — via une fenêtre glissante :
// au plus N échecs autorisés par fenêtre. Sert de garde-fou anti brute-force,
// en complément du plafond de tentatives par passage d'applet (cf. S11.4).
//
// L'horloge est injectable (now) pour des tests déterministes.
package throttle

import (
	"sync"
	"time"
)

// Limiter compte les échecs par clé sur une fenêtre glissante. Sûr en concurrence.
// Un Limiter nil (ou de capacité <= 0) est un no-op : tout est autorisé.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	fails  map[string][]time.Time
	now    func() time.Time
}

// New crée un limiteur autorisant au plus max échecs par fenêtre window.
func New(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, fails: map[string][]time.Time{}, now: time.Now}
}

// Allowed indique si une nouvelle tentative est permise pour la clé (strictement
// moins de max échecs dans la fenêtre courante). N'enregistre rien.
func (l *Limiter) Allowed(key string) bool {
	if l == nil || l.max <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentLocked(key)) < l.max
}

// Fail enregistre un échec pour la clé (horodaté à l'instant courant).
func (l *Limiter) Fail(key string) {
	if l == nil || l.max <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fails[key] = append(l.recentLocked(key), l.now())
}

// Reset efface les échecs d'une clé (ex. après une connexion réussie).
func (l *Limiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

// recentLocked renvoie les échecs de la clé encore dans la fenêtre, en élaguant
// au passage les entrées expirées. À appeler avec le verrou détenu.
func (l *Limiter) recentLocked(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	src := l.fails[key]
	kept := src[:0] // réutilise le tableau sous-jacent (on n'avance que vers l'avant)
	for _, t := range src {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.fails, key) // évite l'accumulation de clés vides
		return nil
	}
	l.fails[key] = kept
	return kept
}
