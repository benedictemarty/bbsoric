package bbs

import (
	"context"
	"fmt"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/xmodem"
	"github.com/benedictemarty/bbsoric/server/internal/files"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre l'applet de l'espace fichiers personnel (cf. ADR-0006).
func init() {
	Register("mesfichiers", mesFichiersApplet)
}

// mesFichiersApplet présente l'espace PRIVÉ de l'utilisateur : liste de ses
// fichiers, téléchargement (par numéro, XMODEM) et téléversement (T, quota
// appliqué). Réservé aux comptes identifiés — un invité (ou une session sans
// espace personnel configuré) est refusé proprement.
func mesFichiersApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	if ac.State == nil || ac.State.User == nil {
		header(s, "MES FICHIERS")
		writeErr(s, "Reserve aux membres identifies (connectez-vous).")
		anyKey(s)
		return Outcome{}
	}
	if ac.State.UserFiles == nil {
		header(s, "MES FICHIERS")
		writeErr(s, "Espace personnel indisponible.")
		anyKey(s)
		return Outcome{}
	}
	handle := ac.State.User.Handle
	lib, err := ac.State.UserFiles.For(handle)
	if err != nil {
		header(s, "MES FICHIERS")
		writeErr(s, "Espace indisponible : "+err.Error())
		anyKey(s)
		return Outcome{}
	}

	for {
		header(s, "MES FICHIERS")
		list, err := lib.List()
		if err != nil {
			writeErr(s, "Lecture impossible : "+err.Error())
			anyKey(s)
			return Outcome{}
		}
		var used int64
		for _, f := range list {
			used += f.Size
		}
		b := oascii.New()
		b.Ink(oascii.Cyan).Text(fmt.Sprintf("%s : %d fichier(s), %do", handle, len(list), used)).Newline().Newline()
		if len(list) == 0 {
			b.Ink(oascii.White).Text("(aucun fichier)").Newline()
		} else {
			for i, f := range list {
				if i >= 9 {
					break // choix par chiffre 1..9
				}
				b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %d ", i+1))
				b.Ink(oascii.White).Text(fmt.Sprintf("%-20s %5do", f.Name, f.Size)).Newline()
			}
		}
		b.Newline().Ink(oascii.Green).Text("1-9=DL T=env E=eff R=ren Q=ret > ")
		if s.Write(b.String()) != nil {
			return Outcome{Quit: true}
		}

		key, err := s.ReadKey()
		if err != nil {
			return Outcome{Quit: true}
		}
		switch {
		case key == 'Q' || key == 'q':
			return Outcome{Done: true}
		case key == 'T' || key == 't':
			uploadToPersonal(s, ac, handle)
		case key == 'E' || key == 'e':
			deleteFromPersonal(s, ac, handle, list)
		case key == 'R' || key == 'r':
			renameInPersonal(s, ac, handle, list)
		case key >= '1' && key <= '9':
			idx := int(key - '1')
			if idx < len(list) && idx < 9 {
				sendFileDownload(s, lib, list[idx].Name)
			}
		}
	}
}

// pickFile demande un numéro de fichier (1..min(len,9)) et renvoie l'indice choisi,
// ou -1 si l'entrée est invalide (annulation).
func pickFile(s *server.Session, list []files.Info, verbe string) int {
	b := oascii.New()
	b.Newline().Ink(oascii.Green).Text(verbe + " quel fichier (1-" + itoa(min(len(list), 9)) + ", autre=annuler) ? ")
	if s.Write(b.String()) != nil {
		return -1
	}
	key, err := s.ReadKey()
	if err != nil {
		return -1
	}
	idx := int(key - '1')
	if idx < 0 || idx >= len(list) || idx >= 9 {
		return -1
	}
	return idx
}

// deleteFromPersonal supprime un fichier de l'espace personnel (avec confirmation).
func deleteFromPersonal(s *server.Session, ac *AppContext, handle string, list []files.Info) {
	if len(list) == 0 {
		return
	}
	idx := pickFile(s, list, "Supprimer")
	if idx < 0 {
		return
	}
	f := list[idx]
	c := oascii.New()
	c.Newline().Ink(oascii.Yellow).Text("Supprimer " + f.Name + " ? (O/N) ")
	_ = s.Write(c.String())
	k, err := s.ReadKey()
	if err != nil || (k != 'O' && k != 'o') {
		return
	}
	if err := ac.State.UserFiles.Delete(handle, f.Name); err != nil {
		writeErr(s, "Echec : "+err.Error())
		anyKey(s)
	}
	// Retour à la boucle : la liste rafraîchie montre la suppression.
}

// renameInPersonal renomme un fichier de l'espace personnel.
func renameInPersonal(s *server.Session, ac *AppContext, handle string, list []files.Info) {
	if len(list) == 0 {
		return
	}
	idx := pickFile(s, list, "Renommer")
	if idx < 0 {
		return
	}
	f := list[idx]
	nom, err := prompt(s, "Nouveau nom (vide=annuler)")
	if err != nil {
		return
	}
	nom = strings.TrimSpace(nom)
	if nom == "" || nom == f.Name {
		return
	}
	if err := ac.State.UserFiles.Rename(handle, f.Name, nom); err != nil {
		writeErr(s, "Echec : "+err.Error())
		anyKey(s)
	}
}

// uploadToPersonal reçoit un fichier via XMODEM et l'enregistre dans l'espace
// personnel de handle, quota appliqué (Store.Write). Refus explicite si le nom
// est invalide ou le quota dépassé — sans écriture partielle.
func uploadToPersonal(s *server.Session, ac *AppContext, handle string) {
	header(s, "TELEVERSEMENT PERSO")
	name, err := prompt(s, "Nom du fichier (vide=annuler)")
	if err != nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	info := oascii.New()
	info.Newline().Ink(oascii.Yellow).Text("Pret a recevoir " + name + " (XMODEM)...").Newline()
	info.Ink(oascii.White).Text("Demarrez l'envoi sur votre terminal.").Newline()
	_ = s.Write(info.String())
	_ = s.Write(oascii.SendCmd())

	data, err := xmodem.Receive(s.Raw())
	if err != nil {
		s.ClearDeadline()
		writeErr(s, "Transfert echoue : "+err.Error())
		anyKey(s)
		return
	}
	s.ClearDeadline()
	if err := ac.State.UserFiles.Write(handle, name, data); err != nil {
		writeErr(s, "Refuse : "+err.Error())
		anyKey(s)
		return
	}
	okMsg := oascii.New()
	okMsg.Newline().Ink(oascii.Green).Text(fmt.Sprintf("Recu : %s (%d octets).", name, len(data))).Newline()
	okMsg.Text("Appuyez sur une touche...").Newline()
	_ = s.Write(okMsg.String())
	_, _ = s.ReadKey()
}
