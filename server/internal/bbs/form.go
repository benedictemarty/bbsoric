package bbs

import (
	"context"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/render"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre l'applet générique de saisie déclarative.
func init() { Register("form", formApplet) }

// formApplet exécute un écran de saisie décrit par la page (content.Form) : il
// affiche le décor (buffer raw composé, ou bandeau de titre), saisit chaque
// champ dans l'ordre, puis applique l'action (login/inscription — logique en Go).
// La présentation est déclarative (studio) ; la logique sensible (hachage,
// vérification) reste centralisée côté serveur.
func formApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	f := ac.Page.Form
	if f == nil {
		writeErr(s, "Formulaire absent.")
		return Outcome{}
	}
	if ac.Users == nil {
		writeErr(s, "Comptes indisponibles.")
		return Outcome{}
	}

	// Décor : une page « écran brut » fournit le fond composé case par case,
	// affiché plein écran depuis le coin (0,0) — les champs se positionnent
	// ensuite par leurs coordonnées (At). Sinon, un bandeau de titre, puis les
	// invites séquentielles.
	if ac.Page.Raw {
		if s.Write(oascii.Plot(0, 0)+string(render.RawScreen(ac.Page))) != nil {
			return Outcome{Quit: true}
		}
	} else {
		header(s, ac.Page.Title)
	}

	// Saisie des champs dans l'ordre déclaré. Premier champ vide = annulation.
	vals := make(map[string]string, len(f.Fields))
	for i, fld := range f.Fields {
		if fld.Secret {
			warnCleartext(s)
		}
		// Positionnement absolu optionnel de l'invite (plot X,Y).
		if len(fld.At) == 2 {
			if s.Write(oascii.Plot(fld.At[0], fld.At[1])) != nil {
				return Outcome{Quit: true}
			}
		}
		v, err := prompt(s, fld.Label)
		if err != nil {
			return Outcome{Quit: true}
		}
		if i == 0 && strings.TrimSpace(v) == "" {
			return Outcome{} // annulation → retour au menu appelant
		}
		vals[fld.Key] = v
	}

	switch f.Action {
	case content.FormLogin:
		u, err := ac.Users.Authenticate(vals["login"], vals["password"])
		if err != nil {
			writeErr(s, "Echec : "+err.Error())
			return Outcome{}
		}
		ac.State.User = &u
		greet(s, u)
		return Outcome{Done: true}

	case content.FormRegister:
		if vals["password"] != vals["confirm"] {
			writeErr(s, "Les mots de passe different.")
			return Outcome{}
		}
		u, err := ac.Users.Register(vals["login"], vals["password"])
		if err != nil {
			writeErr(s, "Echec : "+err.Error())
			return Outcome{}
		}
		ac.State.User = &u
		b := oascii.New()
		b.Newline().Ink(oascii.Green).Text(center("Compte cree !")).Newline()
		b.Ink(oascii.Yellow).Text(center("Bienvenue " + u.Handle + " !")).Newline()
		b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
		_ = s.Write(b.String())
		_, _ = s.ReadKey()
		return Outcome{Done: true}

	default:
		writeErr(s, "Action inconnue : "+f.Action)
		return Outcome{}
	}
}
