#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-loci-emu.sh — validation runtime de loci_save (sauvegarde carte SD LOCI).
#
# Assemble un harnais 6502 autonome qui remplit $4000 avec 0..255 puis sauve les 200 premiers,
# pose dlname="TEST.BIN" et appelle loci_save, puis l'exécute dans oric1-emu et
# vérifie le fichier écrit :
#   1. backend --loci-flash  : fichier hôte TEST.BIN identique octet par octet ;
#   2. backend --loci-sdimg  : image FAT16 contenant les données + l'entrée 8.3
#      (vrai chemin d'écriture FAT ; ignoré si mkfs.vfat absent).
#
# Dépendances externes (ignorées proprement si absentes) : xa (xa65), oric1-emu
# + ROM Atmos. Réglables via ORIC_EMU / ORIC_ROM.
#
# Code de sortie : 0 = tous les cas verts (ou ignorés faute d'outils), 1 = échec.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOCI_S="$ROOT/client/loci.s"
BIN2TAP="$ROOT/client/bin2tap.py"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
CYCLES="${LOCI_TEST_CYCLES:-12000000}"

PASS=0; FAIL=0; SKIP=0
ok()   { echo "  ok   : $*"; PASS=$((PASS + 1)); }
ko()   { echo "  FAIL : $*"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP : $*"; SKIP=$((SKIP + 1)); }

# --- Prérequis -------------------------------------------------------------
command -v xa       >/dev/null || { skip "xa absent (apt-get install xa65)"; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; exit 0; }
command -v python3  >/dev/null || { skip "python3 absent"; exit 0; }
[ -x "$EMU" ] || { skip "oric1-emu absent ($EMU) — règle ORIC_EMU"; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; exit 0; }
[ -f "$ROM" ] || { skip "ROM absente ($ROM) — règle ORIC_ROM"; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; exit 0; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# --- Harnais 6502 autonome (appelle loci_save directement) -----------------
cat > "$WORK/harness.s" <<'ASM'
; Harnais 6502 autonome - remplit 4000 (0..255), XSIZE=200, appelle loci_save.
        * = $1000
XSIZE  = $FE
SRC    = $F4
STRPTR = $EE
start:
        ldx #0
tl_fill:
        txa
        sta $4000,x
        inx
        bne tl_fill
        lda #200
        sta XSIZE
        lda #0
        sta XSIZE+1             ; 200 octets = 1 bloc plein 128 + 1 partiel 72
        ldx #11
tl_nm:
        lda tl_name,x
        sta dlname,x
        dex
        bpl tl_nm
        jsr loci_save
        sta $9000               ; marqueur (1=sauve)
tl_done:
        jmp tl_done
tl_name:
        .byt "TEST     BIN"
print_string:
        rts
sed_save:
        lda #0
        rts
dlname:
        .dsb 12,0
ASM

cat "$WORK/harness.s" "$LOCI_S" > "$WORK/full.s"
if ! xa "$WORK/full.s" -o "$WORK/test.bin" 2>"$WORK/xa.err"; then
    ko "assemblage du harnais (voir $WORK/xa.err)"; cat "$WORK/xa.err"
    echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; exit 1
fi
python3 "$BIN2TAP" "$WORK/test.bin" 0x1000 TESTLOCI "$WORK/test.tap" >/dev/null
ok "harnais assemblé + tap généré"

run_emu() {  # $@ = options émulateur supplémentaires
    SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy timeout 90 \
      "$EMU" -t "$WORK/test.tap" -f -r "$ROM" --headless -c "$CYCLES" "$@" \
      >"$WORK/emu.log" 2>&1
}

# === Cas 1 — backend --loci-flash (passthrough hôte) =======================
echo "Cas 1 — --loci-flash"
FLASH="$WORK/flash"; mkdir -p "$FLASH"
if run_emu --loci-flash "$FLASH"; then
    if [ -f "$FLASH/TEST.BIN" ]; then
        if python3 - "$FLASH/TEST.BIN" <<'PY'
import sys
data = open(sys.argv[1], 'rb').read()
sys.exit(0 if data == bytes(range(200)) else 1)
PY
        then ok "TEST.BIN écrit, 200 o identiques à 0..199 (bloc partiel)"
        else ko "TEST.BIN présent mais contenu != 0..199"
        fi
    else
        ko "TEST.BIN absent du backend flash"
    fi
else
    ko "émulateur en échec (voir $WORK/emu.log)"
fi

# === Cas 2 — backend --loci-sdimg (vrai chemin FAT16) ======================
echo "Cas 2 — --loci-sdimg (FAT16)"
if command -v mkfs.vfat >/dev/null; then
    IMG="$WORK/sd.img"
    dd if=/dev/zero of="$IMG" bs=1M count=16 status=none
    mkfs.vfat -F 16 -n LOCISD "$IMG" >/dev/null 2>&1
    if run_emu --loci-sdimg "$IMG"; then
        # Sans mtools : on scanne l'image brute pour les données + l'entrée 8.3.
        if python3 - "$IMG" <<'PY'
import sys
img = open(sys.argv[1], 'rb').read()
data_ok = bytes(range(200)) in img            # bloc de donnees 0..199 present
entry_ok = b"TEST    BIN" in img              # entrée de répertoire 8.3 (11 o)
print(f"    data={data_ok} entry={entry_ok}")
sys.exit(0 if (data_ok and entry_ok) else 1)
PY
        then ok "image FAT16 contient les données + l'entrée TEST.BIN"
        else ko "image FAT16 sans données ou sans entrée (voir $WORK/emu.log)"
        fi
    else
        ko "émulateur en échec sur --loci-sdimg (voir $WORK/emu.log)"
    fi
else
    skip "mkfs.vfat absent — cas --loci-sdimg ignoré"
fi

echo
echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"
[ "$FAIL" -eq 0 ]
