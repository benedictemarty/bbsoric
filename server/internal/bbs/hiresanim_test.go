package bbs

import (
	"bufio"
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/files"
)

const hiresAnimSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Animation", "applet": "hiresanim", "next": "accueil" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// TestHiresAnimFirstFrame vérifie que l'applet d'animation émet bien un flux HIRES
// différentiel : la 1ʳᵉ image débute par 1F FC HiOn et contient un HiBlit. On lit
// juste le début puis on ferme (inutile d'attendre les 5 s d'animation).
func TestHiresAnimFirstFrame(t *testing.T) {
	lib, _ := files.Open(t.TempDir(), 0)
	addr, stop := startBBSFiles(t, hiresAnimSiteJSON, lib)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	if _, err := conn.Write([]byte("1")); err != nil { // -> applet hiresanim
		t.Fatalf("write: %v", err)
	}

	// Lit le début du flux d'animation (première image).
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("lecture flux animation: %v", err)
	}
	frame := buf[:n]

	// 1ʳᵉ image : 1F FC HiOn.
	head := []byte{oascii.PlotByte, 0xFC, oascii.HiOn}
	if !bytes.HasPrefix(frame, head) {
		t.Fatalf("le flux doit débuter par 1F FC HiOn, a %v", frame[:min3(len(frame))])
	}
	// Doit contenir au moins un HiBlit (le cadre + la balle).
	if !bytes.Contains(frame, []byte{oascii.HiBlit}) {
		t.Error("la 1ʳᵉ image devrait contenir un HiBlit")
	}
	// Fermer coupe l'animation : le prochain Write de l'applet échoue et il rend la main.
}

func min3(n int) int {
	if n < 3 {
		return n
	}
	return 3
}
