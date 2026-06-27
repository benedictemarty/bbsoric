package bbs

import (
	"context"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/render"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// siteOf renvoie le Site courant du store (contenu par défaut si store nil).
func siteOf(store *content.Store) *content.Site {
	if store == nil {
		return content.DefaultSite()
	}
	return store.Site()
}

// runSite déroule le flux de pages décrit par le Site (relu à chaque écran,
// donc une modification à chaud du JSON s'applique dès la navigation suivante).
// Une pile gère le retour arrière. La navigation réagit à une touche unique
// (ReadKey) ; les champs texte des applets utilisent ReadLine (cf. ADR-0002).
//
// Le rendu de l'écran provient de internal/render (source unique, partagée avec
// le studio).
func runSite(ctx context.Context, s *server.Session, store *content.Store, users *user.Store, state *SessionState) {
	if state == nil {
		state = &SessionState{}
	}
	stack := []string{siteOf(store).Start}
	for {
		if ctx.Err() != nil || len(stack) == 0 {
			return
		}
		site := siteOf(store) // relit le contenu (prise en compte du hot-reload)
		id := stack[len(stack)-1]
		page := site.Pages[id]
		if page == nil {
			// Page disparue (édition à chaud) : on retombe sur la page de départ.
			if id != site.Start {
				stack = []string{site.Start}
				continue
			}
			return
		}

		switch {
		case page.Applet != "": // page applet auto-lancée (compat JSON manuel)
			if !runAppletPage(ctx, s, page, &stack, site, users, state) {
				return
			}
		case page.Form != nil: // page de saisie déclarative (login/inscription)
			if !runFormPage(ctx, s, page, &stack, site, users, state) {
				return
			}
		case page.DataWindow != nil: // grille de données (CRUD)
			if runApplet(ctx, s, "datawindow", page, site, users, state).Quit {
				return
			}
			popOrHome(&stack, site)
		case page.Hires != nil: // page graphique haute résolution (240×200)
			if s.Write(string(render.Hires(page))) != nil {
				return
			}
			if len(page.Entries) > 0 { // menu (libellés en bas, lignes texte)
				if !routeMenuChoice(ctx, s, page, &stack, site, users, state) {
					return
				}
			} else { // décor seul : une touche pour revenir
				if _, err := s.ReadKey(); err != nil {
					return
				}
				popOrHome(&stack, site)
			}
		case len(page.Entries) > 0: // écran interactif (menu)
			// Décor : une page « écran brut » sert de fond composé case par case
			// (titre, libellés et invite sont déjà dessinés dedans) ; une page
			// normale est rendue avec titre + lignes + choix + invite. Dans les
			// deux cas, la navigation est pilotée par les entries (touche → cible
			// ou applet).
			var screen []byte
			if page.Raw {
				screen = render.RawScreen(page)
			} else {
				screen = render.Screen(page)
			}
			if s.Write(string(screen)) != nil {
				return
			}
			if !routeMenuChoice(ctx, s, page, &stack, site, users, state) {
				return
			}
		case page.Raw: // écran brut sans navigation : une touche pour sortir
			if s.Write(string(render.RawScreen(page))) != nil {
				return
			}
			if _, err := s.ReadKey(); err != nil {
				return
			}
			popOrHome(&stack, site)
		default: // écran de contenu : une touche pour revenir
			if s.Write(string(render.Screen(page))) != nil {
				return
			}
			if _, err := s.ReadKey(); err != nil {
				return
			}
			popOrHome(&stack, site)
		}
	}
}

// routeMenuChoice lit une touche et applique le choix correspondant. Une entrée
// peut lancer un applet (e.Applet) au lieu de naviguer (e.Target) — un menu peut
// donc proposer plusieurs applets au choix. Renvoie false si la session doit se
// terminer (erreur I/O ou quitter).
func routeMenuChoice(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, site *content.Site, users *user.Store, state *SessionState) bool {
	key, err := s.ReadKey()
	if err != nil {
		return false
	}
	e := findEntry(page, upperKey(key))
	if e == nil {
		b := oascii.New()
		b.Ink(oascii.Red).Text("Choix invalide.").Newline()
		return s.Write(b.String()) == nil
	}

	// Entrée-applet : on lance l'applet ; succès -> page Next (si définie),
	// sinon on reste sur le menu.
	if e.Applet != "" {
		out := runApplet(ctx, s, e.Applet, page, site, users, state)
		if out.Quit {
			return false
		}
		if out.Done && e.Next != "" {
			*stack = append(*stack, e.Next)
		}
		if out.Failed && e.Fail != "" {
			*stack = append(*stack, e.Fail)
		}
		return true
	}

	switch e.Target {
	case content.TargetQuit:
		b := oascii.New()
		b.Newline().Ink(oascii.Yellow).Text(center("A bientot sur le BBS Oric !")).Newline()
		_ = s.Write(b.String())
		return false
	case content.TargetBack:
		if len(*stack) > 1 {
			*stack = (*stack)[:len(*stack)-1]
		}
	case content.TargetHome:
		*stack = []string{site.Start}
	default:
		*stack = append(*stack, e.Target)
	}
	return true
}

// runApplet résout un applet par son nom et l'exécute. Si l'applet est inconnu,
// affiche une erreur et renvoie un Outcome neutre.
func runApplet(ctx context.Context, s *server.Session, name string, page *content.Page, site *content.Site, users *user.Store, state *SessionState) Outcome {
	app, ok := lookupApplet(name)
	if !ok {
		b := oascii.New()
		b.Ink(oascii.Red).Text("Applet \"" + name + "\" indisponible.").Newline()
		_ = s.Write(b.String())
		return Outcome{}
	}
	return app(ctx, s, &AppContext{Users: users, Files: state.Files, Data: state.Data, State: state, Site: site, Page: page})
}

// runFormPage exécute l'applet générique « form » d'une page de saisie et
// applique son Outcome (succès → Form.Next, sinon Page.Next). Renvoie false si
// la session doit se terminer.
func runFormPage(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, site *content.Site, users *user.Store, state *SessionState) bool {
	out := runApplet(ctx, s, "form", page, site, users, state)
	if out.Quit {
		return false
	}
	*stack = (*stack)[:len(*stack)-1]
	switch {
	case out.Done:
		next := page.Form.Next
		if next == "" {
			next = page.Next
		}
		if next != "" {
			*stack = append(*stack, next)
		}
	case out.Failed && page.Form.Fail != "":
		*stack = append(*stack, page.Form.Fail)
	}
	return len(*stack) > 0
}

// runAppletPage exécute l'applet auto-lancé d'une page applet (compat) et
// applique son Outcome. Renvoie false si la session doit se terminer.
func runAppletPage(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, site *content.Site, users *user.Store, state *SessionState) bool {
	out := runApplet(ctx, s, page.Applet, page, site, users, state)
	if out.Quit {
		return false
	}
	// On quitte toujours la page applet (succès → next, sinon retour).
	*stack = (*stack)[:len(*stack)-1]
	if out.Done && page.Next != "" {
		*stack = append(*stack, page.Next)
	}
	return len(*stack) > 0
}

// popOrHome dépile une page de contenu : retour arrière, ou page de départ à la
// racine.
func popOrHome(stack *[]string, site *content.Site) {
	if len(*stack) > 1 {
		*stack = (*stack)[:len(*stack)-1]
	} else {
		*stack = []string{site.Start}
	}
}

// findEntry cherche l'entrée dont la touche (insensible à la casse) correspond.
func findEntry(p *content.Page, key byte) *content.Entry {
	for i := range p.Entries {
		ek := p.Entries[i].Key
		if ek == "" {
			continue
		}
		if upperKey(ek[0]) == key {
			return &p.Entries[i]
		}
	}
	return nil
}
