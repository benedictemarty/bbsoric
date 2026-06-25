package bbs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/presence"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre les applets de communication entre appelants.
func init() {
	Register("who", whoApplet)
	Register("chat", chatApplet)
}

// whoApplet affiche la liste des appelants connectés (« qui est en ligne »),
// triée par ancienneté de connexion, avec la durée de présence.
func whoApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "QUI EST EN LIGNE")
	reg := ac.State.Presence
	if reg == nil {
		writeErr(s, "Presence indisponible.")
		anyKey(s)
		return Outcome{}
	}
	list := reg.List()
	b := oascii.New()
	b.Ink(oascii.Cyan).Text(fmt.Sprintf(" %d appelant(s) connecte(s) :", len(list))).Newline().Newline()
	now := time.Now()
	for i, m := range list {
		if i >= 20 {
			b.Ink(oascii.Magenta).Text(" ...").Newline()
			break
		}
		me := ""
		if m.ID == ac.State.MemberID {
			me = " (vous)"
		}
		b.Ink(oascii.Yellow).Text(fmt.Sprintf(" %-16s", trunc(m.Handle, 16)))
		b.Ink(oascii.White).Text(fmt.Sprintf(" %6s", since(now, m.Since)))
		b.Ink(oascii.Green).Text(me).Newline()
	}
	b.Newline().Ink(oascii.Green).Text("Appuyez sur une touche...").Newline()
	_ = s.Write(b.String())
	anyKey(s)
	return Outcome{}
}

// chatApplet est un salon de discussion temps réel entre appelants connectés.
//
// Contrainte : le moteur lit l'entrée de façon synchrone et un seul goroutine
// doit lire le flux de la session (sinon vol d'octets au moteur). On évite donc
// tout goroutine lecteur : une boucle unique lit octet par octet avec une courte
// échéance (deadline) ; à chaque expiration on draine les messages reçus et on
// repeint la ligne de saisie en cours. Les frappes ne sont jamais perdues entre
// deux expirations (le tampon de ligne persiste).
func chatApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	header(s, "CHAT")
	reg := ac.State.Presence
	if reg == nil {
		writeErr(s, "Chat indisponible.")
		anyKey(s)
		return Outcome{}
	}
	name := ac.State.displayName()
	id := ac.State.MemberID

	intro := oascii.New()
	intro.Ink(oascii.White).Text(" Salon de discussion. Tapez un message").Newline()
	intro.Text(" puis RETURN. ").Ink(oascii.Yellow).Text("/q").Ink(oascii.White).Text(" pour quitter.").Newline().Newline()
	if s.Write(intro.String()) != nil {
		return Outcome{Quit: true}
	}

	ch, backlog := reg.Subscribe(id)
	defer reg.Unsubscribe(id)
	for _, m := range backlog {
		writeChatLine(s, m, id)
	}
	reg.Publish(presence.Message{FromID: id, From: name, Text: "a rejoint le chat", System: true})
	defer reg.Publish(presence.Message{FromID: id, From: name, Text: "a quitte le chat", System: true})

	defer s.ClearDeadline()
	raw := s.Raw()
	var line []byte
	writePrompt(s, line)

	for {
		if ctx.Err() != nil {
			return Outcome{Quit: true}
		}
		// Draine les messages reçus depuis la dernière itération.
		got := false
		for drain := true; drain; {
			select {
			case m := <-ch:
				if !got {
					_ = s.Write("\r\n") // quitte la ligne de saisie en cours
				}
				writeChatLine(s, m, id)
				got = true
			default:
				drain = false
			}
		}
		if got {
			writePrompt(s, line) // réaffiche l'invite + la saisie en cours
		}

		raw.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		c, err := readByteRaw(raw)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue // rien tapé : on reboucle pour drainer le chat
			}
			return Outcome{Quit: true} // déconnexion
		}
		switch c {
		case '\r', '\n':
			text := strings.TrimSpace(string(line))
			line = line[:0]
			_ = s.Write("\r\n")
			switch {
			case text == "":
				// ligne vide : simple rafraîchissement
			case text == "/q" || text == "/quit":
				return Outcome{Done: true}
			default:
				reg.Publish(presence.Message{FromID: id, From: name, Text: text})
			}
			writePrompt(s, line)
		case 8, 127: // backspace / delete
			if len(line) > 0 {
				line = line[:len(line)-1]
				_ = s.Write("\b \b")
			}
		case 0xFF: // début de commande telnet IAC : on saute les 2 octets suivants
			_, _ = readByteRaw(raw)
			_, _ = readByteRaw(raw)
		default:
			if c >= 32 && c < 127 && len(line) < oascii.Cols-4 {
				line = append(line, c)
				_ = s.Write(string(c)) // écho local
			}
		}
	}
}

// readByteRaw lit un octet sur le canal brut de la session.
func readByteRaw(raw *server.RawConn) (byte, error) {
	var buf [1]byte
	n, err := raw.Read(buf[:])
	if n == 1 {
		return buf[0], nil
	}
	return 0, err
}

// writePrompt (ré)affiche l'invite du chat et la saisie en cours.
func writePrompt(s *server.Session, line []byte) {
	_ = s.Write(makeInk(oascii.Green) + "> " + makeInk(oascii.White) + string(line))
}

// writeChatLine affiche un message du chat. Les messages émis par la session
// elle-même (FromID == id) ne sont pas réaffichés (déjà vus à la frappe).
func writeChatLine(s *server.Session, m presence.Message, id uint64) {
	if m.FromID == id {
		return
	}
	b := oascii.New()
	ts := m.At.Format("15:04")
	if m.System {
		b.Ink(oascii.Magenta).Text(fmt.Sprintf(" * %s %s", m.From, m.Text)).Newline()
	} else {
		b.Ink(oascii.Cyan).Text(ts + " ")
		b.Ink(oascii.Yellow).Text(trunc(m.From, 12) + " ")
		b.Ink(oascii.White).Text(m.Text).Newline()
	}
	_ = s.Write(b.String())
}

// anyKey attend une frappe (écran « appuyez sur une touche »).
func anyKey(s *server.Session) { _, _ = s.ReadKey() }

// trunc tronque une chaîne à n caractères (sans ellipse).
func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// since rend une durée écoulée compacte (« 3m », « 1h05 », « 12s »).
func since(now, t time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh%02d", int(d.Hours()), int(d.Minutes())%60)
	}
}
