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
	// L'en-tête de download code la taille réelle sur 16 bits (cf. downloadHeader) :
	// au-delà, elle serait tronquée silencieusement et la sauvegarde côté terminal
	// serait corrompue. On refuse proprement plutôt que d'émettre un en-tête faux.
	if len(data) > maxDownloadSize {
		writeErr(s, fmt.Sprintf("Fichier trop volumineux : %do (max %do).", len(data), maxDownloadSize))
		return Outcome{}
	}

	info := oascii.New()
	info.Newline().Ink(oascii.Yellow).Text("Envoi de " + f.Name + " (XMODEM)...").Newline()
	info.Ink(oascii.White).Text("Demarrez la reception sur votre terminal.").Newline()
	_ = s.Write(info.String())

	// Signale au terminal Oric de basculer en réception XMODEM (les autres
	// clients ignorent cette séquence de contrôle). En-tête de download (après
	// 1F FE), longueur fixe pour une lecture déterministe côté 6502 :
	//   - nombre total de blocs de 128 o (octet bas, octet haut) -> jauge ;
	//   - nom de fichier au format Sedoric 8.3 (12 octets, complété d'espaces)
	//     -> le terminal sauve sous ce nom réel au lieu d'un nom figé ;
	//   - taille réelle du fichier en octets (octet bas, octet haut) -> header v3,
	//     le terminal tronque la sauvegarde à cette taille au lieu du multiple de
	//     128 paddé par XMODEM.
	// Un terminal plus ancien (qui ne lisait que 2 octets) n'est PAS compatible
	// avec cet en-tête : terminal et serveur évoluent ensemble.
	_ = s.Write(oascii.RecvCmd())
	_ = s.Write(string(downloadHeader(f.Name, len(data))))

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

// maxDownloadSize borne la taille d'un fichier téléchargeable : l'en-tête de
// download (downloadHeader) code la taille réelle sur 2 octets, donc 0xFFFF est
// le plus grand fichier représentable sans troncature.
const maxDownloadSize = 0xFFFF

// itoa convertit un petit entier positif (0..9) en chaîne.
func itoa(n int) string { return string(rune('0' + n)) }

// downloadHeader construit l'en-tête v3 envoyé (après RecvCmd) avant le flux
// XMODEM, de longueur fixe pour une lecture déterministe côté 6502 :
//   - nombre total de blocs de 128 o (octet bas, octet haut) -> jauge ;
//   - nom de fichier Sedoric 8.3 (12 octets) -> sauvegarde sous le vrai nom ;
//   - taille réelle en octets (octet bas, octet haut) -> le terminal tronque la
//     sauvegarde à cette taille au lieu du multiple de 128 paddé par XMODEM.
func downloadHeader(name string, dataLen int) []byte {
	totalBlocks := (dataLen + 127) / 128
	hdr := []byte{byte(totalBlocks & 0xFF), byte((totalBlocks >> 8) & 0xFF)}
	hdr = append(hdr, sedoricName(name)...)
	hdr = append(hdr, byte(dataLen&0xFF), byte((dataLen>>8)&0xFF))
	return hdr
}

// sedoricName convertit un nom de fichier en nom Sedoric 8.3 sur 12 octets :
// 9 octets de nom (justifié à gauche, complété d'espaces) + 3 octets d'extension.
// Majuscules ; seuls [A-Z0-9] sont conservés (les autres deviennent espace). Le
// terminal Oric copie ces 12 octets tels quels dans BUFNOM pour la sauvegarde.
func sedoricName(name string) []byte {
	base, ext := name, ""
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		base, ext = name[:i], name[i+1:]
	}
	out := make([]byte, 12)
	for i := range out {
		out[i] = ' '
	}
	put := func(src string, off, n int) {
		j := 0
		for k := 0; k < len(src) && j < n; k++ {
			c := src[k]
			if c >= 'a' && c <= 'z' {
				c -= 0x20 // majuscule
			}
			if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				out[off+j] = c
				j++
			}
		}
	}
	put(base, 0, 9)
	put(ext, 9, 3)
	return out
}
