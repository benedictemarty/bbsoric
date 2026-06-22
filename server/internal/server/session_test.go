package server

import (
	"net"
	"testing"
)

// pipeSession crée une Session reliée à un bout de net.Pipe ; l'autre bout sert
// à injecter des octets façon client.
func pipeSession(t *testing.T) (*Session, net.Conn) {
	t.Helper()
	srv, cli := net.Pipe()
	t.Cleanup(func() { srv.Close(); cli.Close() })
	return newSession(srv, 0), cli
}

func TestReadKeySingleByte(t *testing.T) {
	s, cli := pipeSession(t)
	go func() { cli.Write([]byte{'1'}) }()
	k, err := s.ReadKey()
	if err != nil {
		t.Fatalf("ReadKey : %v", err)
	}
	if k != '1' {
		t.Errorf("touche attendue '1', got %q", k)
	}
}

func TestReadKeySkipsResidualNewlines(t *testing.T) {
	// Un client « bête » (nc) laisse CR/LF derrière une ligne ; ils ne doivent
	// pas être pris pour une touche.
	s, cli := pipeSession(t)
	go func() { cli.Write([]byte{'\r', '\n', '2'}) }()
	k, err := s.ReadKey()
	if err != nil {
		t.Fatalf("ReadKey : %v", err)
	}
	if k != '2' {
		t.Errorf("les CR/LF residuels doivent etre ignores, got %q", k)
	}
}

func TestReadKeyFiltersTelnetIAC(t *testing.T) {
	s, cli := pipeSession(t)
	// IAC WILL ECHO (0xFF 0xFB 0x01) puis la vraie touche.
	go func() { cli.Write([]byte{iac, telnetWill, 1, 'A'}) }()
	k, err := s.ReadKey()
	if err != nil {
		t.Fatalf("ReadKey : %v", err)
	}
	if k != 'A' {
		t.Errorf("IAC non filtre, got %q", k)
	}
}

func TestReadKeyDrainsTrailingEOL(t *testing.T) {
	// Un client en mode ligne (nc) envoie « 1\r\n » puis la saisie : le CR/LF
	// derrière la touche ne doit pas être lu comme une ligne vide par ReadLine.
	s, cli := pipeSession(t)
	go func() { cli.Write([]byte("1\r\nbob\r")) }()
	k, err := s.ReadKey()
	if err != nil || k != '1' {
		t.Fatalf("ReadKey : %q err %v", k, err)
	}
	line, err := s.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine : %v", err)
	}
	if line != "bob" {
		t.Errorf("ligne attendue \"bob\" (pas vide), got %q", line)
	}
}

func TestReadKeyThenReadLine(t *testing.T) {
	// Scénario réel : une touche de menu, puis une saisie ligne (champ texte).
	s, cli := pipeSession(t)
	go func() { cli.Write([]byte("1bob\r")) }()
	k, err := s.ReadKey()
	if err != nil || k != '1' {
		t.Fatalf("ReadKey : %q err %v", k, err)
	}
	line, err := s.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine : %v", err)
	}
	if line != "bob" {
		t.Errorf("ligne attendue \"bob\", got %q", line)
	}
}
