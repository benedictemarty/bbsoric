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

	if f.Action != content.FormLogin && f.Action != content.FormRegister {
		writeErr(s, "Action inconnue : "+f.Action)
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

	retries := f.Retries
	if retries <= 0 {
		retries = maxLoginAttempts // défaut
	}

	// Réessai sur place : on redemande les champs jusqu'au succès ou épuisement
	// des tentatives. Échec définitif → Outcome.Failed (le moteur route vers Fail).
	for attempt := 0; attempt < retries; attempt++ {
		vals, canceled, err := readFields(s, f)
		if err != nil {
			return Outcome{Quit: true}
		}
		if canceled {
			return Outcome{} // annulation volontaire (1er champ vide) → retour
		}
		if applyFormAction(s, f.Action, vals, ac) {
			return Outcome{Done: true}
		}
		// échec : message déjà affiché ; on reboucle s'il reste des tentatives.
	}
	return Outcome{Failed: true}
}

// readFields saisit les champs du formulaire dans l'ordre. canceled vaut true si
// le premier champ est laissé vide (annulation). err signale une coupure d'E/S.
func readFields(s *server.Session, f *content.Form) (vals map[string]string, canceled bool, err error) {
	vals = make(map[string]string, len(f.Fields))
	for i, fld := range f.Fields {
		if fld.Secret {
			warnCleartext(s)
		}
		if len(fld.At) == 2 { // positionnement absolu optionnel (plot X,Y)
			if err = s.Write(oascii.Plot(fld.At[0], fld.At[1])); err != nil {
				return nil, false, err
			}
		}
		v, e := prompt(s, fld.Label)
		if e != nil {
			return nil, false, e
		}
		if i == 0 && strings.TrimSpace(v) == "" {
			return nil, true, nil
		}
		vals[fld.Key] = v
	}
	return vals, false, nil
}

// applyFormAction exécute l'action et renvoie true en cas de succès. En cas
// d'échec, affiche le message d'erreur et renvoie false (à reboucler).
func applyFormAction(s *server.Session, action string, vals map[string]string, ac *AppContext) bool {
	switch action {
	case content.FormLogin:
		if loginThrottled(s, ac.State) {
			return false
		}
		u, err := ac.Users.Authenticate(vals["login"], vals["password"])
		if err != nil {
			recordLoginFailure(ac.State)
			writeErr(s, "Echec : "+err.Error())
			return false
		}
		ac.State.User = &u
		setPresenceHandle(ac.State, u.Handle)
		recordLoginSuccess(ac.State)
		greet(s, u)
		return true

	case content.FormRegister:
		if vals["password"] != vals["confirm"] {
			writeErr(s, "Les mots de passe different.")
			return false
		}
		u, err := ac.Users.Register(vals["login"], vals["password"])
		if err != nil {
			writeErr(s, "Echec : "+err.Error())
			return false
		}
		ac.State.User = &u
		setPresenceHandle(ac.State, u.Handle)
		b := oascii.New()
		b.Newline().Ink(oascii.Green).Text(center("Compte cree !")).Newline()
		b.Ink(oascii.Yellow).Text(center("Bienvenue " + u.Handle + " !")).Newline()
		b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
		_ = s.Write(b.String())
		_, _ = s.ReadKey()
		return true
	}
	return false
}
