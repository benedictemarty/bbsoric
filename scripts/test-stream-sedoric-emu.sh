#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-stream-sedoric-emu.sh — validation runtime de la SAUVEGARDE PAR TRANCHES
# Sedoric (histoire I2b-b : download > 30 Ko sur une machine Sedoric seule, qui
# ne peut pas streamer car XSAVEB exige une région RAM contiguë).
#
# Une machine Sedoric ne stream pas : le récepteur accumule dans $4000 et, à
# chaque remplissage (~30 Ko), sauve une TRANCHE numérotée FICHIER.001, .002…
# recombinée côté hôte. Ce test pilote les VRAIES routines du firmware
# (`xr_sed_write_slice` / `xr_sed_final` de `client/xmodem.s`, `set_slice_ext`
# de `client/term.s`, `sed_save`/`sed_present` de `client/sedoric.s`), extraites
# des sources et assemblées dans un harnais 6502 qui, sous Sedoric RÉSIDENT :
#   1. remplit $4000 puis appelle `xr_sed_write_slice` -> écrit BIGFILE.001 (300 o) ;
#   2. remplit à nouveau puis `xr_sed_final` -> écrit la dernière tranche
#      BIGFILE.002 (150 o, taille tronquée depuis `xdrem`).
# On boote une COPIE de la disquette Sedoric master, `--disk-writeback` persiste,
# puis on vérifie que le `.dsk` contient les DEUX entrées catalogue + les données.
#
# Ceci prouve le mécanisme NOUVEAU (dimensionnement de tranche depuis XBUF,
# nommage .00N, appels sed_save multiples persistés). La décision « quand
# trancher » dans la boucle de réception réutilise la même structure flush-sur-
# événement que le streaming LOCI, déjà prouvée par test-stream-loci-emu.sh.
#
# Dépendances (skip propre si absentes) : xa, python3, oric1-emu + ROMs + master
# Sedoric. Réglables : ORIC_EMU / ORIC_ROM / ORIC_DISKROM / SEDORIC_DSK.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
XMODEM_S="$ROOT/client/xmodem.s"
TERM_S="$ROOT/client/term.s"
SEDORIC_S="$ROOT/client/sedoric.s"
BIN2TAP="$ROOT/client/bin2tap.py"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
DISKROM="${ORIC_DISKROM:-$HOME/Oric1/roms/microdis.rom}"
MASTER="${SEDORIC_DSK:-$HOME/Oric1/disks/sedoric3.dsk}"
CYCLES="${SED_TEST_CYCLES:-40000000}"

PASS=0; FAIL=0; SKIP=0
ok()   { echo "  ok   : $*"; PASS=$((PASS + 1)); }
ko()   { echo "  FAIL : $*"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP : $*"; SKIP=$((SKIP + 1)); }
bilan(){ echo; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; [ "$FAIL" -eq 0 ]; }

command -v xa      >/dev/null || { skip "xa absent"; bilan; exit 0; }
command -v python3 >/dev/null || { skip "python3 absent"; bilan; exit 0; }
[ -x "$EMU" ]     || { skip "oric1-emu absent ($EMU)"; bilan; exit 0; }
[ -f "$ROM" ]     || { skip "ROM BASIC absente ($ROM)"; bilan; exit 0; }
[ -f "$DISKROM" ] || { skip "ROM Microdisc absente ($DISKROM)"; bilan; exit 0; }
[ -f "$MASTER" ]  || { skip "disquette Sedoric master absente ($MASTER)"; bilan; exit 0; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# --- Extraction des VRAIES routines depuis les sources ------------------------
# xr_sed_write_slice + xr_sed_final (bloc contigu, jusqu'à la section gauge).
awk '/^xr_sed_write_slice:/{f=1} f{print} /^xsf_done:/{g=1} g&&/rts/{exit}' "$XMODEM_S" > "$WORK/sed_routines.s"
# set_slice_ext (jusqu'au rts suivant "sta dlname+11").
awk '/^set_slice_ext:/{f=1} f{print} f&&/sta dlname\+11/{g=1} g&&/rts/{exit}' "$TERM_S" > "$WORK/set_ext.s"
if ! grep -q "xr_sed_write_slice" "$WORK/sed_routines.s" || ! grep -q "set_slice_ext" "$WORK/set_ext.s"; then
    ko "extraction des routines depuis les sources"; bilan; exit 1
fi

# --- Harnais 6502 (sous Sedoric résident, lancé par CALL#1000) ----------------
cat > "$WORK/harness.s" <<'ASM'
        * = $1000
XBUF   = $E0
XSIZE  = $FE
STRPTR = $EE
start:
        lda #<300                ; xdrem = 300 (200 + 100) - tranches < 256 o pour
        sta xdrem                ; rester dans un secteur Sedoric (verif hote contigue)
        lda #>300
        sta xdrem+1
        lda #1
        sta slicenum             ; premiere tranche = 001
        ldx #11                  ; dlname = "BIGFILE  XXX" (ext posee par set_slice_ext)
h_nm:
        lda h_name,x
        sta dlname,x
        dex
        bpl h_nm
        lda #$A1                 ; tranche 1 - remplir $4000..$41FF de 0xA1
        ldx #0
h_f1:
        sta $4000,x
        sta $4100,x
        inx
        bne h_f1
        lda #$C8                 ; XBUF = $40C8 -> tranche = 200 octets
        sta XBUF
        lda #$40
        sta XBUF+1
        jsr xr_sed_write_slice   ; -> BIGFILE.001 (200 o), xdrem=100, slicenum=2
        lda #$B2                 ; tranche 2 (finale) - remplir de 0xB2
        ldx #0
h_f2:
        sta $4000,x
        sta $4100,x
        inx
        bne h_f2
        jsr xr_sed_final         ; -> BIGFILE.002 (100 o depuis xdrem)
        rts                      ; retour a Sedoric BASIC
h_name:
        .byt "BIGFILE  XXX"
print_string:
        rts
xdrem:
        .byt 0,0
slicenum:
        .byt 0
dlname:
        .dsb 12,0
ASM

cat "$WORK/harness.s" "$WORK/sed_routines.s" "$WORK/set_ext.s" "$SEDORIC_S" > "$WORK/full.s"
if ! xa "$WORK/full.s" -o "$WORK/h.bin" 2>"$WORK/xa.err"; then
    ko "assemblage du harnais Sedoric (voir ci-dessous)"; cat "$WORK/xa.err"; bilan; exit 1
fi
python3 "$BIN2TAP" "$WORK/h.bin" 0x1000 SEDSLICE "$WORK/h.tap" >/dev/null
# .tap NON-autorun : chargé en $1000 sans exécution, survit au boot Sedoric.
python3 - "$WORK/h.tap" <<'PY'
import sys
d = bytearray(open(sys.argv[1], 'rb').read())
d[7] = 0x00
open(sys.argv[1], 'wb').write(bytes(d))
PY
ok "harnais Sedoric assemblé (routines réelles extraites) + tap non-autorun"

# --- Boot Sedoric + fast-load harnais + CALL#1000 + writeback -----------------
OUT="$WORK/sed.dsk"; cp "$MASTER" "$OUT"
SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy timeout 180 \
  "$EMU" -n -r "$ROM" --disk-rom "$DISKROM" -d "$OUT" -t "$WORK/h.tap" -f \
    --disk-writeback -c "$CYCLES" \
    --type-keys '13000000:\n\p1CALL#1000\n\p8' >"$WORK/emu.log" 2>&1
EMU_RC=$?
if [ "$EMU_RC" -ne 0 ]; then
    ko "émulateur en échec (rc=$EMU_RC, voir $WORK/emu.log)"; bilan; exit 1
fi

# --- Vérification du .dsk : entrées catalogue + données des 2 tranches --------
python3 - "$OUT" <<'PY'
import sys
img = open(sys.argv[1], 'rb').read()
e1 = b"BIGFILE  001"          # entrée catalogue 8.3 (9 nom + 3 ext), 12 o
e2 = b"BIGFILE  002"
d1 = bytes([0xA1]) * 200      # données tranche 1 (< 256 o -> 1 secteur, contiguë)
d2 = bytes([0xB2]) * 100      # données tranche 2
r = {"entree_001": e1 in img, "entree_002": e2 in img,
     "data_001(200xA1)": d1 in img, "data_002(100xB2)": d2 in img}
for k, v in r.items():
    print(f"    {k} = {v}")
sys.exit(0 if all(r.values()) else 1)
PY
if [ $? -eq 0 ]; then
    ok "2 tranches persistées dans le .dsk : BIGFILE.001 (200 o) + BIGFILE.002 (100 o)"
else
    ko "tranches manquantes dans le .dsk (voir $WORK/emu.log)"
fi

bilan
