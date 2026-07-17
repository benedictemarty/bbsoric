package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/bbsoric/server/internal/wall"
)

// wallMaxShown borne le nombre de messages affichés à l'écran (les plus récents).
const wallMaxShown = 8

// init enregistre l'applet du mur de messages.
func init() {
	Register("wall", wallApplet)
}

// wallApplet affiche le mur de messages (les plus récents en tête) et permet à
// l'appelant d'y laisser un message. C'est le premier applet à écriture
// persistée : la saisie (ReadLine) est publiée dans le store wall, borné et
// nettoyé côté serveur.
func wallApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	store := ac.State.Wall
	for {
		header(s, "MUR DE MESSAGES")
		if store == nil {
			writeErr(s, "Mur indisponible.")
			anyKey(s)
			return Outcome{}
		}
		if err := writeWallMessages(s, store); err != nil {
			return Outcome{Quit: true}
		}
		text, err := prompt(s, "Message (vide=retour)")
		if err != nil {
			return Outcome{Quit: true}
		}
		if strings.TrimSpace(text) == "" {
			return Outcome{} // retour au menu appelant
		}
		if _, err := store.Post(ac.State.displayName(), text); err != nil {
			writeErr(s, "Refuse : "+err.Error())
			anyKey(s)
		}
		// On reboucle : le nouveau message apparaît en tête.
	}
}

// writeWallMessages rend les derniers messages du mur (antéchronologique).
func writeWallMessages(s *server.Session, store *wall.Store) error {
	list := store.List(wallMaxShown)
	b := oascii.New()
	if len(list) == 0 {
		b.Ink(oascii.Magenta).Text(" Mur vide : soyez le premier a signer !").Newline().Newline()
		return s.Write(b.String())
	}
	b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %d message(s) :", store.Count())).Newline().Newline()
	for _, m := range list {
		b.Ink(oascii.Yellow).Text(" " + trunc(m.Handle, 16))
		b.Ink(oascii.White).Text(" " + m.At.Format("02/01 15:04")).Newline()
		for _, seg := range wrapText(m.Text, oascii.Cols-2) {
			b.Ink(oascii.Green).Text(" " + seg).Newline()
		}
	}
	b.Newline()
	return s.Write(b.String())
}

// wrapText découpe un texte en segments d'au plus width caractères, sans couper
// les mots quand c'est possible. Un mot plus long que width est coupé net.
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var out []string
	var line string
	for _, word := range strings.Fields(text) {
		for len(word) > width { // mot trop long : coupé en tranches
			if line != "" {
				out = append(out, line)
				line = ""
			}
			out = append(out, word[:width])
			word = word[width:]
		}
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= width:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
