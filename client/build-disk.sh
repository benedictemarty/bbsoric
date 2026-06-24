#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# build-disk.sh — fabrique une disquette Sedoric contenant le terminal Oric,
# pour le déploiement « voie B » (terminal sous Sedoric résident, avec
# sauvegarde disquette des fichiers reçus).
#
# Principe (validé end-to-end dans l'émulateur, cf. docs/sedoric-api.md) :
#   1. assemble le terminal (term.bin, ML $1000-$1E26) via build.sh
#   2. fabrique une cassette NON-autorun du terminal (octet autorun $C7 -> $00)
#   3. pilote oric1-emu : boot Sedoric (disquette master) + fast-load de la
#      cassette (le terminal est injecté en RAM $1000 sans CLOAD), puis
#      SAVE"TERM",A#1000,E#1E26 -> écrit TERM.COM sur une COPIE du master
#   4. --disk-writeback persiste la disquette : <out>.dsk
#
# La disquette produite boote Sedoric ; lancer le terminal par :
#     LOAD"TERM":CALL#1000
# (choisir l'ACIA « LOCI $03A0 » au menu pour cohabiter avec le Microdisc —
#  $031C entre en conflit avec la plage I/O Microdisc $0310-$031F).
#
# Prérequis : oric1-emu + ROMs + une disquette Sedoric master.
#   ORIC_EMU     binaire émulateur     (défaut $HOME/Oric1/oric1-emu)
#   ORIC_ROM     ROM BASIC Atmos       (défaut $HOME/Oric1/roms/basic11b.rom)
#   ORIC_DISKROM ROM Microdisc         (défaut $HOME/Oric1/roms/microdis.rom)
#   SEDORIC_DSK  disquette Sedoric master (défaut $HOME/Oric1/disks/sedoric3.dsk)
# Usage : client/build-disk.sh [sortie.dsk]   (défaut client/term-boot.dsk)
# ---------------------------------------------------------------------------
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
DISKROM="${ORIC_DISKROM:-$HOME/Oric1/roms/microdis.rom}"
MASTER="${SEDORIC_DSK:-$HOME/Oric1/disks/sedoric3.dsk}"
OUT="${1:-$HERE/term-boot.dsk}"

for f in "$EMU" "$ROM" "$DISKROM" "$MASTER"; do
	[ -e "$f" ] || { echo "ERREUR : introuvable : $f"; exit 1; }
done

echo "--- 1. Assemblage du terminal ---"
bash "$HERE/build.sh" >/dev/null
[ -f "$HERE/term.bin" ] || { echo "ERREUR : term.bin manquant."; exit 1; }
echo "  term.bin : $(stat -c%s "$HERE/term.bin") octets"

echo "--- 2. Cassette non-autorun ---"
TAP="$(mktemp --suffix=.tap)"
trap 'rm -f "$TAP"' EXIT
python3 - "$HERE/term.tap" "$TAP" <<'PY'
import sys
src, dst = sys.argv[1], sys.argv[2]
d = bytearray(open(src, "rb").read())
d[7] = 0x00          # octet autorun $C7 -> $00 : charge sans exécuter
open(dst, "wb").write(bytes(d))
PY

echo "--- 3. Copie du master + écriture de TERM.COM via Sedoric ---"
cp "$MASTER" "$OUT"
# Boot Sedoric (~9M) ; le fast-load injecte le terminal en $1000 vers 3M et il
# survit au boot. Au prompt (~13M) : SAVE du terminal en fichier Sedoric.
"$EMU" -n -r "$ROM" --disk-rom "$DISKROM" -d "$OUT" -t "$TAP" -f \
	--disk-writeback -c 40000000 \
	--type-keys '13000000:\n\p1SAVE"TERM",A#1000,E#1E26\n\p8' >/dev/null 2>&1

echo "--- 4. Vérification ---"
if grep -aboqE 'TERM     COM' "$OUT"; then
	echo "  OK : TERM.COM présent dans $(basename "$OUT")"
else
	echo "  ERREUR : TERM.COM absent — la sauvegarde Sedoric a échoué."
	exit 1
fi

echo ""
echo "=== Disquette bootable produite : $OUT ==="
echo "    Boot Sedoric, puis lancer le terminal :  LOAD\"TERM\":CALL#1000"
echo "    (au menu modem, choisir LOCI \$03A0 si Microdisc présent)"
