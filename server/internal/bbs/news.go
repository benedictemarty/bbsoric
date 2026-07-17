package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/news"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

const newsPerPage = 3 // annonces affichées par page

// init enregistre l'applet des actualités.
func init() {
	Register("news", newsApplet)
}

// newsApplet affiche les actualités / annonces datées (lecture pour tous). Un
// admin (sysop) peut en publier (touche N). Adossé à server/internal/news.
func newsApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	store := ac.State.News
	if store == nil {
		header(s, "ACTUALITES")
		writeErr(s, "Actualites indisponibles.")
		anyKey(s)
		return Outcome{}
	}
	admin := ac.State.IsAdmin()
	page := 0
	for {
		items := store.List(0)
		nbPages := pageCount(len(items), newsPerPage)
		page = clampPage(page, nbPages)
		if err := writeNews(s, items, page, nbPages, admin); err != nil {
			return Outcome{Quit: true}
		}
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		switch upperKey(key) {
		case 'Q':
			return Outcome{}
		case 'N':
			if admin {
				if out := newsCompose(s, ac, store); out.Quit {
					return out
				}
			}
		case 'S':
			if page < nbPages-1 {
				page++
			}
		case 'P':
			if page > 0 {
				page--
			}
		}
	}
}

// writeNews rend une page d'annonces (antéchronologique).
func writeNews(s *server.Session, items []news.Item, page, nbPages int, admin bool) error {
	header(s, "ACTUALITES")
	b := oascii.New()
	if len(items) == 0 {
		b.Ink(oascii.Magenta).Text(" Aucune annonce pour le moment.").Newline().Newline()
	} else {
		start := page * newsPerPage
		end := min(start+newsPerPage, len(items))
		for i := start; i < end; i++ {
			it := items[i]
			b.Ink(oascii.Cyan).Text(" " + it.At.Format("02/01/2006") + " ")
			b.Ink(oascii.Yellow).Text(trunc(it.Title, oascii.Cols-13)).Newline()
			for _, seg := range wrapText(it.Body, oascii.Cols-2) {
				b.Ink(oascii.White).Text(" " + seg).Newline()
			}
			b.Newline()
		}
	}
	b.Ink(oascii.Green).Text(fmt.Sprintf(" Page %d/%d  ", page+1, nbPages))
	if admin {
		b.Ink(oascii.Green).Text("N").Ink(oascii.White).Text("=publier  ")
	}
	b.Ink(oascii.Green).Text("S/P").Ink(oascii.White).Text("=page  ")
	b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
	return s.Write(b.String())
}

// newsCompose publie une annonce (titre + corps). Corps vide = annulation.
// Appelé seulement pour un admin.
func newsCompose(s *server.Session, ac *AppContext, store *news.Store) Outcome {
	header(s, "NOUVELLE ANNONCE")
	title, err := prompt(s, "Titre (vide=Annonce)")
	if err != nil {
		return Outcome{Quit: true}
	}
	body, err := prompt(s, "Texte (vide=annuler)")
	if err != nil {
		return Outcome{Quit: true}
	}
	if strings.TrimSpace(body) == "" {
		return Outcome{}
	}
	author := "sysop"
	if ac.State.User != nil {
		author = ac.State.User.Handle
	}
	if _, err := store.Post(author, title, body); err != nil {
		writeErr(s, "Refuse : "+err.Error())
		anyKey(s)
	}
	return Outcome{}
}
