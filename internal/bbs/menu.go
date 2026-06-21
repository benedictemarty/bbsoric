package bbs

import (
	"strings"

	"github.com/bmarty/bbsoric/internal/oascii"
	"github.com/bmarty/bbsoric/internal/server"
)

// Version applicative affichée par le BBS (suit le sprint courant).
const bbsVersion = "Sprint 2"

// firstKey renvoie le premier caractère significatif d'une ligne, en majuscule
// (0 si la ligne est vide). Sert à router les choix de menu.
func firstKey(line string) byte {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}
	c := line[0]
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return c
}

// rule trace une règle pleine largeur (40 col) en couleur par défaut.
func rule() string { return strings.Repeat("=", oascii.Cols) }

// mainMenu construit l'écran du menu principal (en couleurs OASCII).
func mainMenu() string {
	b := oascii.New()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center("MENU PRINCIPAL")).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	b.Ink(oascii.Cyan).Text(" 1").Ink(oascii.White).Text(" - Informations systeme").Newline()
	b.Ink(oascii.Cyan).Text(" 2").Ink(oascii.White).Text(" - A propos du BBS").Newline()
	b.Ink(oascii.Cyan).Text(" 3").Ink(oascii.White).Text(" - Livre d'or").Newline()
	b.Ink(oascii.Cyan).Text(" Q").Ink(oascii.White).Text(" - Quitter").Newline()
	b.Newline()
	return b.String()
}

// menuLoop affiche le menu principal et route les choix jusqu'à la sortie.
func menuLoop(s *server.Session) {
	for {
		if err := s.Write(mainMenu()); err != nil {
			return
		}
		if err := s.Write(makeInk(oascii.Green) + "Votre choix" + makeInk(oascii.White) + "> "); err != nil {
			return
		}
		line, err := s.ReadLine()
		if err != nil {
			return
		}
		switch firstKey(line) {
		case 0:
			continue
		case '1':
			if screenInfo(s) != nil {
				return
			}
		case '2':
			if screenAbout(s) != nil {
				return
			}
		case '3':
			if screenGuestbook(s) != nil {
				return
			}
		case 'Q':
			b := oascii.New()
			b.Newline().Ink(oascii.Yellow).Text(center("A bientot sur le BBS Oric !")).Newline()
			_ = s.Write(b.String())
			return
		default:
			b := oascii.New()
			b.Ink(oascii.Red).Text("Choix invalide.").Newline()
			if s.Write(b.String()) != nil {
				return
			}
		}
	}
}

// makeInk renvoie l'octet d'attribut d'encre sous forme de chaîne (1 caractère).
func makeInk(c oascii.Color) string { return string([]byte{oascii.InkAttr(c)}) }

// page affiche un écran de contenu (titre + corps) puis attend RETURN.
// Renvoie une erreur si la connexion est rompue (pour propager la sortie).
func page(s *server.Session, title string, body func(*oascii.Builder)) error {
	b := oascii.New()
	b.Newline()
	b.Text(rule()).Newline()
	b.Ink(oascii.Yellow).Text(center(title)).Newline()
	b.Text(rule()).Newline()
	b.Newline()
	body(b)
	b.Newline()
	b.Ink(oascii.Green).Text("[RETURN] retour au menu").Newline()
	if err := s.Write(b.String()); err != nil {
		return err
	}
	_, err := s.ReadLine() // attend une ligne (RETURN) pour revenir
	return err
}

func screenInfo(s *server.Session) error {
	return page(s, "INFORMATIONS SYSTEME", func(b *oascii.Builder) {
		b.Ink(oascii.White)
		b.Text(" Serveur  - BBS Oric (Go)").Newline()
		b.Text(" Version  - " + bbsVersion).Newline()
		b.Text(" Ecran    - TEXT 40x28, OASCII").Newline()
		b.Text(" Port     - 6502").Newline()
		b.Text(" Encodage - ASCII + attributs Teletexte").Newline()
	})
}

func screenAbout(s *server.Session) error {
	return page(s, "A PROPOS", func(b *oascii.Builder) {
		b.Ink(oascii.White)
		b.Text(" BBS pour ordinateurs Oric, dans").Newline()
		b.Text(" l'esprit des serveurs retro type").Newline()
		b.Text(" PETSCII BBS / ATASCII.").Newline()
		b.Newline()
		b.Text(" Le serveur tourne sur une machine").Newline()
		b.Text(" moderne ; l'Oric se connecte via").Newline()
		b.Text(" un modem WiFi (ACIA serie).").Newline()
	})
}

func screenGuestbook(s *server.Session) error {
	return page(s, "LIVRE D'OR", func(b *oascii.Builder) {
		b.Ink(oascii.Magenta).Text(" (bientot disponible)").Newline()
		b.Ink(oascii.White).Text(" La messagerie arrive au Sprint 3.").Newline()
	})
}
