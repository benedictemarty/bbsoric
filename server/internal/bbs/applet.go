package bbs

import (
	"context"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// SessionState porte l'état d'une session : utilisateur connecté (nil si pas
// encore authentifié) et indicateur d'accès invité.
type SessionState struct {
	User  *user.User
	Guest bool
}

// LoggedIn indique si la session est authentifiée ou en accès invité.
func (st *SessionState) LoggedIn() bool { return st != nil && (st.User != nil || st.Guest) }

// Outcome indique au moteur quoi faire après l'exécution d'un applet.
type Outcome struct {
	Done bool // succès → naviguer vers la page Next de la page applet
	Quit bool // terminer la session
}

// AppContext injecte les dépendances accessibles à un applet.
type AppContext struct {
	Users *user.Store   // magasin de comptes (peut être nil)
	State *SessionState // état de la session courante
	Page  *content.Page // page applet courante (titre, intro, Next…)
}

// Applet est une petite unité interactive (login, inscription, jeu…) déclenchée
// par une page de type "applet". Il fait son propre rendu OASCII et sa propre
// saisie ; il ne connaît pas le flux de pages (cf. ADR-0001).
type Applet func(ctx context.Context, s *server.Session, ac *AppContext) Outcome

// applets est le registre nom → applet. Le contenu JSON référence un applet par
// son nom ; ajouter un applet = l'enregistrer ici (ou via Register).
var applets = map[string]Applet{}

// Register enregistre un applet sous un nom. Un nom déjà présent est remplacé.
func Register(name string, a Applet) { applets[name] = a }

// lookupApplet résout un applet par son nom.
func lookupApplet(name string) (Applet, bool) {
	a, ok := applets[name]
	return a, ok
}
