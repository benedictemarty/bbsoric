package xmodem

import (
	"bytes"
	"io"
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

// TestSendChecksumMode exerce la branche SOMME DE CONTRÔLE de Send (le récepteur
// démarre par NAK au lieu de 'C') — non couverte par le round-trip (qui démarre en CRC).
func TestSendChecksumMode(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	data := bytes.Repeat([]byte("Z"), 200) // 2 blocs
	var serr error
	done := make(chan struct{})
	go func() { serr = Send(a, data); close(done) }()

	// Récepteur manuel en mode somme de contrôle.
	if _, err := b.Write([]byte{nak}); err != nil {
		t.Fatal(err)
	}
	var got []byte
	for {
		one := make([]byte, 1)
		if _, err := io.ReadFull(b, one); err != nil {
			t.Fatalf("lecture: %v", err)
		}
		if one[0] == eot {
			b.Write([]byte{ack})
			break
		}
		if one[0] != soh {
			t.Fatalf("attendu SOH, got %d", one[0])
		}
		hdr := make([]byte, 2)
		io.ReadFull(b, hdr) // blk, ^blk
		buf := make([]byte, blockSize)
		io.ReadFull(b, buf)
		ckb := make([]byte, 1)
		io.ReadFull(b, ckb)
		if checksum(buf) != ckb[0] {
			t.Fatalf("somme de contrôle invalide")
		}
		got = append(got, buf...)
		b.Write([]byte{ack})
	}
	<-done
	if serr != nil {
		t.Fatalf("Send: %v", serr)
	}
	if got = trimPadding(got); !bytes.Equal(got, data) {
		t.Errorf("round-trip somme de contrôle: got %q want %q", got, data)
	}
}

// TestReadBlockChecksum exerce la vérification de bloc en mode somme de contrôle.
func TestReadBlockChecksum(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	payload := bytes.Repeat([]byte("Q"), blockSize)
	frame := []byte{5, ^byte(5)}
	frame = append(frame, payload...)
	frame = append(frame, checksum(payload))
	go func() { a.Write(frame) }()

	data, blk, ok := readBlock(b, false)
	if !ok {
		t.Fatal("readBlock (checksum) doit réussir")
	}
	if blk != 5 {
		t.Errorf("num de bloc = %d, want 5", blk)
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("données de bloc incohérentes")
	}
}

// TestSendSurfacesIOError : une erreur d'E/S réelle (pair fermé) est remontée
// immédiatement, sans boucler jusqu'à ErrTooManyNAK (S11.6).
func TestSendSurfacesIOError(t *testing.T) {
	a, b := net.Pipe()
	b.Close() // pair fermé avant tout échange

	err := Send(a, []byte("data"))
	a.Close()
	if err == nil {
		t.Fatal("Send doit échouer sur un pair fermé")
	}
	if err == ErrTooManyNAK || err == ErrTimeout {
		t.Errorf("une erreur d'E/S réelle doit être remontée, pas %v", err)
	}
}

// TestReceiveRejectsOversize : Receive refuse un flux dépassant la borne mémoire
// et signale l'annulation (CAN) au pair (S11.6).
func TestReceiveRejectsOversize(t *testing.T) {
	old := maxReceiveBytes
	maxReceiveBytes = 200 // 2 blocs de 128 suffisent à dépasser
	defer func() { maxReceiveBytes = old }()

	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	got := make(chan error, 1)
	go func() { _, err := Receive(b); got <- err }()

	// Émetteur manuel CRC.
	start := make([]byte, 1)
	if _, err := io.ReadFull(a, start); err != nil {
		t.Fatal(err)
	}
	if start[0] != crcChar {
		t.Fatalf("attendu 'C', got %d", start[0])
	}
	send := func(blk byte) {
		payload := bytes.Repeat([]byte{blk}, blockSize)
		frame := []byte{soh, blk, ^blk}
		frame = append(frame, payload...)
		ck := crc16(payload)
		frame = append(frame, byte(ck>>8), byte(ck))
		a.Write(frame)
	}
	send(1)
	io.ReadFull(a, make([]byte, 1)) // ACK du bloc 1
	send(2)                         // franchit la borne
	io.ReadFull(a, make([]byte, 2)) // draine le CAN émis par Receive

	if err := <-got; err != ErrTooLarge {
		t.Errorf("attendu ErrTooLarge, got %v", err)
	}
}

func TestTrimPadding(t *testing.T) {
	in := append([]byte("ok"), sub, sub, sub)
	if got := trimPadding(in); !bytes.Equal(got, []byte("ok")) {
		t.Errorf("trimPadding = %q, want \"ok\"", got)
	}
}
