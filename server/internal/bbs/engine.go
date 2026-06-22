package bbs

import (
	"context"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
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

		switch page.Type {
		case "menu":
			if !navigateMenu(s, page, &stack, site) {
				return
			}
		case "applet":
			if !runAppletPage(ctx, s, page, &stack, users, state) {
				return
			}
		default: // "page" : écran de contenu
			if err := renderContent(s, page); err != nil {
				return
			}
			if _, err := s.ReadKey(); err != nil { // une touche pour revenir
				return
			}
			popOrHome(&stack, site)
		}
	}
}

// navigateMenu affiche un menu, lit une touche et applique le choix. Renvoie
// false si la session doit se terminer (erreur I/O ou quitter).
func navigateMenu(s *server.Session, page *content.Page, stack *[]string, site *content.Site) bool {
	if err := renderMenu(s, page); err != nil {
		return false
	}
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

// runAppletPage affiche l'intro éventuelle, exécute l'applet et applique son
// Outcome. Renvoie false si la session doit se terminer.
func runAppletPage(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, users *user.Store, state *SessionState) bool {
	if len(page.Lines) > 0 {
		if err := renderContent(s, page); err != nil {
			return false
		}
	}
	app, ok := lookupApplet(page.Applet)
	if !ok {
		b := oascii.New()
		b.Ink(oascii.Red).Text("Applet \"" + page.Applet + "\" indisponible.").Newline()
		_ = s.Write(b.String())
		*stack = (*stack)[:len(*stack)-1] // on quitte la page applet
		return len(*stack) > 0
	}
	out := app(ctx, s, &AppContext{Users: users, State: state, Page: page})
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

// renderMenu affiche un menu en OASCII et l'invite de saisie.
func renderMenu(s *server.Session, p *content.Page) error {
	b := oascii.New()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(p.Title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	for _, e := range p.Entries {
		b.Ink(oascii.Cyan).Text(" " + e.Key)
		b.Ink(oascii.White).Text(" - " + e.Label).Newline()
	}
	b.Newline()
	if err := s.Write(b.String()); err != nil {
		return err
	}
	return s.Write(makeInk(oascii.Green) + "Votre choix" + makeInk(oascii.White) + "> ")
}

// renderContent affiche un écran de contenu (lignes + invite « une touche »).
func renderContent(s *server.Session, p *content.Page) error {
	b := oascii.New()
	b.Newline()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(p.Title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	for _, ln := range p.Lines {
		b.Ink(content.Ink(ln.Ink)).Text(ln.Text).Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	return s.Write(b.String())
}
