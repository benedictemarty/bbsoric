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

		switch {
		case page.Applet != "": // page applet auto-lancée (compat JSON manuel)
			if !runAppletPage(ctx, s, page, &stack, users, state) {
				return
			}
		case len(page.Entries) > 0: // écran interactif (menu, avec texte optionnel)
			if !navigateMenu(ctx, s, page, &stack, site, users, state) {
				return
			}
		default: // écran de contenu : une touche pour revenir
			if err := renderContent(s, page); err != nil {
				return
			}
			if _, err := s.ReadKey(); err != nil {
				return
			}
			popOrHome(&stack, site)
		}
	}
}

// navigateMenu affiche un menu, lit une touche et applique le choix. Une entrée
// peut lancer un applet (e.Applet) au lieu de naviguer (e.Target) — un menu peut
// donc proposer plusieurs applets au choix. Renvoie false si la session doit se
// terminer (erreur I/O ou quitter).
func navigateMenu(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, site *content.Site, users *user.Store, state *SessionState) bool {
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

	// Entrée-applet : on lance l'applet ; succès -> page Next (si définie),
	// sinon on reste sur le menu.
	if e.Applet != "" {
		out := runApplet(ctx, s, e.Applet, page, users, state)
		if out.Quit {
			return false
		}
		if out.Done && e.Next != "" {
			*stack = append(*stack, e.Next)
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
func runApplet(ctx context.Context, s *server.Session, name string, page *content.Page, users *user.Store, state *SessionState) Outcome {
	app, ok := lookupApplet(name)
	if !ok {
		b := oascii.New()
		b.Ink(oascii.Red).Text("Applet \"" + name + "\" indisponible.").Newline()
		_ = s.Write(b.String())
		return Outcome{}
	}
	return app(ctx, s, &AppContext{Users: users, State: state, Page: page})
}

// runAppletPage affiche l'intro éventuelle, exécute l'applet et applique son
// Outcome. Renvoie false si la session doit se terminer.
func runAppletPage(ctx context.Context, s *server.Session, page *content.Page, stack *[]string, users *user.Store, state *SessionState) bool {
	if len(page.Lines) > 0 {
		if err := renderContent(s, page); err != nil {
			return false
		}
	}
	out := runApplet(ctx, s, page.Applet, page, users, state)
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

// renderMenu affiche un écran interactif : titre, texte d'intro optionnel
// (Lines), choix (Entries) et invite de saisie.
func renderMenu(s *server.Session, p *content.Page) error {
	b := oascii.New()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(p.Title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	for _, ln := range p.Lines { // texte d'intro éventuel, au-dessus des choix
		writeLine(b, ln)
	}
	if len(p.Lines) > 0 {
		b.Newline()
	}
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
		writeLine(b, ln)
	}
	b.Newline()
	b.Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	return s.Write(b.String())
}

// writeLine ajoute une ligne de contenu avec ses attributs Oric : fond (paper),
// clignotement / double hauteur (un seul octet), puis encre et texte.
func writeLine(b *oascii.Builder, ln content.Line) {
	if ln.Paper != "" {
		b.Paper(content.Ink(ln.Paper))
	}
	if ln.Blink || ln.DoubleHeight {
		b.Attrs(ln.Blink, ln.DoubleHeight, false)
	}
	b.Ink(content.Ink(ln.Ink)).Text(ln.Text).Newline()
}
