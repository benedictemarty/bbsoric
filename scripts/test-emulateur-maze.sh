#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-emulateur-maze.sh — preuve runtime que le jeu LABYRINTHE HIRES (applet
# maze, rendu via oascii.HiresScreen + writeHiresPaced) s'affiche sur le VRAI
# firmware Oric.
#
#   1. bbsd sert docs/examples/maze.json (entree 1 -> applet maze).
#   2. client/term.tap boote dans oric1-emu, se connecte, navigue, lance le jeu.
#   3. le labyrinthe (statique tant qu'aucune touche n'est envoyee) est dessine ;
#      on dumpe la VRAM et on verifie un contenu HIRES SUBSTANTIEL (les murs =
#      beaucoup de traits) dans la zone HIRES-only $A000..$BB80.
#
# Prerequis (skip propre si absents) : xa, go, oric1-emu + ROM, client/term.tap.
# Code de sortie : 0 = vert (ou skip), 1 = echec.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
TAP="$ROOT/client/term.tap"
CONTENT="$ROOT/docs/examples/maze.json"
PORT="${BBS_PORT:-6601}"
DUMP="/tmp/oric-maze.bin"

PASS=0; FAIL=0; SKIP=0
ok()   { echo "  ok   : $*"; PASS=$((PASS + 1)); }
ko()   { echo "  FAIL : $*"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP : $*"; SKIP=$((SKIP + 1)); }
bilan(){ echo; echo "Bilan: $PASS ok, $FAIL ko, $SKIP skip"; [ "$FAIL" -eq 0 ]; }

command -v go  >/dev/null || { skip "go absent"; bilan; exit 0; }
[ -x "$EMU" ] || { skip "oric1-emu absent ($EMU)"; bilan; exit 0; }
[ -f "$ROM" ] || { skip "ROM absente ($ROM)"; bilan; exit 0; }
[ -f "$TAP" ] || { skip "term.tap absent — 'make client'"; bilan; exit 0; }

go build -o /tmp/bbsd "$ROOT/server/cmd/bbsd" || { ko "build bbsd"; bilan; exit 1; }
/tmp/bbsd -addr "127.0.0.1:$PORT" -content "$CONTENT" -idle 120s >/tmp/bbsd-maze.log 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null || true' EXIT
sleep 0.5

# Navigation identique aux autres tests HIRES : modem 1, repertoire m, host/port,
# proto 1, puis accueil 1 -> lance le labyrinthe (laisser connexion + rendu paced).
KEYS=(--type-keys "7500000:1" --type-keys "9500000:m")
c=11000000
addkey() { KEYS+=(--type-keys "$c:$1"); c=$((c + 700000)); }
host="127.0.0.1"; for ((i=0; i<${#host}; i++)); do addkey "${host:$i:1}"; done
KEYS+=(--type-keys "$c:"$'\n'); c=$((c + 1500000))
portseq=""; for ((i=0; i<${#PORT}; i++)); do portseq+="${PORT:$i:1}\\p1"; done
KEYS+=(--type-keys "$c:"$portseq$'\n'); c=$((c + 7000000))
KEYS+=(--type-keys "$c:1"); c=$((c + 3000000))   # proto telnet -> connexion
KEYS+=(--type-keys "$c:1"); c=$((c + 12000000))  # accueil -> lance le labyrinthe ; laisser le rendu paced
DUMPC=$c; STOP=$((c + 2000000))

SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy \
  "$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "modem:127.0.0.1:$PORT" --serial-buffer 512 \
    --headless --realtime \
    "${KEYS[@]}" \
    --dump-ram-at "$DUMPC:$DUMP" -c "$STOP" >/tmp/emu-maze.log 2>&1

[ -s "$DUMP" ] || { ko "dump RAM manquant (voir /tmp/emu-maze.log)"; bilan; exit 1; }

# Contenu HIRES-only $A000..$BB80 : les murs du labyrinthe = beaucoup de pixels.
python3 - "$DUMP" <<'PY'
import sys
BASE, HIONLY = 0xA000, 0x1B80
d = open(sys.argv[1], 'rb').read()
v = d[BASE:BASE+HIONLY]
lit = sum(1 for x in v if (x & 0x3F) != 0)
print(f"pixels HIRES-only = {lit}")
if lit < 200:
    print("ECHEC: pas de labyrinthe rendu (contenu HIRES insuffisant)"); sys.exit(1)
print("OK: labyrinthe rendu (murs presents dans la VRAM)")
PY
if [ $? -eq 0 ]; then
    ok "labyrinthe HIRES rendu par le vrai firmware (flux differentiel cadence)"
else
    ko "labyrinthe non prouve (voir sortie + /tmp/emu-maze.log)"
fi

bilan
