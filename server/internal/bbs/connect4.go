package bbs

import (
	"context"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/connect4"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre l'applet du jeu Puissance 4.
func init() {
	Register("connect4", connect4Applet)
}

// connect4Applet est une partie de Puissance 4 contre l'ordinateur. Le joueur
// dépose un jeton avec les touches 1..7 ; l'ordinateur répond (heuristique
// server/internal/connect4). Valide l'interactivité tour par tour sur lien série.
func connect4Applet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	g := connect4.New()
	msg := "A vous : colonne 1-7 (Q=quitter)."
	for {
		if err := writeBoard(s, g, msg); err != nil {
			return Outcome{Quit: true}
		}
		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		if upperKey(key) == 'Q' {
			return Outcome{}
		}
		col := int(key) - '1'
		if col < 0 || col >= connect4.Cols {
			msg = "Touche invalide : tapez 1 a 7."
			continue
		}
		if g.ColumnFull(col) {
			msg = "Colonne pleine, choisissez-en une autre."
			continue
		}
		g.Drop(col, connect4.Player)
		if g.Winner() == connect4.Player {
			return endGame(s, g, "GAGNE ! Bravo, 4 alignes.")
		}
		if g.Full() {
			return endGame(s, g, "Match nul : plateau plein.")
		}
		// Coup de l'ordinateur.
		cc := g.ComputerMove()
		if cc >= 0 {
			g.Drop(cc, connect4.Computer)
		}
		if g.Winner() == connect4.Computer {
			return endGame(s, g, "PERDU : l'ordinateur aligne 4.")
		}
		if g.Full() {
			return endGame(s, g, "Match nul : plateau plein.")
		}
		msg = "Ordi a joue colonne " + string(rune('1'+cc)) + ". A vous 1-7."
	}
}

// endGame affiche le plateau final + le résultat, attend une touche, et rend la
// main au menu appelant.
func endGame(s *server.Session, g *connect4.Game, result string) Outcome {
	_ = writeBoard(s, g, result+" Appuyez sur une touche.")
	anyKey(s)
	return Outcome{}
}

// writeBoard rend le plateau en OASCII (bas du plateau = ligne du bas de
// l'écran) avec les jetons colorés, plus la ligne d'état.
func writeBoard(s *server.Session, g *connect4.Game, msg string) error {
	header(s, "PUISSANCE 4")
	b := oascii.New()
	b.Ink(oascii.Cyan).Text(" 1 2 3 4 5 6 7").Newline()
	for r := connect4.Rows - 1; r >= 0; r-- {
		b.Text(" ")
		for c := 0; c < connect4.Cols; c++ {
			switch g.At(r, c) {
			case connect4.Player:
				b.Ink(oascii.Red).Text("O")
			case connect4.Computer:
				b.Ink(oascii.Yellow).Text("X")
			default:
				b.Ink(oascii.Blue).Text(".")
			}
			b.Ink(oascii.White).Text(" ")
		}
		b.Newline()
	}
	b.Newline()
	b.Ink(oascii.Green).Text(" Vous=").Ink(oascii.Red).Text("O")
	b.Ink(oascii.Green).Text("  Ordi=").Ink(oascii.Yellow).Text("X").Newline()
	b.Ink(oascii.White).Text(" " + msg).Newline()
	return s.Write(b.String())
}
