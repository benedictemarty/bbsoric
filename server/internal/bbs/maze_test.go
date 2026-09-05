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

const mazeSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": { "title": "BIENVENUE", "entries": [
      { "key": "1", "label": "Labyrinthe", "applet": "maze", "next": "accueil" },
      { "key": "Q", "label": "Quitter", "target": "__quit__" }
    ]}
  }
}`

// TestMazeInitialFrameAndQuit : l'applet émet une 1ʳᵉ image HIRES (1F FC HiOn +
// HiBlit du labyrinthe) puis, sur 'Q', repasse en TEXT (écran d'abandon).
func TestMazeInitialFrameAndQuit(t *testing.T) {
	lib, _ := files.Open(t.TempDir(), 0)
	addr, stop := startBBSFiles(t, mazeSiteJSON, lib)
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)

	readUntil(t, r, conn, "Votre choix")
	if _, err := conn.Write([]byte("1")); err != nil { // -> applet maze
		t.Fatalf("write: %v", err)
	}

	// 1ʳᵉ image : lit un bon morceau (le labyrinthe = beaucoup de HiBlit).
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("lecture 1re image: %v", err)
	}
	frame := buf[:n]
	if !bytes.HasPrefix(frame, []byte{oascii.PlotByte, 0xFC, oascii.HiOn}) {
		t.Fatalf("la 1re image doit débuter par 1F FC HiOn, a %v", frame[:min3m(len(frame))])
	}
	if !bytes.Contains(frame, []byte{oascii.HiBlit}) {
		t.Error("la 1re image devrait contenir des HiBlit (murs du labyrinthe)")
	}

	// Abandon : 'Q' -> retour TEXT + écran d'abandon.
	_ = conn.SetReadDeadline(time.Time{})
	if _, err := conn.Write([]byte("Q")); err != nil {
		t.Fatalf("write Q: %v", err)
	}
	readUntil(t, r, conn, "Abandon")
}

func min3m(n int) int {
	if n < 3 {
		return n
	}
	return 3
}
