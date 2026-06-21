#!/usr/bin/env python3
"""Génère un fichier .TAP Oric (autorun) à partir d'un binaire brut.

Format d'en-tête Oric (machine code, autorun) :
  16 16 16 24 | 00 00 | 80(type) | C7(autorun) | endHi endLo | startHi startLo
  | 00 | <nom ASCII> 00 | <données>

Usage : bin2tap.py <in.bin> <start_hex> <name> <out.tap>
"""
import sys


def main():
    if len(sys.argv) != 5:
        sys.exit("usage: bin2tap.py <in.bin> <start_hex> <name> <out.tap>")
    in_bin, start_hex, name, out_tap = sys.argv[1:]
    start = int(start_hex, 16)
    with open(in_bin, "rb") as f:
        data = f.read()
    end = start + len(data) - 1  # adresse du dernier octet (inclus)

    hdr = bytes([0x16, 0x16, 0x16, 0x24,
                 0x00, 0x00,
                 0x80,            # type : code machine
                 0xC7,            # autorun
                 (end >> 8) & 0xFF, end & 0xFF,
                 (start >> 8) & 0xFF, start & 0xFF,
                 0x00])
    name_field = name.encode("ascii") + b"\x00"
    with open(out_tap, "wb") as f:
        f.write(hdr + name_field + data)
    print(f"OK -> {out_tap} ({len(hdr)+len(name_field)+len(data)} o, "
          f"start ${start:04X} end ${end:04X})")


if __name__ == "__main__":
    main()
