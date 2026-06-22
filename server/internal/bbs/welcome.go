// Package bbs contient la logique applicative du BBS Oric : écran d'accueil,
// moteur de menus et écrans de contenu. Le rendu s'appuie sur la couche OASCII
// (attributs Téletexte Oric).
package bbs

import (
	"context"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// WelcomeHandler affiche la bannière d'accueil puis déroule le flux de pages
// décrit par le Store (contenu JSON rechargeable à chaud ; contenu par défaut
// si Store est nil). Users est le magasin de comptes injecté aux applets
// (login, inscription…) ; il peut être nil si aucun applet ne l'exige.
type WelcomeHandler struct {
	Store *content.Store
	Users *user.Store
}

// largeur utile de l'écran TEXT de l'Oric : 40 colonnes.
const oricCols = oascii.Cols

func (h WelcomeHandler) Handle(ctx context.Context, s *server.Session) {
	if err := h.banner(s); err != nil {
		return
	}
	runSite(ctx, s, h.Store, h.Users, &SessionState{})
}

// banner construit l'écran d'accueil avec attributs OASCII (couleurs Oric).
//
// Note Oric : un octet d'attribut occupe une case écran. Les lignes pleine
// largeur (40 « = ») restent donc en couleur par défaut (blanc/noir, aucun
// octet d'attribut émis) pour tenir exactement sur 40 colonnes ; seules les
// lignes centrées, qui disposent d'une marge, sont colorées.
func (h WelcomeHandler) banner(s *server.Session) error {
	line := strings.Repeat("=", oricCols)
	b := oascii.New()
	b.Text(line).Newline()                                       // blanc (défaut ULA)
	b.Ink(oascii.Yellow).Text(center("B B S   O R I C")).Newline()
	b.Ink(oascii.Cyan).Text(center("bienvenue !")).Newline()
	b.Text(line).Newline()                                       // blanc
	b.Newline()
	b.Ink(oascii.Green).Text("Serveur en ligne (" + bbsVersion + ").").Newline()
	return s.Write(b.String())
}

// center centre un texte sur la largeur de l'écran Oric (40 colonnes).
func center(text string) string {
	if len(text) >= oricCols {
		return text
	}
	pad := (oricCols - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}
