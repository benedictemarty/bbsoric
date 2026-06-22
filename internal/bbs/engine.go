package bbs

import (
	"context"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/server"
)

// siteOf renvoie le Site courant du store (contenu par défaut si store nil).
func siteOf(store *content.Store) *content.Site {
	if store == nil {
		return content.DefaultSite()
	}
	return store.Site()
}

// runSite déroule le flux de pages décrit par le Site (rechargé à chaque écran,
// donc une modification à chaud du JSON s'applique dès la navigation suivante).
// Une pile gère le retour arrière.
func runSite(ctx context.Context, s *server.Session, store *content.Store) {
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

		if err := renderPage(s, page); err != nil {
			return
		}

		if page.Type == "menu" {
			line, err := s.ReadLine()
			if err != nil {
				return
			}
			key := firstKey(line)
			if key == 0 {
				continue
			}
			e := findEntry(page, key)
			if e == nil {
				b := oascii.New()
				b.Ink(oascii.Red).Text("Choix invalide.").Newline()
				if s.Write(b.String()) != nil {
					return
				}
				continue
			}
			switch e.Target {
			case content.TargetQuit:
				b := oascii.New()
				b.Newline().Ink(oascii.Yellow).Text(center("A bientot sur le BBS Oric !")).Newline()
				_ = s.Write(b.String())
				return
			case content.TargetBack:
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
				}
			case content.TargetHome:
				stack = []string{site.Start}
			default:
				stack = append(stack, e.Target)
			}
			continue
		}

		// Page de contenu : RETURN revient en arrière (ou à l'accueil à la racine).
		if _, err := s.ReadLine(); err != nil {
			return
		}
		if len(stack) > 1 {
			stack = stack[:len(stack)-1]
		} else {
			stack = []string{site.Start}
		}
	}
}

// findEntry cherche l'entrée dont la touche (insensible à la casse) correspond.
func findEntry(p *content.Page, key byte) *content.Entry {
	for i := range p.Entries {
		ek := p.Entries[i].Key
		if ek == "" {
			continue
		}
		k := ek[0]
		if k >= 'a' && k <= 'z' {
			k -= 'a' - 'A'
		}
		if k == key {
			return &p.Entries[i]
		}
	}
	return nil
}

// renderPage affiche un menu ou un écran de contenu en OASCII.
func renderPage(s *server.Session, p *content.Page) error {
	b := oascii.New()
	if p.Type == "menu" {
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

	// type "page" : contenu + retour
	b.Newline()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(p.Title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	for _, ln := range p.Lines {
		b.Ink(content.Ink(ln.Ink)).Text(ln.Text).Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text("[RETURN] retour au menu").Newline()
	return s.Write(b.String())
}
