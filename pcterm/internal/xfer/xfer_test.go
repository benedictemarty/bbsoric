package xfer

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/xmodem"
)

func TestSedoricName(t *testing.T) {
	cas := []struct {
		in   string // 12 caractères
		want string
	}{
		{"CENTI    TAP", "CENTI.TAP"},
		{"JEU      BIN", "JEU.BIN"},
		{"NOEXT       ", "NOEXT"},
		{"            ", ""},
	}
	for _, c := range cas {
		if got := SedoricName([]byte(c.in)); got != c.want {
			t.Errorf("SedoricName(%q) = %q, veut %q", c.in, got, c.want)
		}
	}
}

// TestDownloadLoopback : un vrai transfert XMODEM en boucle locale via net.Pipe.
// Un côté joue le serveur (en-tête downloadHeader + xmodem.Send), l'autre est le
// client qui télécharge et écrit le fichier.
func TestDownloadLoopback(t *testing.T) {
	srv, cli := net.Pipe()
	defer cli.Close()

	payload := []byte("Bonjour depuis le BBS Oric — contenu de test XMODEM.")
	dir := t.TempDir()

	// Côté serveur : en-tête (blocs, nom Sedoric 12 o, taille) puis flux XMODEM.
	go func() {
		defer srv.Close()
		blocks := (len(payload) + 127) / 128
		hdr := []byte{byte(blocks & 0xFF), byte(blocks >> 8)}
		hdr = append(hdr, []byte("TESTFILE TXT")...) // 12 octets (9+3)
		// Taille réelle sur 3 octets (en-tête v4, lo/mid/hi).
		hdr = append(hdr, byte(len(payload)&0xFF), byte((len(payload)>>8)&0xFF), byte((len(payload)>>16)&0xFF))
		_ = srv.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := srv.Write(hdr); err != nil {
			return
		}
		_ = xmodem.Send(srv, payload)
	}()

	path, n, err := Download(cli, dir)
	if err != nil {
		t.Fatalf("Download : %v", err)
	}
	if n != len(payload) {
		t.Errorf("taille reçue = %d, veut %d", n, len(payload))
	}
	if base := filepath.Base(path); base != "TESTFILE.TXT" {
		t.Errorf("nom de fichier = %q, veut TESTFILE.TXT", base)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("contenu = %q, veut %q", got, payload)
	}
}
