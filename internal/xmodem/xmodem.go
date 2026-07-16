// Package xmodem implémente le protocole de transfert de fichiers XMODEM
// (blocs de 128 octets), historique des BBS. Il fonctionne sur tout canal
// bidirectionnel d'octets (net.Conn, etc.) exposant une échéance de lecture.
//
// Send (l'appelant émet) et Receive (l'appelant reçoit) gèrent les deux modes de
// contrôle d'intégrité : somme de contrôle simple (NAK) et CRC-16 ('C'). Le
// récepteur impose le mode via son caractère de démarrage.
//
// Limite connue d'XMODEM : la taille exacte du fichier n'est pas transmise ; le
// dernier bloc est complété par des octets SUB (0x1A). Receive élague ce padding
// final — fidèle pour du texte, à garder à l'esprit pour un binaire finissant
// réellement par 0x1A.
package xmodem

import (
	"errors"
	"io"
	"net"
	"os"
	"time"
)

// Octets de contrôle XMODEM.
const (
	soh     = 0x01 // début d'un bloc de 128 octets
	eot     = 0x04 // fin de transmission
	ack     = 0x06 // bloc accepté
	nak     = 0x15 // bloc refusé / démarrage en mode somme de contrôle
	can     = 0x18 // annulation
	sub     = 0x1A // remplissage du dernier bloc
	crcChar = 0x43 // 'C' : démarrage en mode CRC-16
)

const (
	blockSize    = 128
	maxRetries   = 10
	startRetries = 16
	ackTimeout   = 5 * time.Second
	startTimeout = 3 * time.Second
)

// Erreurs renvoyées par Send/Receive.
var (
	ErrTimeout    = errors.New("xmodem : délai dépassé")
	ErrCanceled   = errors.New("xmodem : transfert annulé")
	ErrTooManyNAK = errors.New("xmodem : trop d'erreurs de transmission")
	ErrTooLarge   = errors.New("xmodem : fichier trop volumineux")
)

// maxReceiveBytes borne la taille d'un fichier reçu (garde-fou mémoire face à un
// émetteur bogué ou malveillant). Variable pour permettre des tests déterministes.
var maxReceiveBytes = 1 << 20 // 1 Mio

// isTimeout distingue un dépassement d'échéance de lecture (transitoire, on
// ré-essaie) d'une vraie erreur d'E/S (connexion fermée : on remonte aussitôt).
func isTimeout(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// sendCan signale l'annulation au pair (2 CAN, convention XMODEM). Best-effort.
func sendCan(c Conn) { _, _ = c.Write([]byte{can, can}) }

// Conn est un canal d'octets bidirectionnel avec échéance de lecture
// (satisfait par net.Conn).
type Conn interface {
	io.Reader
	io.Writer
	SetReadDeadline(t time.Time) error
}

func crc16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func checksum(data []byte) byte {
	var s byte
	for _, b := range data {
		s += b
	}
	return s
}

// readByte lit un octet avec échéance.
func readByte(c Conn, timeout time.Duration) (byte, error) {
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	var b [1]byte
	if _, err := io.ReadFull(c, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

// readFull lit exactement len(buf) octets avec échéance.
func readFull(c Conn, buf []byte, timeout time.Duration) error {
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	_, err := io.ReadFull(c, buf)
	return err
}

// Send émet data via XMODEM (l'appelant est l'émetteur). Il attend le caractère
// de démarrage du récepteur (NAK ou 'C'), envoie les blocs avec ré-émission sur
// NAK/timeout, puis EOT.
func Send(c Conn, data []byte) error {
	crc, err := waitStart(c)
	if err != nil {
		return err
	}
	blk := byte(1)
	for off := 0; off < len(data); off += blockSize {
		var block [blockSize]byte
		n := copy(block[:], data[off:])
		for i := n; i < blockSize; i++ {
			block[i] = sub
		}
		if err := sendBlock(c, blk, block[:], crc); err != nil {
			return err
		}
		blk++
	}
	// Fin de transmission : EOT jusqu'à ACK.
	for retry := 0; retry < maxRetries; retry++ {
		if _, err := c.Write([]byte{eot}); err != nil {
			return err
		}
		resp, err := readByte(c, ackTimeout)
		if err != nil {
			if isTimeout(err) {
				continue // pas d'ACK à temps → ré-émettre EOT
			}
			return err // erreur d'E/S réelle : inutile d'insister
		}
		if resp == ack {
			return nil
		}
	}
	sendCan(c)
	return ErrTooManyNAK
}

// waitStart attend le caractère de démarrage du récepteur : 'C' → CRC, NAK →
// somme de contrôle.
func waitStart(c Conn) (crc bool, err error) {
	for i := 0; i < startRetries; i++ {
		b, e := readByte(c, startTimeout)
		if e != nil {
			if isTimeout(e) {
				continue
			}
			return false, e // erreur d'E/S réelle : remonter
		}
		switch b {
		case crcChar:
			return true, nil
		case nak:
			return false, nil
		case can:
			return false, ErrCanceled
		}
	}
	return false, ErrTimeout
}

// sendBlock envoie un bloc et attend l'ACK (ré-émission sur NAK/timeout).
func sendBlock(c Conn, blk byte, data []byte, crc bool) error {
	frame := make([]byte, 0, 3+blockSize+2)
	frame = append(frame, soh, blk, ^blk)
	frame = append(frame, data...)
	if crc {
		ck := crc16(data)
		frame = append(frame, byte(ck>>8), byte(ck))
	} else {
		frame = append(frame, checksum(data))
	}
	for retry := 0; retry < maxRetries; retry++ {
		if _, err := c.Write(frame); err != nil {
			return err
		}
		resp, err := readByte(c, ackTimeout)
		if err != nil {
			if isTimeout(err) {
				continue // timeout → ré-émettre
			}
			return err // erreur d'E/S réelle : remonter
		}
		switch resp {
		case ack:
			return nil
		case can:
			return ErrCanceled
		}
		// NAK ou autre → ré-émettre
	}
	sendCan(c)
	return ErrTooManyNAK
}

// Receive reçoit un fichier via XMODEM (l'appelant est le récepteur). Il amorce
// en mode CRC ('C'), bascule en somme de contrôle si l'émetteur ne répond pas,
// accuse réception de chaque bloc, et élague le remplissage SUB final.
func Receive(c Conn) ([]byte, error) {
	var out []byte
	crc := true
	expected := byte(1)
	starting := true
	errCount := 0

	for {
		if starting {
			start := byte(crcChar)
			if !crc {
				start = nak
			}
			if _, err := c.Write([]byte{start}); err != nil {
				return nil, err
			}
		}
		b, err := readByte(c, startTimeout)
		if err != nil {
			if !isTimeout(err) {
				return nil, err // erreur d'E/S réelle : remonter tout de suite
			}
			errCount++
			if errCount > startRetries {
				sendCan(c)
				return nil, ErrTimeout
			}
			if crc && errCount >= 4 { // bascule en somme de contrôle
				crc = false
			}
			continue
		}
		switch b {
		case eot:
			_, _ = c.Write([]byte{ack})
			return trimPadding(out), nil
		case can:
			return nil, ErrCanceled
		case soh:
			starting = false
			block, blkNum, ok := readBlock(c, crc)
			if !ok {
				_, _ = c.Write([]byte{nak})
				continue
			}
			switch {
			case blkNum == expected:
				out = append(out, block...)
				expected++
				if len(out) > maxReceiveBytes {
					sendCan(c)
					return nil, ErrTooLarge
				}
				_, _ = c.Write([]byte{ack})
			case blkNum == expected-1:
				_, _ = c.Write([]byte{ack}) // bloc répété → ré-ACK
			default:
				_, _ = c.Write([]byte{nak})
			}
		}
	}
}

// readBlock lit l'en-tête (num, ~num), les 128 octets et le contrôle d'intégrité.
func readBlock(c Conn, crc bool) (data []byte, blkNum byte, ok bool) {
	hdr := make([]byte, 2)
	if readFull(c, hdr, ackTimeout) != nil {
		return nil, 0, false
	}
	if hdr[0] != ^hdr[1] {
		return nil, 0, false
	}
	buf := make([]byte, blockSize)
	if readFull(c, buf, ackTimeout) != nil {
		return nil, 0, false
	}
	if crc {
		ck := make([]byte, 2)
		if readFull(c, ck, ackTimeout) != nil {
			return nil, 0, false
		}
		if crc16(buf) != uint16(ck[0])<<8|uint16(ck[1]) {
			return nil, 0, false
		}
	} else {
		ck := make([]byte, 1)
		if readFull(c, ck, ackTimeout) != nil {
			return nil, 0, false
		}
		if checksum(buf) != ck[0] {
			return nil, 0, false
		}
	}
	return buf, hdr[0], true
}

// trimPadding retire le remplissage SUB (0x1A) du dernier bloc.
func trimPadding(data []byte) []byte {
	i := len(data)
	for i > 0 && data[i-1] == sub {
		i--
	}
	return data[:i]
}
