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
			// ignoré : le '\n' qui suit (ou non) termine la ligne
		case 0:
			// NUL ignoré (cf. comportement telnet)
		default:
			b.WriteByte(c)
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
