// Package xfer déroule les transferts de fichiers XMODEM côté client oterm, une
// fois qu'un déclencheur (1F FE download / 1F FD upload) a été détecté dans le
// flux. Il réutilise le protocole partagé internal/xmodem (identique au serveur
// et au firmware Oric).
//
// Download : après 1F FE, le serveur envoie un en-tête de longueur fixe (2 octets
// de nombre de blocs + 12 octets de nom Sedoric + 3 octets de taille réelle, cf.
// server/internal/bbs/xfer.go downloadHeader v4) puis le flux XMODEM. On lit l'en-tête,
// on reçoit le fichier, on le tronque à la taille réelle et on l'enregistre sous
// son nom réel dans le répertoire de téléchargement.
package xfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benedictemarty/bbsoric/internal/xmodem"
)

// headerLen est la longueur de l'en-tête de download v4 (2 + 12 + 3).
const headerLen = 17

// Download lit l'en-tête, reçoit le fichier via XMODEM et l'écrit dans dir sous
// son nom réel. Renvoie le chemin écrit et le nombre d'octets.
func Download(c xmodem.Conn, dir string) (string, int, error) {
	var hdr [headerLen]byte
	_ = c.SetReadDeadline(time.Now().Add(30 * time.Second))
	if _, err := io.ReadFull(c, hdr[:]); err != nil {
		return "", 0, fmt.Errorf("en-tête de download : %w", err)
	}
	name := SedoricName(hdr[2:14])
	size := int(hdr[14]) | int(hdr[15])<<8 | int(hdr[16])<<16

	data, err := xmodem.Receive(c)
	if err != nil {
		return "", 0, err
	}
	if size > 0 && size <= len(data) {
		data = data[:size] // taille exacte (sans le bourrage 128 de XMODEM)
	}
	if name == "" {
		name = "BBSFILE.BIN"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", 0, err
	}
	return path, len(data), nil
}

// CancelUpload avorte proprement un upload demandé par le serveur (1F FD) que le
// client texte ne prend pas en charge : envoie des CAN pour que le récepteur du
// serveur abandonne au lieu d'attendre son délai.
func CancelUpload(c xmodem.Conn) {
	_, _ = c.Write([]byte{0x18, 0x18, 0x18}) // CAN CAN CAN
}

// SedoricName convertit les 12 octets de nom Sedoric (9 de nom + 3 d'extension,
// complétés d'espaces, en majuscules) en un nom de fichier « NOM.EXT ». Les octets
// non alphanumériques sont ignorés (parité avec sedoricName côté serveur).
func SedoricName(b []byte) string {
	if len(b) < 12 {
		return ""
	}
	clean := func(part []byte) string {
		var sb strings.Builder
		for _, c := range part {
			if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				sb.WriteByte(c)
			}
		}
		return sb.String()
	}
	base := clean(b[0:9])
	ext := clean(b[9:12])
	if base == "" {
		return ""
	}
	if ext != "" {
		return base + "." + ext
	}
	return base
}
