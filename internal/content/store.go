package content

import (
	"log/slog"
	"os"
	"sync"
	"time"
)

// Store détient le Site courant et le recharge à chaud quand le fichier JSON
// change. En cas d'erreur de rechargement, l'ancienne version est conservée.
type Store struct {
	path string
	log  *slog.Logger

	mu    sync.RWMutex
	site  *Site
	mtime time.Time
}

// PollInterval est la période de surveillance du fichier de contenu.
var PollInterval = 2 * time.Second

// NewStore crée un Store. Si path est vide, le contenu par défaut est utilisé
// (sans rechargement). Sinon le fichier est chargé puis surveillé en tâche de
// fond ; en cas d'échec initial, le contenu par défaut sert de repli.
func NewStore(path string, log *slog.Logger) *Store {
	if log == nil {
		log = slog.Default()
	}
	s := &Store{path: path, log: log, site: DefaultSite()}
	if path != "" {
		if err := s.reload(); err != nil {
			s.log.Warn("contenu : chargement initial échoué, contenu par défaut utilisé",
				"path", path, "err", err)
		}
		go s.watch()
	}
	return s
}

// Site renvoie le Site courant (sûr en concurrence).
func (s *Store) Site() *Site {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.site
}

func (s *Store) reload() error {
	fi, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	site, err := Parse(b)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.site = site
	s.mtime = fi.ModTime()
	s.mu.Unlock()
	s.log.Info("contenu rechargé", "path", s.path, "pages", len(site.Pages))
	return nil
}

func (s *Store) watch() {
	t := time.NewTicker(PollInterval)
	defer t.Stop()
	for range t.C {
		fi, err := os.Stat(s.path)
		if err != nil {
			continue // fichier temporairement absent (édition) : on réessaie
		}
		s.mu.RLock()
		// Tout changement de mtime (y compris vers une date plus ancienne, ex.
		// restauration par mv/cp d'une sauvegarde) déclenche un rechargement.
		changed := !fi.ModTime().Equal(s.mtime)
		s.mu.RUnlock()
		if !changed {
			continue
		}
		if err := s.reload(); err != nil {
			s.log.Warn("contenu : rechargement échoué, ancienne version conservée", "err", err)
			// On mémorise quand même le mtime pour ne pas réessayer en boucle
			// le même fichier invalide à chaque tick.
			s.mu.Lock()
			s.mtime = fi.ModTime()
			s.mu.Unlock()
		}
	}
}
