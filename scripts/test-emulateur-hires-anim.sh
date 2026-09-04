#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-emulateur-hires-anim.sh — preuve runtime de l'ANIMATION HIRES via le
# buffer différentiel (oascii.HiresScreen + applet hiresanim).
#
#   1. bbsd sert docs/examples/hires-anim.json (entrée 1 -> applet hiresanim :
#      une balle rebondit dans un cadre fixe ; seule la balle est réémise).
#   2. le VRAI firmware Oric (client/term.tap) boote dans oric1-emu, se connecte,
#      navigue jusqu'à l'accueil et lance l'animation (touche « 1 »).
#   3. on ÉCHANTILLONNE la VRAM HIRES ($A000, 8000 o) tout au long de la session et
#      on vérifie qu'elle prend PLUSIEURS états substantiels distincts : le vrai
#      firmware rend bien le flux différentiel et l'écran vit (animation). Note : le
#      timing série ne permet pas de capter deux images-sprite propres de façon
#      fiable ; la correction du différentiel est prouvée de façon DÉTERMINISTE par
#      les tests unitaires (round-trip + séquence) d'internal/oascii.
#
# Prérequis (skip propre si absents) : xa, go, oric1-emu + ROM, client/term.tap.
# Code de sortie : 0 = vert (ou skip), 1 = échec.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
TAP="$ROOT/client/term.tap"
CONTENT="$ROOT/docs/examples/hires-anim.json"
PORT="${BBS_PORT:-6596}"

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
/tmp/bbsd -addr "127.0.0.1:$PORT" -content "$CONTENT" -idle 120s >/tmp/bbsd-anim.log 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null || true' EXIT
sleep 0.5

# Navigation (cf. test-emulateur-hires.sh) : modem « 1 » (ACIA $031C), répertoire
# « m », host/port, proto « 1 », puis accueil « 1 » -> lance l'applet hiresanim.
KEYS=(--type-keys "7500000:1" --type-keys "9500000:m")
c=11000000
addkey() { KEYS+=(--type-keys "$c:$1"); c=$((c + 700000)); }
host="127.0.0.1"; for ((i=0; i<${#host}; i++)); do addkey "${host:$i:1}"; done
KEYS+=(--type-keys "$c:"$'\n'); c=$((c + 1500000))
portseq=""; for ((i=0; i<${#PORT}; i++)); do portseq+="${PORT:$i:1}\\p1"; done
KEYS+=(--type-keys "$c:"$portseq$'\n'); c=$((c + 7000000))
KEYS+=(--type-keys "$c:1"); c=$((c + 3000000))   # proto telnet -> connexion
KEYS+=(--type-keys "$c:1"); c=$((c + 1000000))   # accueil -> lance l'animation
# Le timing émulateur/serveur (temps réel + time.Sleep) dérive de plusieurs
# secondes : plutôt que viser deux instants précis, on ÉCHANTILLONNE densément
# toute la fenêtre d'animation (~10 s) et on cherche a posteriori deux images
# PLEINES qui diffèrent (balle déplacée).
DUMP_DIR="/tmp/oric-anim"; rm -rf "$DUMP_DIR"; mkdir -p "$DUMP_DIR"
DUMPARGS=()
for k in $(seq 0 19); do
    DUMPARGS+=(--dump-ram-at "$c:$DUMP_DIR/f$(printf '%02d' "$k").bin")
    c=$((c + 600000))   # ~0,6 s entre dumps, 20 dumps -> ~12 s couverts
done
STOP=$((c + 1000000))

SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy \
  "$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "modem:127.0.0.1:$PORT" --serial-buffer 512 \
    --headless --realtime \
    "${KEYS[@]}" \
    "${DUMPARGS[@]}" \
    -c "$STOP" >/tmp/emu-anim.log 2>&1

ls "$DUMP_DIR"/*.bin >/dev/null 2>&1 || { ko "aucun dump RAM (voir /tmp/emu-anim.log)"; bilan; exit 1; }

# Analyse (runtime, best-effort) : le timing série/émulateur ne permet pas de
# capter deux images-sprite PROPRES de façon fiable (dumps à mi-écriture). On
# prouve ici le point robuste : le VRAI firmware REND le flux différentiel et la
# VRAM prend PLUSIEURS états distincts non triviaux (l'écran vit = animation). La
# preuve rigoureuse de la CORRECTION du différentiel est déterministe (tests
# unitaires round-trip + séquence d'animation dans internal/oascii).
python3 - "$DUMP_DIR" <<'PY'
import sys, glob, os
A000, N = 0xA000, 8000
def vram(p):
    d = open(p, 'rb').read()
    return d[A000:A000+N]
def lit(v):  # octets avec au moins un pixel (bits 0-5) allumé
    return sum(1 for x in v if (x & 0x3F) != 0)
frames = []
for p in sorted(glob.glob(os.path.join(sys.argv[1], "*.bin"))):
    v = vram(p); n = lit(v)
    print(f"  {os.path.basename(p)} : {n} pixels")
    if n > 60:                       # image pleine (2 rails ~80 o + sprite ~30 o)
        frames.append(v)
if len(frames) < 2:
    print(f"ECHEC: moins de 2 etats HIRES substantiels captures ({len(frames)})"); sys.exit(1)
# Au moins deux états HIRES substantiels distincts (la VRAM évolue = animation) ?
moved = any(frames[i] != frames[j] for i in range(len(frames)) for j in range(i+1, len(frames)))
if not moved:
    print("ECHEC: etats HIRES tous identiques (ecran statique, pas d'animation)"); sys.exit(1)
print(f"OK: {len(frames)} etats HIRES substantiels, au moins deux distincts (VRAM evolue)")
PY
if [ $? -eq 0 ]; then
    ok "firmware rend le flux differentiel ; la VRAM evolue (animation)"
else
    ko "animation non prouvee (voir sortie ci-dessus + /tmp/emu-anim.log)"
fi

bilan
