// Package bbs contient la logique applicative du BBS Oric : écran d'accueil,
// moteur de menus et écrans de contenu. Le rendu s'appuie sur la couche OASCII
// (attributs Téletexte Oric).
package bbs

import (
	"context"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/datawindow"
	"github.com/benedictemarty/bbsoric/server/internal/files"
	"github.com/benedictemarty/bbsoric/server/internal/presence"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/user"
)

// WelcomeHandler affiche la bannière d'accueil puis déroule le flux de pages
// décrit par le Store (contenu JSON rechargeable à chaud ; contenu par défaut
// si Store est nil). Users est le magasin de comptes injecté aux applets
// (login, inscription…) ; il peut être nil si aucun applet ne l'exige.
type WelcomeHandler struct {
	Store    *content.Store
	Users    *user.Store
	Files    *files.Library     // bibliothèque de fichiers (download/upload ; peut être nil)
	Presence *presence.Registry // registre « qui est en ligne » + chat (peut être nil)
	Data     *datawindow.Engine // moteur DataWindow SQLite (peut être nil)
}

// largeur utile de l'écran TEXT de l'Oric : 40 colonnes.
const oricCols = oascii.Cols

func (h WelcomeHandler) Handle(ctx context.Context, s *server.Session) {
	if err := h.banner(s); err != nil {
		return
	}
	state := &SessionState{Files: h.Files, Presence: h.Presence, Data: h.Data}
	if h.Presence != nil {
		// Pseudo provisoire jusqu'à l'identification (login/invité le fixe).
		state.MemberID = h.Presence.Join("connexion...", s.RemoteIP())
		defer h.Presence.Leave(state.MemberID)
	}
	runSite(ctx, s, h.Store, h.Users, state)
}

// oricArt est l'ASCII-art « ORIC » (5 lignes), affiché centré dans la bannière.
// Construit par glyphes pour garantir une largeur exacte (23 colonnes) qui, une
// fois centrée et précédée d'un octet d'attribut, tient dans les 40 colonnes.
var oricArt = buildOricArt()

// buildOricArt assemble les 4 glyphes O R I C (5×5) en 5 lignes de 23 colonnes.
func buildOricArt() []string {
	glyphs := [][5]string{
		{" ### ", "#   #", "#   #", "#   #", " ### "}, // O
		{"#### ", "#   #", "#### ", "#  # ", "#   #"}, // R
		{"#####", "  #  ", "  #  ", "  #  ", "#####"}, // I
		{" ####", "#    ", "#    ", "#    ", " ####"}, // C
	}
	rows := make([]string, 5)
	for r := range rows {
		parts := make([]string, len(glyphs))
		for g := range glyphs {
			parts[g] = glyphs[g][r]
		}
		rows[r] = strings.Join(parts, " ")
	}
	return rows
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
	b.Text(line).Newline() // blanc (défaut ULA)
	for _, row := range oricArt {
		b.Ink(oascii.Yellow).Text(center(row)).Newline()
	}
	b.Ink(oascii.Cyan).Text(center("B B S   O R I C")).Newline()
	b.Ink(oascii.White).Text(center("bienvenue !")).Newline()
	b.Text(line).Newline() // blanc
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
