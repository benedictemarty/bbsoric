package bbs

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// hiresSiteJSON : accueil → page graphique HIRES (fond + une primitive).
const hiresSiteJSON = `{
  "start": "accueil",
  "pages": {
    "accueil": {"title":"BIENVENUE","entries":[
      {"key":"1","label":"Logo","target":"logo"},
      {"key":"Q","label":"Quitter","target":"__quit__"}
    ]},
    "logo": {"title":"LOGO","hires":{
      "draw":[
        {"op":"ink","c":3},
        {"op":"curset","x":0,"y":0},
        {"op":"box","x":239,"y":199},
        {"op":"circle","r":40}
      ]
    }}
  }
}`

// TestHiresPageEmitsStream : naviguer vers une page HIRES émet le flux de commandes
// graphique (1F FC … HiEnd), avec les opcodes des primitives déclarées.
func TestHiresPageEmitsStream(t *testing.T) {
	addr, stop := startServerWithStore(t, storeFromJSON(t, hiresSiteJSON))
	defer stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Lit l'accueil, puis demande la page HIRES.
	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("lecture accueil: %v", err)
	}
	if _, err := conn.Write([]byte("1")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Accumule la réponse jusqu'à voir la fin du flux HIRES (ou timeout).
	var got []byte
	deadline := time.Now().Add(2 * time.Second)
	cmd := []byte{oascii.PlotByte, 0xFC}
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		got = append(got, buf[:n]...)
		if bytes.Contains(got, cmd) && bytes.IndexByte(got[bytes.Index(got, cmd):], oascii.HiEnd) >= 0 {
			break
		}
		if err != nil {
			break
		}
	}

	i := bytes.Index(got, cmd)
	if i < 0 {
		t.Fatalf("flux HIRES (1F FC) absent ; reçu %d octets", len(got))
	}
	stream := got[i:]
	if stream[2] != oascii.HiOn {
		t.Errorf("le flux ne commence pas par HiOn : %v", stream[:4])
	}
	for _, op := range []byte{oascii.HiInk, oascii.HiCurset, oascii.HiBox, oascii.HiCircle, oascii.HiEnd} {
		if bytes.IndexByte(stream, op) < 0 {
			t.Errorf("opcode 0x%02X attendu dans le flux, absent", op)
		}
	}
}
