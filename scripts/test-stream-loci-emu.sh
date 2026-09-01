#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-stream-loci-emu.sh — validation runtime du STREAMING XMODEM -> LOCI SD
# (histoire I2b : download de fichiers > 32 Ko sans bufferiser tout $4000).
#
# Assemble un harnais 6502 autonome qui :
#   1. initialise l'ACIA 6551 à $031C (backend --serial tcp:) ;
#   2. pose dlname="BIGFILE.BIN" puis ouvre le fichier sur la carte SD (loci_open) ;
#   3. active le mode streaming (xstream=1, xdrem=taille) et appelle xmodem_recv,
#      qui reçoit un fichier de 40000 o en XMODEM et écrit CHAQUE bloc directement
#      sur la carte SD (jamais plus de 128 o en RAM), tronquant le padding final.
# Un émetteur XMODEM en Go (internal/xmodem.Send) envoie le fichier via TCP.
# On vérifie ensuite que le fichier hôte écrit par le LOCI est OCTET-IDENTIQUE
# à la source (40000 o, taille NON multiple de 128 -> teste la troncature).
#
# C'est la preuve que le plafond ~30 Ko du buffer $4000 est levé : 40000 > 30720.
#
# Dépendances (ignorées proprement si absentes) : xa (xa65), go, oric1-emu + ROM.
# Réglables via ORIC_EMU / ORIC_ROM. Code de sortie : 0 = vert (ou skip), 1 = échec.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
XMODEM_S="$ROOT/client/xmodem.s"
LOCI_S="$ROOT/client/loci.s"
BIN2TAP="$ROOT/client/bin2tap.py"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
CYCLES="${STREAM_TEST_CYCLES:-90000000}"
PORT="${STREAM_TEST_PORT:-6540}"
SIZE="${STREAM_TEST_SIZE:-40000}"
TRACE_OPT=()
[ -n "${STREAM_TEST_TRACE:-}" ] && TRACE_OPT=(--serial-trace "$STREAM_TEST_TRACE")

PASS=0; FAIL=0; SKIP=0
ok()   { echo "  ok   : $*"; PASS=$((PASS + 1)); }
ko()   { echo "  FAIL : $*"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP : $*"; SKIP=$((SKIP + 1)); }
bilan(){ echo; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; [ "$FAIL" -eq 0 ]; }

command -v xa      >/dev/null || { skip "xa absent (apt-get install xa65)"; bilan; exit 0; }
command -v go      >/dev/null || { skip "go absent"; bilan; exit 0; }
command -v python3 >/dev/null || { skip "python3 absent"; bilan; exit 0; }
[ -x "$EMU" ] || { skip "oric1-emu absent ($EMU) — règle ORIC_EMU"; bilan; exit 0; }
[ -f "$ROM" ] || { skip "ROM absente ($ROM) — règle ORIC_ROM"; bilan; exit 0; }

# Répertoire de travail SOUS le dépôt (le helper Go importe internal/xmodem, donc
# doit vivre dans l'arbre du module).
WORK="$(mktemp -d "$ROOT/.stream-test.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

# --- Fichier source (motif déterministe, 40000 o, non multiple de 128) ---------
python3 - "$WORK/src.bin" "$SIZE" <<'PY'
import sys
path, n = sys.argv[1], int(sys.argv[2])
open(path, 'wb').write(bytes((i * 37 + 11) & 0xFF for i in range(n)))
PY

# --- Émetteur XMODEM Go (listen TCP, accept, Send) -----------------------------
cat > "$WORK/xsend.go" <<'GO'
package main

import (
	"net"
	"os"
	"time"

	"github.com/benedictemarty/bbsoric/internal/xmodem"
)

func main() {
	addr, file := os.Args[1], os.Args[2]
	data, err := os.ReadFile(file)
	if err != nil {
		os.Exit(3)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		os.Exit(4)
	}
	// L'émulateur (client tcp:) se connecte ; on lui envoie le fichier.
	conn, err := ln.Accept()
	if err != nil {
		os.Exit(5)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(100 * time.Second))
	if err := xmodem.Send(conn, data); err != nil {
		os.Exit(6)
	}
	time.Sleep(500 * time.Millisecond) // laisse l'EOT/ACK se vider
}
GO

# --- Harnais 6502 autonome (pilote xmodem_recv en streaming) -------------------
# Carte zéro-page alignée sur term.s/xmodem.s/loci.s.
cat > "$WORK/harness.s" <<'ASM'
        * = $1000
; Equates NON definies par xmodem.s (qui pose XBUF/XBLK/XSUM/XSIZE/XCRC/XREM/XSAVY).
SCRPTR  = $F0
COL     = $F2
SRC     = $F4
KTMP    = $FC
STRPTR  = $EE
ACIAPTR = $EA
XTOTAL  = $E8
XACC    = $E3
XSEG    = $E5
SCREEN  = $BB80
RDRF    = $08
TDRE    = $10

start:
        sei
        lda #$1C                 ; ACIAPTR = $031C (ACIA 6551, backend serie brut)
        sta ACIAPTR
        lda #$03
        sta ACIAPTR+1
        lda #$1E                 ; ACIA 9600 8N1, DTR on, IRQ off, TX on
        ldy #3
        sta (ACIAPTR),y
        lda #$0B
        ldy #2
        sta (ACIAPTR),y
        ldx #11                  ; dlname = "BIGFILE  BIN"
h_nm:
        lda h_name,x
        sta dlname,x
        dex
        bpl h_nm
        jsr loci_open            ; ouvre le fichier SD (A=1 si ok)
        sta $9001                ; marqueur open
        lda #1
        sta xstream              ; mode streaming
        lda #1
        sta xsink                ; evier = LOCI
        lda #0
        sta XTOTAL               ; jauge desactivee (pas d'ecran a piloter)
        sta XTOTAL+1
        lda h_size               ; xdrem = taille reelle (40000)
        sta xdrem
        lda h_size+1
        sta xdrem+1
        jsr xmodem_recv          ; recoit + streame chaque bloc sur SD + ferme
        sta $9000                ; marqueur succes (1) / echec (0)
h_done:
        jmp h_done
h_name:
        .byt "BIGFILE  BIN"
h_size:
        .byt <SIZEVAL,>SIZEVAL
; --- stubs (routines term.s non embarquees ; inertes ici) ---
print_string:
        rts
putbyte:
        rts
sed_save:
        lda #0
        rts
set_slice_ext:                   ; inerte ici (chemin SED non exerce par ce harnais)
        rts
; --- primitives serie (copie fidele de term.s) ---
ser_tx:
        pha
stx_wait:
        ldy #1
        lda (ACIAPTR),y
        and #TDRE
        beq stx_wait
        pla
        ldy #0
        sta (ACIAPTR),y
        rts
ser_rx_ready:
        ldy #1
        lda (ACIAPTR),y
        and #RDRF
        rts
ser_rx:
        ldy #0
        lda (ACIAPTR),y
        rts
dlname:
        .dsb 12,0
ASM

sed -i "s/SIZEVAL/$SIZE/g" "$WORK/harness.s"
cat "$WORK/harness.s" "$XMODEM_S" "$LOCI_S" > "$WORK/full.s"
if ! xa "$WORK/full.s" -o "$WORK/test.bin" 2>"$WORK/xa.err"; then
    ko "assemblage du harnais streaming (voir $WORK/xa.err)"; cat "$WORK/xa.err"; bilan; exit 1
fi
python3 "$BIN2TAP" "$WORK/test.bin" 0x1000 STREAM "$WORK/test.tap" >/dev/null
ok "harnais streaming assemblé + tap généré"

# --- Lancement : émetteur Go (background) puis émulateur ------------------------
FLASH="$WORK/flash"; mkdir -p "$FLASH"
( cd "$ROOT" && go run "$WORK/xsend.go" "127.0.0.1:$PORT" "$WORK/src.bin" ) \
    >"$WORK/xsend.log" 2>&1 &
GOPID=$!
sleep 2   # laisse le listener s'ouvrir avant que l'ACIA ne se connecte

SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy timeout 250 \
  "$EMU" -t "$WORK/test.tap" -f -r "$ROM" \
    --serial "tcp:127.0.0.1:$PORT" --serial-buffer 512 --acia-addr 031C \
    --headless --realtime -c "$CYCLES" "${TRACE_OPT[@]}" \
    --loci-flash "$FLASH" >"$WORK/emu.log" 2>&1
EMU_RC=$?
wait "$GOPID" 2>/dev/null; GO_RC=$?

if [ "$EMU_RC" -ne 0 ]; then
    ko "émulateur en échec (rc=$EMU_RC, voir $WORK/emu.log)"; bilan; exit 1
fi
if [ "$GO_RC" -ne 0 ]; then
    ko "émetteur XMODEM Go en échec (rc=$GO_RC, voir $WORK/xsend.log)"; bilan; exit 1
fi

# --- Vérification : fichier hôte octet-identique à la source (40000 o) ---------
if [ ! -f "$FLASH/BIGFILE.BIN" ]; then
    ko "BIGFILE.BIN absent (fichiers : $(ls "$FLASH" 2>/dev/null | tr '\n' ' '))"; bilan; exit 1
fi
if python3 - "$FLASH/BIGFILE.BIN" "$WORK/src.bin" <<'PY'
import sys
got = open(sys.argv[1], 'rb').read()
want = open(sys.argv[2], 'rb').read()
if got == want:
    sys.exit(0)
print(f"    tailles got={len(got)} want={len(want)}")
sys.exit(1)
PY
then
    if [ "$SIZE" -gt 30720 ]; then
        ok "BIGFILE.BIN streamé sur SD, $SIZE o identiques (buffer 30 Ko dépassé)"
    else
        ok "BIGFILE.BIN streamé sur SD, $SIZE o identiques (mécanisme streaming)"
    fi
else ko "BIGFILE.BIN présent mais contenu != source"
fi

bilan
