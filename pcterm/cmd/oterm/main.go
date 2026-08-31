// Commande oterm : terminal OASCII portable pour le BBS Oric.
//
// Se connecte au serveur bbsd en TCP et rend le flux OASCII (mode TEXT 40×28,
// attributs Téletexte, couleurs, inverse) dans n'importe quel terminal ANSI —
// Linux, Windows ou macOS, en binaire statique unique. Le clavier est transmis
// caractère par caractère (modèle d'entrée du BBS, ADR-0002). Les pages HIRES et
// les transferts XMODEM ne sont pas rendus par ce client texte (message d'état).
//
// Usage :
//
//	oterm -addr host:port      (défaut 127.0.0.1:6502)
//
// Quitter : Ctrl-] (0x1D).
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/benedictemarty/bbsoric/pcterm/internal/ula"
	"golang.org/x/term"
)

const quitKey = 0x1D // Ctrl-] : ferme proprement la session

func main() {
	addr := flag.String("addr", "127.0.0.1:6502", "adresse du serveur bbsd (host:port)")
	flag.Parse()

	if err := run(*addr); err != nil {
		fmt.Fprintln(os.Stderr, "oterm:", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connexion à %s : %w", addr, err)
	}
	defer conn.Close()

	// Passe l'entrée standard en mode brut (touche par touche, sans écho local) —
	// portable Linux/Windows/macOS via golang.org/x/term. Si l'entrée n'est pas un
	// terminal (redirection, smoke-test headless), on saute le mode brut.
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("mode brut du terminal : %w", err)
		}
		defer term.Restore(fd, oldState)
	}

	t := ula.New()
	var mu sync.Mutex
	var lastStatus string

	repaint := func() {
		mu.Lock()
		defer mu.Unlock()
		out := t.RenderANSI()
		if st := t.Status(); st != lastStatus {
			lastStatus = st
			// Affiche le statut sur une 29e ligne (sous l'écran 40×28).
			out += "\x1b[29;1H\x1b[2K"
			if st != "" {
				out += "\x1b[33m[" + st + "]\x1b[0m"
			}
		}
		io.WriteString(os.Stdout, out)
	}

	// Écran d'accueil propre.
	io.WriteString(os.Stdout, "\x1b[2J")
	repaint()

	// Réception : lit le flux du serveur, décode et repeint.
	errc := make(chan error, 2)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				mu.Lock()
				t.Write(buf[:n])
				mu.Unlock()
				repaint()
			}
			if err != nil {
				errc <- err
				return
			}
		}
	}()

	// Émission : lit le clavier octet par octet et l'envoie au serveur.
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if buf[0] == quitKey {
					errc <- nil
					return
				}
				if _, err := conn.Write(buf[:n]); err != nil {
					errc <- err
					return
				}
			}
			if err != nil {
				errc <- err
				return
			}
		}
	}()

	err = <-errc
	// Nettoyage visuel : curseur visible, ligne sous l'écran, message de fin.
	io.WriteString(os.Stdout, "\x1b[?25h\x1b[29;1H\x1b[2K\r\n")
	if err != nil && err != io.EOF {
		return err
	}
	fmt.Fprintln(os.Stdout, "Session terminée.")
	return nil
}
