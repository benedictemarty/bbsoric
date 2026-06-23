// Package content modélise le contenu et le flux de pages du BBS sous forme de
// données (chargeables depuis un fichier JSON, rechargeable à chaud), pour que
// l'enchaînement des écrans soit modifiable sans recompiler ni redémarrer.
package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Cibles spéciales de navigation (au lieu d'un identifiant de page).
const (
	TargetQuit = "__quit__" // termine la session
	TargetBack = "__back__" // page précédente (dépile)
	TargetHome = "__home__" // page de départ
)

// Site est l'ensemble du contenu navigable.
type Site struct {
	Start string           `json:"start"` // identifiant de la page de départ
	Pages map[string]*Page `json:"pages"` // pages indexées par identifiant
}

// Page est un écran du BBS. Type unique : elle affiche optionnellement du texte
// (Lines) et/ou des choix (Entries).
//   - avec Entries  → écran interactif (menu) : une touche route vers la cible
//     de l'entrée (ou lance son applet) ;
//   - sans Entries  → écran de contenu : une touche revient en arrière.
//
// Raw + Screen : « écran brut » rendu tel quel (buffer 40×28 composé case par
// case dans le studio), sans barre de titre ni invite. Raw se COMBINE avec
// Entries : le buffer sert alors de FOND d'écran (décor, libellés du menu
// dessinés dedans) tandis que les Entries assurent la navigation (touche →
// cible, ou ▶ applet). C'est le « menu sur fond d'écran » : présentation
// (Screen) et logique (Entries) sont séparées.
//
// Applet (optionnel, compat JSON écrit à la main) : à l'arrivée sur la page, on
// lance l'applet nommé puis on va vers Next. Le studio ne crée plus de telles
// pages — les applets se lancent via une entrée de menu (Entry.Applet).
type Page struct {
	Title   string  `json:"title"`
	Lines   []Line  `json:"lines,omitempty"`   // texte (optionnel)
	Entries []Entry `json:"entries,omitempty"` // choix (optionnel → menu)
	Applet  string  `json:"applet,omitempty"`  // applet auto-lancé à l'arrivée (compat)
	Next    string  `json:"next,omitempty"`    // page après succès de l'applet
	Raw     bool    `json:"raw,omitempty"`     // écran brut (rendu tel quel)
	Screen  []byte  `json:"screen,omitempty"`  // écran brut : buffer 40×28 d'octets (base64 JSON)
	Form    *Form   `json:"form,omitempty"`    // page de saisie déclarative (login/inscription)
}

// Form décrit un écran de saisie déclaratif, exécuté par l'applet générique
// « form » : on saisit les Fields dans l'ordre, puis l'Action (logique en Go,
// jeu fermé) est appliquée. Succès → navigation vers Next (ou Page.Next).
//
// La présentation reste séparée : si la page est Raw, son buffer sert de décor
// (les invites de saisie s'affichent ensuite, séquentiellement) ; sinon un
// bandeau de titre est affiché.
type Form struct {
	Action string  `json:"action"`           // "login" | "register"
	Fields []Field `json:"fields,omitempty"` // champs saisis dans l'ordre
	Next   string  `json:"next,omitempty"`   // page après succès (sinon Page.Next)
}

// Field est un champ de saisie d'un Form. Key identifie la valeur pour l'action
// (clés attendues : "login", "password", "confirm") ; Secret avertit que la
// saisie est visible à l'écran (l'Oric ne masque pas la frappe). At, s'il est
// présent, positionne l'invite à des coordonnées absolues [col, row] (plot X,Y) ;
// sinon les invites s'affichent séquentiellement.
type Field struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Secret bool   `json:"secret,omitempty"`
	At     []int  `json:"at,omitempty"` // [col, row] optionnel (positionnement)
}

// Actions de formulaire reconnues (logique exécutée en Go).
const (
	FormLogin    = "login"
	FormRegister = "register"
)

// Style regroupe les attributs sériels Oric (Téletexte). Chacun est optionnel ;
// non renseigné = valeur par défaut (encre blanche, fond noir, pas d'effet).
type Style struct {
	Ink          string `json:"ink,omitempty"`          // encre (couleur du texte)
	Paper        string `json:"paper,omitempty"`        // fond
	Blink        bool   `json:"blink,omitempty"`        // clignotement
	DoubleHeight bool   `json:"doubleHeight,omitempty"` // double hauteur
	AltCharset   bool   `json:"altCharset,omitempty"`   // charset alternatif (semi-graphiques)
	Inverse      bool   `json:"inverse,omitempty"`      // vidéo inverse
}

// Span est un fragment de texte stylé (pour le multicolore sur une même ligne).
type Span struct {
	Text  string `json:"text"`
	Style        // attributs promus au niveau du fragment
}

// Line est une ligne d'écran. Soit du texte simple (Text + Style), soit une
// suite de fragments stylés (Segments) pour mêler plusieurs styles sur la ligne.
type Line struct {
	Text     string `json:"text,omitempty"`
	Style           // attributs promus (ligne simple)
	Segments []Span `json:"segments,omitempty"`
}

// Entry est un choix de menu : une touche qui, soit navigue vers une cible
// (Target = page ou cible spéciale), soit lance un applet (Applet), auquel cas
// Next est la page où aller après succès (vide = on reste sur le menu). Un même
// menu peut ainsi proposer plusieurs applets au choix.
type Entry struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Target string `json:"target,omitempty"` // identifiant de page ou cible spéciale
	Applet string `json:"applet,omitempty"` // applet à lancer (au lieu de naviguer)
	Next   string `json:"next,omitempty"`   // page après succès de l'applet
}

// Parse décode et valide un Site depuis du JSON.
func Parse(b []byte) (*Site, error) {
	var s Site
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("JSON invalide : %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate vérifie la cohérence (page de départ, cibles existantes, types).
func (s *Site) Validate() error {
	if len(s.Pages) == 0 {
		return fmt.Errorf("aucune page définie")
	}
	if s.Start == "" {
		return fmt.Errorf("champ 'start' manquant")
	}
	if _, ok := s.Pages[s.Start]; !ok {
		return fmt.Errorf("page de départ %q introuvable", s.Start)
	}
	for id, p := range s.Pages {
		// Page applet (auto-lancé) : Next, si présent, doit désigner une page.
		if p.Applet != "" && p.Next != "" {
			if _, ok := s.Pages[p.Next]; !ok {
				return fmt.Errorf("page %q : 'next' %q introuvable", id, p.Next)
			}
		}
		for _, e := range p.Entries {
			if e.Applet != "" {
				// Entrée-applet : Next (si présent) doit désigner une page.
				if e.Next != "" {
					if _, ok := s.Pages[e.Next]; !ok {
						return fmt.Errorf("page %q : 'next' %q introuvable", id, e.Next)
					}
				}
				continue
			}
			if e.Target == "" {
				return fmt.Errorf("page %q : entrée %q sans 'target' ni 'applet'", id, e.Key)
			}
			if isSpecialTarget(e.Target) {
				continue
			}
			if _, ok := s.Pages[e.Target]; !ok {
				return fmt.Errorf("page %q : cible %q introuvable", id, e.Target)
			}
		}
		if p.Form != nil {
			if err := p.Form.validate(id, s); err != nil {
				return err
			}
		}
	}
	return nil
}

// validate vérifie un Form : action connue, champs requis présents, Next existant.
func (f *Form) validate(pageID string, s *Site) error {
	hasField := func(key string) bool {
		for _, fld := range f.Fields {
			if fld.Key == key {
				return true
			}
		}
		return false
	}
	switch f.Action {
	case FormLogin, FormRegister:
		if !hasField("login") || !hasField("password") {
			return fmt.Errorf("page %q : form %q exige les champs 'login' et 'password'", pageID, f.Action)
		}
		if f.Action == FormRegister && !hasField("confirm") {
			return fmt.Errorf("page %q : form 'register' exige un champ 'confirm'", pageID)
		}
	case "":
		return fmt.Errorf("page %q : form sans 'action'", pageID)
	default:
		return fmt.Errorf("page %q : action de form inconnue %q", pageID, f.Action)
	}
	for _, fld := range f.Fields {
		if len(fld.At) != 0 {
			if len(fld.At) != 2 {
				return fmt.Errorf("page %q : champ %q : 'at' doit être [col, row]", pageID, fld.Key)
			}
			if fld.At[0] < 0 || fld.At[0] >= oascii.Cols || fld.At[1] < 0 || fld.At[1] >= oascii.Rows {
				return fmt.Errorf("page %q : champ %q : 'at' hors écran (%d×%d)", pageID, fld.Key, oascii.Cols, oascii.Rows)
			}
		}
	}
	if f.Next != "" {
		if _, ok := s.Pages[f.Next]; !ok {
			return fmt.Errorf("page %q : form 'next' %q introuvable", pageID, f.Next)
		}
	}
	return nil
}

func isSpecialTarget(t string) bool {
	return t == TargetQuit || t == TargetBack || t == TargetHome
}

var colorByName = map[string]oascii.Color{
	"black": oascii.Black, "red": oascii.Red, "green": oascii.Green,
	"yellow": oascii.Yellow, "blue": oascii.Blue, "magenta": oascii.Magenta,
	"cyan": oascii.Cyan, "white": oascii.White,
}

// Ink convertit un nom de couleur en couleur OASCII (blanc par défaut).
func Ink(name string) oascii.Color {
	if c, ok := colorByName[strings.ToLower(name)]; ok {
		return c
	}
	return oascii.White
}

// DefaultSite renvoie le contenu intégré par défaut (utilisé si aucun fichier
// JSON n'est fourni). Reproduit le menu historique du BBS.
func DefaultSite() *Site {
	return &Site{
		Start: "main",
		Pages: map[string]*Page{
			"main": {Title: "MENU PRINCIPAL", Entries: []Entry{
				{Key: "1", Label: "Informations systeme", Target: "info"},
				{Key: "2", Label: "A propos du BBS", Target: "about"},
				{Key: "3", Label: "Livre d'or", Target: "guestbook"},
				{Key: "Q", Label: "Quitter", Target: TargetQuit},
			}},
			"info": {Title: "INFORMATIONS SYSTEME", Lines: []Line{
				{Text: " Serveur  - BBS Oric (Go)"},
				{Text: " Ecran    - TEXT 40x28, OASCII"},
				{Text: " Port     - 6502 (telnet) / 6992 (TLS)"},
				{Text: " Encodage - ASCII + attributs Teletexte"},
			}},
			"about": {Title: "A PROPOS", Lines: []Line{
				{Text: " BBS pour ordinateurs Oric, dans"},
				{Text: " l'esprit des serveurs retro type"},
				{Text: " PETSCII BBS / ATASCII."},
				{Text: ""},
				{Text: " Contenu pilote par un fichier JSON", Style: Style{Ink: "cyan"}},
				{Text: " modifiable a chaud.", Style: Style{Ink: "cyan"}},
			}},
			"guestbook": {Title: "LIVRE D'OR", Lines: []Line{
				{Text: " (bientot disponible)", Style: Style{Ink: "magenta"}},
				{Text: " La messagerie arrive au Sprint 3."},
			}},
		},
	}
}
