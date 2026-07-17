package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/pm"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

const pmPerPage = 8 // messages listés par page (touches 1..8)

// init enregistre l'applet de messagerie privée.
func init() {
	Register("pm", pmApplet)
}

// pmApplet est la messagerie privée entre membres. Réservé aux comptes
// identifiés (pas les invités). Boîte de réception paginée, lecture (marquage
// lu), réponse et nouveau message. Adossé à server/internal/pm.
func pmApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "MESSAGERIE PRIVEE")
	store := ac.State.PM
	if store == nil {
		writeErr(s, "Messagerie indisponible.")
		anyKey(s)
		return Outcome{}
	}
	if ac.State.User == nil {
		writeErr(s, "Reserve aux membres. Connectez-vous.")
		anyKey(s)
		return Outcome{}
	}
	return pmInbox(s, ac, store, ac.State.User.Handle)
}

// pmInbox affiche la boîte de réception et route les choix.
func pmInbox(s *server.Session, ac *AppContext, store *pm.Store, me string) Outcome {
	page := 0
	for {
		box := store.Inbox(me)
		nbPages := pageCount(len(box), pmPerPage)
		if page >= nbPages {
			page = nbPages - 1
		}
		if page < 0 {
			page = 0
		}
		if err := writePMInbox(s, store, me, box, page, nbPages); err != nil {
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
			if out := pmCompose(s, ac, store, me, ""); out.Quit {
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
			if idx := int(key) - '1'; idx >= 0 && idx < pmPerPage {
				gi := page*pmPerPage + idx
				if gi < len(box) {
					if out := pmRead(s, ac, store, me, box[gi]); out.Quit {
						return out
					}
				}
			}
		}
	}
}

// writePMInbox rend une page de la boîte de réception.
func writePMInbox(s *server.Session, store *pm.Store, me string, box []pm.Message, page, nbPages int) error {
	header(s, "MESSAGERIE PRIVEE")
	b := oascii.New()
	unread := store.Unread(me)
	b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %d recu(s), %d non lu(s)", len(box), unread)).Newline().Newline()
	if len(box) == 0 {
		b.Ink(oascii.Magenta).Text(" Boite vide.").Newline().Newline()
		b.Ink(oascii.Green).Text(" N").Ink(oascii.White).Text("=nouveau  ")
		b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
		return s.Write(b.String())
	}
	start := page * pmPerPage
	end := min(start+pmPerPage, len(box))
	for i := start; i < end; i++ {
		m := box[i]
		mark := " "
		if !m.Read {
			mark = "*" // non lu
		}
		b.Ink(oascii.Yellow).Text(fmt.Sprintf(" %d%s ", i-start+1, mark))
		b.Ink(oascii.White).Text(fmt.Sprintf("%-14s %s", trunc(m.From, 14), m.At.Format("02/01 15:04"))).Newline()
		b.Ink(oascii.Green).Text("   " + trunc(m.Text, oascii.Cols-4)).Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text(fmt.Sprintf(" Page %d/%d  ", page+1, nbPages))
	b.Ink(oascii.Green).Text("1-8").Ink(oascii.White).Text("=lire  ")
	b.Ink(oascii.Green).Text("N").Ink(oascii.White).Text("=nouveau  ")
	b.Ink(oascii.Green).Text("S/P").Ink(oascii.White).Text("=page  ")
	b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
	return s.Write(b.String())
}

// pmRead affiche un message en entier, le marque lu, et propose de répondre.
func pmRead(s *server.Session, ac *AppContext, store *pm.Store, me string, m pm.Message) Outcome {
	_ = store.MarkRead(me, m.ID) // lecture : marque lu (no-op si déjà lu)
	header(s, "MESSAGERIE PRIVEE")
	b := oascii.New()
	b.Ink(oascii.Cyan).Text(" De   : ").Ink(oascii.Yellow).Text(trunc(m.From, 16)).Newline()
	b.Ink(oascii.Cyan).Text(" Date : ").Ink(oascii.White).Text(m.At.Format("02/01/2006 15:04")).Newline()
	b.Ink(oascii.White).Text(strings.Repeat("-", oascii.Cols-1)).Newline()
	for _, seg := range wrapText(m.Text, oascii.Cols-2) {
		b.Ink(oascii.Green).Text(" " + seg).Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text(" R").Ink(oascii.White).Text("=repondre  ")
	b.Ink(oascii.Green).Text("Q").Ink(oascii.White).Text("=retour").Newline()
	if s.Write(b.String()) != nil {
		return Outcome{Quit: true}
	}
	for {
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		switch upperKey(key) {
		case 'R':
			return pmCompose(s, ac, store, me, m.From)
		case 'Q', 'B':
			return Outcome{}
		}
	}
}

// pmCompose rédige un message. Si to est vide, il est demandé et validé (le
// destinataire doit être un compte existant). Corps vide = annulation.
func pmCompose(s *server.Session, ac *AppContext, store *pm.Store, me, to string) Outcome {
	header(s, "NOUVEAU MESSAGE")
	if to == "" {
		dest, err := prompt(s, "Destinataire (vide=annuler)")
		if err != nil {
			return Outcome{Quit: true}
		}
		if strings.TrimSpace(dest) == "" {
			return Outcome{}
		}
		to = dest
	}
	// Valide l'existence du destinataire et récupère sa casse canonique.
	if ac.Users == nil {
		writeErr(s, "Comptes indisponibles.")
		anyKey(s)
		return Outcome{}
	}
	u, ok := ac.Users.Get(to)
	if !ok {
		writeErr(s, "Destinataire inconnu : "+trunc(strings.TrimSpace(to), 16))
		anyKey(s)
		return Outcome{}
	}
	text, err := prompt(s, "Message")
	if err != nil {
		return Outcome{Quit: true}
	}
	if strings.TrimSpace(text) == "" {
		return Outcome{}
	}
	if _, err := store.Send(me, u.Handle, text); err != nil {
		writeErr(s, "Refuse : "+err.Error())
		anyKey(s)
		return Outcome{}
	}
	ok2 := oascii.New()
	ok2.Ink(oascii.Green).Text(" Message envoye a " + trunc(u.Handle, 16) + ".").Newline()
	_ = s.Write(ok2.String())
	anyKey(s)
	return Outcome{}
}
