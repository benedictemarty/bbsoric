package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// maxLoginAttempts borne le nombre d'essais d'identification par passage.
const maxLoginAttempts = 3

// init enregistre les applets d'authentification. Une page JSON les référence
// par leur nom : {"type":"applet","applet":"login","next":"main"}.
func init() {
	Register("login", loginApplet)
	Register("register", registerApplet)
	Register("guest", guestApplet)
}

// loginApplet identifie un utilisateur existant (pseudo + mot de passe). Succès
// → State.User renseigné et navigation vers la page "next".
func loginApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "IDENTIFICATION")
	if ac.Users == nil {
		writeErr(s, "Comptes indisponibles.")
		return Outcome{}
	}
	for attempt := 1; attempt <= maxLoginAttempts; attempt++ {
		handle, err := prompt(s, "Pseudo (vide=annuler)")
		if err != nil {
			return Outcome{Quit: true}
		}
		if strings.TrimSpace(handle) == "" {
			return Outcome{} // annulation → retour au menu appelant
		}
		warnCleartext(s)
		pass, err := prompt(s, "Mot de passe")
		if err != nil {
			return Outcome{Quit: true}
		}
		u, err := ac.Users.Authenticate(handle, pass)
		if err != nil {
			writeErr(s, "Echec : "+err.Error())
			continue
		}
		ac.State.User = &u
		setPresenceHandle(ac.State, u.Handle)
		greet(s, u)
		return Outcome{Done: true}
	}
	writeErr(s, "Trop de tentatives.")
	return Outcome{Failed: true}
}

// registerApplet crée un nouveau compte (pseudo + mot de passe + confirmation).
func registerApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "CREATION DE COMPTE")
	if ac.Users == nil {
		writeErr(s, "Comptes indisponibles.")
		return Outcome{}
	}
	for attempt := 1; attempt <= maxLoginAttempts; attempt++ {
		handle, err := prompt(s, "Pseudo desire (vide=annuler)")
		if err != nil {
			return Outcome{Quit: true}
		}
		if strings.TrimSpace(handle) == "" {
			return Outcome{}
		}
		warnCleartext(s)
		pass, err := prompt(s, "Mot de passe")
		if err != nil {
			return Outcome{Quit: true}
		}
		confirm, err := prompt(s, "Confirmer")
		if err != nil {
			return Outcome{Quit: true}
		}
		if pass != confirm {
			writeErr(s, "Les mots de passe different.")
			continue
		}
		u, err := ac.Users.Register(handle, pass)
		if err != nil {
			writeErr(s, "Echec : "+err.Error())
			continue
		}
		ac.State.User = &u
		setPresenceHandle(ac.State, u.Handle)
		b := oascii.New()
		b.Newline().Ink(oascii.Green).Text(center("Compte cree !")).Newline()
		b.Ink(oascii.Yellow).Text(center("Bienvenue " + u.Handle + " !")).Newline()
		b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
		_ = s.Write(b.String())
		_, _ = s.ReadKey()
		return Outcome{Done: true}
	}
	writeErr(s, "Creation abandonnee.")
	return Outcome{Failed: true}
}

// guestApplet accorde un accès invité (lecture seule), sans compte.
func guestApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "ACCES INVITE")
	ac.State.Guest = true
	setPresenceHandle(ac.State, fmt.Sprintf("Invite-%d", ac.State.MemberID))
	b := oascii.New()
	b.Ink(oascii.White).Text(" Bienvenue, visiteur.").Newline()
	b.Ink(oascii.Cyan).Text(" Acces en lecture seule.").Newline()
	b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	if s.Write(b.String()) != nil {
		return Outcome{Quit: true}
	}
	_, _ = s.ReadKey()
	return Outcome{Done: true}
}

// setPresenceHandle fixe le pseudo affiché de la session et le propage au
// registre de présence (« qui est en ligne » + chat), si présent.
func setPresenceHandle(st *SessionState, handle string) {
	if st == nil {
		return
	}
	st.Handle = handle
	if st.Presence != nil {
		st.Presence.SetHandle(st.MemberID, handle)
	}
}

// --- helpers de présentation/saisie ---

// header affiche un bandeau de titre centré.
func header(s *server.Session, title string) {
	b := oascii.New()
	b.Newline()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	_ = s.Write(b.String())
}

// prompt affiche une invite verte et lit une ligne (saisie texte, RETURN).
func prompt(s *server.Session, label string) (string, error) {
	if err := s.Write(makeInk(oascii.Green) + label + makeInk(oascii.White) + "> "); err != nil {
		return "", err
	}
	return s.ReadLine()
}

// warnCleartext rappelle (une fois par saisie) que le mot de passe s'affiche.
func warnCleartext(s *server.Session) {
	b := oascii.New()
	b.Ink(oascii.Red).Text(" (saisie visible a l'ecran)").Newline()
	_ = s.Write(b.String())
}

// writeErr affiche un message d'erreur en rouge.
func writeErr(s *server.Session, msg string) {
	b := oascii.New()
	b.Ink(oascii.Red).Text(" " + msg).Newline()
	_ = s.Write(b.String())
}

// greet affiche l'accueil personnalisé (pseudo + numéro d'appel) et marque une
// pause, à la manière des BBS historiques.
func greet(s *server.Session, u user.User) {
	b := oascii.New()
	b.Newline()
	b.Ink(oascii.Yellow).Text(center("Bonjour " + u.Handle + " !")).Newline()
	b.Ink(oascii.Cyan).Text(center(fmt.Sprintf("Appel n%d", u.Calls))).Newline()
	b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	_ = s.Write(b.String())
	_, _ = s.ReadKey()
}
