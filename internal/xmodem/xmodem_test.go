package xmodem

import (
	"bytes"
	"net"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"court", []byte("Hello, BBS Oric!")},
		{"exact128", bytes.Repeat([]byte("A"), 128)},
		{"multibloc", bytes.Repeat([]byte("xy"), 200)}, // 400 octets
		{"vide", []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			defer b.Close()

			var got []byte
			var rerr error
			done := make(chan struct{})
			go func() {
				got, rerr = Receive(b)
				close(done)
			}()
			serr := Send(a, tc.data)
			<-done

			if serr != nil {
				t.Fatalf("Send: %v", serr)
			}
			if rerr != nil {
				t.Fatalf("Receive: %v", rerr)
			}
			if !bytes.Equal(got, tc.data) {
				t.Errorf("round-trip incohérent:\n got %q\nwant %q", got, tc.data)
			}
		})
	}
}

func TestCRCAndChecksum(t *testing.T) {
	// crc16 et checksum sur un vecteur connu.
	data := []byte("123456789")
	if c := crc16(data); c != 0x31C3 {
		t.Errorf("crc16(\"123456789\") = 0x%04X, want 0x31C3", c)
	}
	if s := checksum([]byte{1, 2, 3, 250, 10}); s != 10 { // 266 mod 256
		t.Errorf("checksum = %d, want 10", s)
	}
}

func TestTrimPadding(t *testing.T) {
	in := append([]byte("ok"), sub, sub, sub)
	if got := trimPadding(in); !bytes.Equal(got, []byte("ok")) {
		t.Errorf("trimPadding = %q, want \"ok\"", got)
	}
}
