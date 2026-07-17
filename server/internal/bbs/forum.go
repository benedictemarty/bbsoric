package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/forum"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

const (
	forumThreadsPerPage = 8 // fils listés par page (touches 1..8)
	forumPostsPerPage   = 5 // messages affichés par page dans un fil
)

// init enregistre l'applet forum.
func init() {
	Register("forum", forumApplet)
}

// forumApplet est le forum de discussion : liste paginée de fils, lecture d'un
// fil (messages paginés), création de fil et réponse. Second applet à écriture
// persistée (après « wall »), adossé à server/internal/forum.
func forumApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	store := ac.State.Forum
	if store == nil {
		header(s, "FORUM")
		writeErr(s, "Forum indisponible.")
		anyKey(s)
		return Outcome{}
	}
	return forumList(s, ac, store)
}

// forumList affiche la liste des fils et route les choix (ouvrir, nouveau,
// pagination, retour). Boucle jusqu'au retour ou à la déconnexion.
func forumList(s *server.Session, ac *AppContext, store *forum.Store) Outcome {
	page := 0
	for {
		infos := store.List()
		nbPages := pageCount(len(infos), forumThreadsPerPage)
		if page >= nbPages {
			page = nbPages - 1
		}
		if page < 0 {
			page = 0
		}
		if err := writeForumList(s, infos, page, nbPages); err != nil {
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
			if out := forumNewThread(s, ac, store); out.Quit {
				return out
			}
		case 'S':
			if page < nbPages-1 {
				page++
			}
		case 'P':
			if page > 0 {
				page--
			}
		default:
			if idx := int(key) - '1'; idx >= 0 && idx < forumThreadsPerPage {
				gi := page*forumThreadsPerPage + idx
				if gi < len(infos) {
					if out := forumThread(s, ac, store, infos[gi].ID); out.Quit {
						return out
					}
				}
			}
		}
	}
}

// writeForumList rend une page de la liste des fils.
func writeForumList(s *server.Session, infos []forum.Info, page, nbPages int) error {
	header(s, "FORUM")
	b := oascii.New()
	if len(infos) == 0 {
		b.Ink(oascii.Magenta).Text(" Aucun fil. Ouvrez le premier sujet !").Newline().Newline()
		b.Ink(oascii.Green).Text(" N").Ink(oascii.White).Text("=nouveau  ")
		b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
		return s.Write(b.String())
	}
	start := page * forumThreadsPerPage
	end := min(start+forumThreadsPerPage, len(infos))
	for i := start; i < end; i++ {
		in := infos[i]
		b.Ink(oascii.Yellow).Text(fmt.Sprintf(" %d ", i-start+1))
		b.Ink(oascii.White).Text(trunc(in.Title, oascii.Cols-4)).Newline()
		b.Ink(oascii.Cyan).Text(fmt.Sprintf("   %s - %d msg - %s",
			trunc(in.Author, 12), in.Posts, in.LastAt.Format("02/01 15:04"))).Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text(fmt.Sprintf(" Page %d/%d  ", page+1, nbPages))
	b.Ink(oascii.Green).Text("1-8").Ink(oascii.White).Text("=ouvrir  ")
	b.Ink(oascii.Green).Text("N").Ink(oascii.White).Text("=nouveau  ")
	b.Ink(oascii.Green).Text("S/P").Ink(oascii.White).Text("=page  ")
	b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
	return s.Write(b.String())
}

// forumThread affiche un fil (messages paginés) et route les choix (répondre,
// pagination, retour). Boucle jusqu'au retour ou à la déconnexion.
func forumThread(s *server.Session, ac *AppContext, store *forum.Store, id uint64) Outcome {
	page := 0
	for {
		th, ok := store.Thread(id)
		if !ok {
			return Outcome{} // fil disparu : retour à la liste
		}
		nbPages := pageCount(len(th.Posts), forumPostsPerPage)
		if page >= nbPages {
			page = nbPages - 1
		}
		if page < 0 {
			page = 0
		}
		if err := writeForumThread(s, th, page, nbPages); err != nil {
			return Outcome{Quit: true}
		}
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		switch upperKey(key) {
		case 'Q', 'B':
			return Outcome{}
		case 'R':
			text, err := prompt(s, "Reponse (vide=annuler)")
			if err != nil {
				return Outcome{Quit: true}
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			if _, err := store.Reply(id, ac.State.displayName(), text); err != nil {
				writeErr(s, "Refuse : "+err.Error())
				anyKey(s)
			} else {
				page = pageCount(len(th.Posts)+1, forumPostsPerPage) - 1 // saute à la dernière page
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

// writeForumThread rend une page de messages d'un fil.
func writeForumThread(s *server.Session, th forum.Thread, page, nbPages int) error {
	header(s, "FORUM")
	b := oascii.New()
	b.Ink(oascii.Yellow).Text(" " + trunc(th.Title, oascii.Cols-2)).Newline()
	b.Ink(oascii.White).Text(strings.Repeat("-", oascii.Cols-1)).Newline()
	start := page * forumPostsPerPage
	end := min(start+forumPostsPerPage, len(th.Posts))
	for i := start; i < end; i++ {
		p := th.Posts[i]
		b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %s %s", trunc(p.Author, 14), p.At.Format("02/01 15:04"))).Newline()
		for _, seg := range wrapText(p.Text, oascii.Cols-2) {
			b.Ink(oascii.Green).Text(" " + seg).Newline()
		}
	}
	b.Newline()
	b.Ink(oascii.Green).Text(fmt.Sprintf(" Page %d/%d  ", page+1, nbPages))
	b.Ink(oascii.Green).Text("R").Ink(oascii.White).Text("=repondre  ")
	b.Ink(oascii.Green).Text("S/P").Ink(oascii.White).Text("=page  ")
	b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
	return s.Write(b.String())
}

// forumNewThread crée un fil (titre + premier message). Titre vide = annulation.
func forumNewThread(s *server.Session, ac *AppContext, store *forum.Store) Outcome {
	header(s, "NOUVEAU FIL")
	title, err := prompt(s, "Titre (vide=annuler)")
	if err != nil {
		return Outcome{Quit: true}
	}
	if strings.TrimSpace(title) == "" {
		return Outcome{}
	}
	text, err := prompt(s, "Message")
	if err != nil {
		return Outcome{Quit: true}
	}
	if _, err := store.NewThread(ac.State.displayName(), title, text); err != nil {
		writeErr(s, "Refuse : "+err.Error())
		anyKey(s)
	}
	return Outcome{}
}

// pageCount renvoie le nombre de pages (au moins 1) pour n éléments par page.
func pageCount(n, perPage int) int {
	if n <= 0 {
		return 1
	}
	return (n + perPage - 1) / perPage
}
