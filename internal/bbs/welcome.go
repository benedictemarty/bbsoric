// Package bbs contient la logique applicative du BBS Oric : écrans, menus et,
// à terme, le moteur de session complet. Pour le Sprint 0, il fournit un
// écran d'accueil « hello world » qui valide la chaîne réseau de bout en bout.
package bbs

import (
	"context"
	"io"
	"strings"

	"github.com/bmarty/bbsoric/internal/server"
)

// WelcomeHandler affiche un écran d'accueil puis une boucle d'écho minimale
// (commande QUIT pour se déconnecter). Le rendu reste en ASCII pur : la couche
// OASCII (couleurs/attributs Téletexte Oric) est l'objet du Sprint 1.
type WelcomeHandler struct{}

// largeur utile de l'écran TEXT de l'Oric : 40 colonnes.
const oricCols = 40

func (h WelcomeHandler) Handle(ctx context.Context, s *server.Session) {
	if err := h.banner(s); err != nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.Write("> "); err != nil {
			return
		}
		line, err := s.ReadLine()
		if err != nil {
			if err != io.EOF {
				// timeout d'inactivité ou erreur réseau : on quitte proprement
			}
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch cmd {
		case "":
			continue
		case "QUIT", "BYE", "EXIT":
			_ = s.Println("Au revoir !")
			return
		case "HELP":
			_ = s.Println("Commandes : HELP, QUIT")
		default:
			_ = s.Println("Vous avez dit : " + line)
		}
	}
}

func (h WelcomeHandler) banner(s *server.Session) error {
	line := strings.Repeat("=", oricCols)
	if err := s.Println(line); err != nil {
		return err
	}
	if err := s.Println(center("B B S   O R I C")); err != nil {
		return err
	}
	if err := s.Println(center("bienvenue !")); err != nil {
		return err
	}
	if err := s.Println(line); err != nil {
		return err
	}
	if err := s.Println(""); err != nil {
		return err
	}
	if err := s.Println("Serveur en ligne (Sprint 0 - hello world)."); err != nil {
		return err
	}
	return s.Println("Tapez HELP pour l'aide, QUIT pour quitter.")
}

// center centre un texte sur la largeur de l'écran Oric (40 colonnes).
func center(text string) string {
	if len(text) >= oricCols {
		return text
	}
	pad := (oricCols - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}
