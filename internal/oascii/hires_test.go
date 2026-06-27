package oascii

import (
	"bytes"
	"testing"
)

func TestRLERoundTrip(t *testing.T) {
	cas := [][]byte{
		{},
		{0},
		bytes.Repeat([]byte{0xAA}, 300),               // run > 255 → plusieurs paires
		append(bytes.Repeat([]byte{0}, 100), 1, 2, 3), // grande plage + variation
		{1, 1, 2, 3, 3, 3, 4},
	}
	for i, in := range cas {
		rle := RLEEncode(in)
		out := RLEDecode(rle, len(in))
		if !bytes.Equal(in, out) {
			t.Errorf("cas %d : round-trip RLE cassé\n in=%v\nout=%v", i, in, out)
		}
	}
}

func TestRLECompresse(t *testing.T) {
	// 8000 octets identiques → ~32 paires (8000/255 ≈ 32) au lieu de 8000 octets.
	in := bytes.Repeat([]byte{0x40}, 8000)
	rle := RLEEncode(in)
	if len(rle) > 100 {
		t.Errorf("RLE d'une plage uniforme trop long : %d octets", len(rle))
	}
	if !bytes.Equal(RLEDecode(rle, len(in)), in) {
		t.Error("décodage RLE d'une plage uniforme incorrect")
	}
}

func TestHiresCmdPrefixe(t *testing.T) {
	if got := []byte(HiresCmd()); !bytes.Equal(got, []byte{PlotByte, hiresByte}) {
		t.Errorf("HiresCmd = %v, attendu [1F FC]", got)
	}
}
