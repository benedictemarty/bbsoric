package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/xmodem"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre les applets de transfert de fichiers (XMODEM).
func init() {
	Register("download", downloadApplet)
	Register("upload", uploadApplet)
}

// downloadApplet liste la bibliothèque, laisse choisir un fichier (par numéro)
// puis l'envoie au client via XMODEM. Le client doit lancer une réception XMODEM.
func downloadApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "TELECHARGEMENT")
	if ac.Files == nil {
		writeErr(s, "Bibliotheque indisponible.")
		return Outcome{}
	}
	list, err := ac.Files.List()
	if err != nil {
		writeErr(s, "Lecture impossible : "+err.Error())
		return Outcome{}
	}
	if len(list) == 0 {
		writeErr(s, "Aucun fichier disponible.")
		return Outcome{}
	}

	b := oascii.New()
	for i, f := range list {
		if i >= 9 {
			break // choix par chiffre 1..9
		}
		b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %d ", i+1))
		b.Ink(oascii.White).Text(fmt.Sprintf("%-20s %5do", f.Name, f.Size)).Newline()
	}
	b.Newline().Ink(oascii.Green).Text("Fichier (1-" + itoa(min(len(list), 9)) + ", autre=annuler) > ")
	b.Ink(oascii.White).Text("")
	if s.Write(b.String()) != nil {
		return Outcome{Quit: true}
	}

	key, err := s.ReadKey()
	if err != nil {
		return Outcome{Quit: true}
	}
	idx := int(key - '1')
	if idx < 0 || idx >= len(list) || idx >= 9 {
		return Outcome{} // annulation
	}
	f := list[idx]
	data, err := ac.Files.Read(f.Name)
	if err != nil {
		writeErr(s, "Lecture impossible : "+err.Error())
		return Outcome{}
	}

	info := oascii.New()
	info.Newline().Ink(oascii.Yellow).Text("Envoi de " + f.Name + " (XMODEM)...").Newline()
	info.Ink(oascii.White).Text("Demarrez la reception sur votre terminal.").Newline()
	_ = s.Write(info.String())

	// Signale au terminal Oric de basculer en réception XMODEM (les autres
	// clients ignorent cette séquence de contrôle), suivi du nombre total de
	// blocs de 128 o (octet bas, octet haut) pour la jauge de progression du
	// terminal. Un terminal plus ancien ignore ces 2 octets (non-SOH).
	_ = s.Write(oascii.RecvCmd())
	totalBlocks := (len(data) + 127) / 128
	_ = s.Write(string([]byte{byte(totalBlocks & 0xFF), byte((totalBlocks >> 8) & 0xFF)}))

	if err := xmodem.Send(s.Raw(), data); err != nil {
		s.ClearDeadline()
		writeErr(s, "Transfert echoue : "+err.Error())
		return Outcome{}
	}
	s.ClearDeadline()
	okMsg := oascii.New()
	okMsg.Newline().Ink(oascii.Green).Text("Telechargement termine.").Newline()
	okMsg.Text("Appuyez sur une touche...").Newline()
	_ = s.Write(okMsg.String())
	_, _ = s.ReadKey()
	return Outcome{Done: true}
}

// uploadApplet reçoit un fichier du client via XMODEM et l'enregistre dans la
// bibliothèque sous le nom saisi.
func uploadApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "TELEVERSEMENT")
	if ac.Files == nil {
		writeErr(s, "Bibliotheque indisponible.")
		return Outcome{}
	}
	name, err := prompt(s, "Nom du fichier (vide=annuler)")
	if err != nil {
		return Outcome{Quit: true}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Outcome{}
	}

	info := oascii.New()
	info.Newline().Ink(oascii.Yellow).Text("Pret a recevoir " + name + " (XMODEM)...").Newline()
	info.Ink(oascii.White).Text("Demarrez l'envoi sur votre terminal.").Newline()
	_ = s.Write(info.String())

	// Signale au terminal Oric de basculer en envoi XMODEM.
	_ = s.Write(oascii.SendCmd())

	data, err := xmodem.Receive(s.Raw())
	if err != nil {
		s.ClearDeadline()
		writeErr(s, "Transfert echoue : "+err.Error())
		return Outcome{}
	}
	s.ClearDeadline()
	if err := ac.Files.Write(name, data); err != nil {
		writeErr(s, "Enregistrement impossible : "+err.Error())
		return Outcome{}
	}
	okMsg := oascii.New()
	okMsg.Newline().Ink(oascii.Green).Text(fmt.Sprintf("Recu : %s (%d octets).", name, len(data))).Newline()
	okMsg.Text("Appuyez sur une touche...").Newline()
	_ = s.Write(okMsg.String())
	_, _ = s.ReadKey()
	return Outcome{Done: true}
}

// itoa convertit un petit entier positif (0..9) en chaîne.
func itoa(n int) string { return string(rune('0' + n)) }
