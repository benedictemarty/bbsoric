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
// Applet (optionnel, compat JSON écrit à la main) : à l'arrivée sur la page, on
// lance l'applet nommé puis on va vers Next. Le studio ne crée plus de telles
// pages — les applets se lancent via une entrée de menu (Entry.Applet).
type Page struct {
	Title   string  `json:"title"`
	Lines   []Line  `json:"lines,omitempty"`   // texte (optionnel)
	Entries []Entry `json:"entries,omitempty"` // choix (optionnel → menu)
	Applet  string  `json:"applet,omitempty"`  // applet auto-lancé à l'arrivée (compat)
	Next    string  `json:"next,omitempty"`    // page après succès de l'applet
}

// Line est une ligne de texte avec une couleur d'encre optionnelle.
type Line struct {
	Text string `json:"text"`
	Ink  string `json:"ink,omitempty"` // nom de couleur (défaut blanc)
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
				{Text: " Contenu pilote par un fichier JSON", Ink: "cyan"},
				{Text: " modifiable a chaud.", Ink: "cyan"},
			}},
			"guestbook": {Title: "LIVRE D'OR", Lines: []Line{
				{Text: " (bientot disponible)", Ink: "magenta"},
				{Text: " La messagerie arrive au Sprint 3."},
			}},
		},
	}
}
