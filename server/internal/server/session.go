package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Session représente une connexion cliente unique au BBS.
//
// Elle masque les détails réseau (deadline d'inactivité, lecture ligne à ligne,
// filtrage minimal des commandes telnet IAC) et offre une petite API d'écran.
// À terme, la couche OASCII (Sprint 1) s'appuiera sur ces primitives pour
// produire les attributs Téletexte sériels de l'Oric.
type Session struct {
	conn        net.Conn
	reader      *bufio.Reader
	idleTimeout time.Duration
}

func newSession(conn net.Conn, idleTimeout time.Duration) *Session {
	return &Session{
		conn:        conn,
		reader:      bufio.NewReader(conn),
		idleTimeout: idleTimeout,
	}
}

// RawConn expose la session comme canal d'octets brut pour les transferts
// binaires (XMODEM) : lecture via le tampon (n'égare pas d'octets déjà bufferisés),
// écriture et échéance de lecture via la connexion. Court-circuite le filtrage
// telnet/ligne — à n'utiliser que pendant un transfert de fichier.
type RawConn struct{ s *Session }

// Raw renvoie un canal brut sur la session (cf. RawConn).
func (s *Session) Raw() *RawConn { return &RawConn{s} }

func (r *RawConn) Read(p []byte) (int, error)            { return r.s.reader.Read(p) }
func (r *RawConn) Write(p []byte) (int, error)           { return r.s.conn.Write(p) }
func (r *RawConn) SetReadDeadline(t time.Time) error     { return r.s.conn.SetReadDeadline(t) }

// ClearDeadline retire toute échéance de lecture (après un transfert, avant de
// revenir aux I/O normales pilotées par l'idle timeout).
func (s *Session) ClearDeadline() { _ = s.conn.SetDeadline(time.Time{}) }

// RemoteIP renvoie l'adresse IP du client (sans le port).
func (s *Session) RemoteIP() string {
	host, _, err := net.SplitHostPort(s.conn.RemoteAddr().String())
	if err != nil {
		return s.conn.RemoteAddr().String()
	}
	return host
}

// touch repousse la deadline d'inactivité. Appelée avant chaque I/O.
func (s *Session) touch() {
	if s.idleTimeout > 0 {
		_ = s.conn.SetDeadline(time.Now().Add(s.idleTimeout))
	}
}

// Write envoie une chaîne brute au client.
func (s *Session) Write(text string) error {
	s.touch()
	_, err := s.conn.Write([]byte(text))
	return err
}

// Printf écrit une chaîne formatée.
func (s *Session) Printf(format string, args ...any) error {
	return s.Write(fmt.Sprintf(format, args...))
}

// Println écrit une ligne suivie de CR LF (fin de ligne attendue par les
// terminaux série / telnet historiques).
func (s *Session) Println(text string) error {
	return s.Write(text + "\r\n")
}

// ReadLine lit une ligne de saisie, en filtrant les commandes telnet IAC
// (négociation complète repoussée au Sprint 1, cf. ROADMAP §Décisions).
// Renvoie la ligne sans le CR/LF final, en minuscules-insensible côté appelant.
func (s *Session) ReadLine() (string, error) {
	s.touch()
	var b strings.Builder
	for {
		c, err := s.reader.ReadByte()
		if err != nil {
			return b.String(), err
		}
		switch c {
		case iac: // 0xFF : début d'une commande telnet → on consomme et on ignore
			if err := s.skipTelnetCommand(); err != nil {
				return b.String(), err
			}
		case '\n':
			return b.String(), nil
		case '\r':
			// L'Oric (RETURN) et de nombreux clients envoient CR seul ou CRLF.
			// On termine la ligne sur CR et on absorbe un LF suivant SEULEMENT
			// s'il est déjà bufferisé (jamais de lecture bloquante : l'Oric
			// envoie CR seul).
			if s.reader.Buffered() > 0 {
				if next, err := s.reader.ReadByte(); err == nil && next != '\n' {
					_ = s.reader.UnreadByte()
				}
			}
			return b.String(), nil
		case 0:
			// NUL ignoré (cf. comportement telnet)
		default:
			b.WriteByte(c)
		}
	}
}

// ReadKey lit une seule touche significative et la renvoie. Utilisé pour les
// choix de menu et les écrans « appuyez sur une touche » : la navigation réagit
// à la première frappe, sans attendre RETURN (cf. ADR-0002).
//
// Filtre les commandes telnet IAC et ignore les CR/LF/NUL résiduels (qu'un
// client « bête » comme nc laisse derrière une ligne précédente) afin qu'ils ne
// soient pas pris pour une touche.
func (s *Session) ReadKey() (byte, error) {
	s.touch()
	for {
		c, err := s.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch c {
		case iac:
			if err := s.skipTelnetCommand(); err != nil {
				return 0, err
			}
		case '\r', '\n', 0:
			// résidus de fin de ligne : ignorés
		default:
			// Draine les CR/LF/NUL DÉJÀ bufferisés derrière la touche (clients en
			// mode ligne comme nc envoient « 1\r\n ») pour qu'ils ne soient pas
			// lus comme une ligne vide par un ReadLine suivant (saisie d'applet).
			// Non bloquant : on ne consomme que ce qui est déjà disponible.
			s.drainBufferedEOL()
			return c, nil
		}
	}
}

// drainBufferedEOL consomme les CR/LF/NUL immédiatement disponibles dans le
// tampon, sans jamais lire sur le réseau (donc sans bloquer).
func (s *Session) drainBufferedEOL() {
	for s.reader.Buffered() > 0 {
		c, err := s.reader.ReadByte()
		if err != nil {
			return
		}
		if c != '\r' && c != '\n' && c != 0 {
			_ = s.reader.UnreadByte()
			return
		}
	}
}

// skipTelnetCommand consomme une commande telnet introduite par IAC (0xFF).
// Gère les commandes à 2 octets (WILL/WONT/DO/DONT + option) et l'IAC échappé.
func (s *Session) skipTelnetCommand() error {
	cmd, err := s.reader.ReadByte()
	if err != nil {
		return err
	}
	switch cmd {
	case iac:
		// IAC IAC = octet 0xFF littéral ; on l'ignore en saisie texte.
		return nil
	case telnetWill, telnetWont, telnetDo, telnetDont:
		_, err = s.reader.ReadByte() // consomme l'octet d'option
		return err
	default:
		return nil
	}
}

// Codes telnet minimaux nécessaires au filtrage de la saisie.
const (
	iac        = 255 // Interpret As Command
	telnetWill = 251
	telnetWont = 252
	telnetDo   = 253
	telnetDont = 254
)
