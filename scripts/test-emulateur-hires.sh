#!/usr/bin/env bash
# Preuve « identique au client Oric » du rasteriseur HIRES d'oterm :
#   1. bbsd sert la page HIRES `logo` (docs/examples/hires-demo.json)
#   2. le VRAI firmware Oric (client/term.tap) boote dans oric1-emu, se connecte
#      en série, navigue jusqu'à la page HIRES, la rasterise dans sa VRAM $A000
#   3. on dumpe la RAM 64 Ko et on compare $A000 (8000 o) au rasteriseur Go via
#      le test gaté TestPixelExactVsEmulateur.
#
# Prérequis : Go, l'émulateur Oric1/oric1-emu + ROM, client/term.tap (client/build.sh).
# Usage : scripts/test-emulateur-hires.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
TAP="$ROOT/client/term.tap"
CONTENT="${BBS_CONTENT:-$ROOT/docs/examples/hires-demo.json}"
PORT="${BBS_PORT:-6595}"
DUMP="${ORIC_RAM_DUMP:-/tmp/oric-ram-hires.bin}"

[ -f "$TAP" ] || { echo "term.tap manquant — lance d'abord client/build.sh"; exit 1; }
[ -x "$EMU" ] || { echo "émulateur introuvable : $EMU (ORIC_EMU pour surcharger)"; exit 1; }

go build -o /tmp/bbsd "$ROOT/server/cmd/bbsd"
/tmp/bbsd -addr "127.0.0.1:$PORT" -content "$CONTENT" -idle 120s >/tmp/bbsd-hires.log 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null || true' EXIT
sleep 0.5

# Navigation (cf. test-emulateur-grille.sh) : menu modem « 1 » (ACIA $031C),
# répertoire « m » (saisie manuelle), host/port/proto, puis accueil « 1 » qui
# ouvre la page HIRES `logo`. Dump RAM ~4M cycles plus tard.
KEYS=(--type-keys "7500000:1" --type-keys "9500000:m")
c=11000000
addkey() { KEYS+=(--type-keys "$c:$1"); c=$((c + 700000)); }
host="127.0.0.1"; for ((i=0; i<${#host}; i++)); do addkey "${host:$i:1}"; done
KEYS+=(--type-keys "$c:"$'\n'); c=$((c + 1500000))
portseq=""; for ((i=0; i<${#PORT}; i++)); do portseq+="${PORT:$i:1}\\p1"; done
KEYS+=(--type-keys "$c:"$portseq$'\n'); c=$((c + 7000000))
KEYS+=(--type-keys "$c:1"); c=$((c + 3000000))   # proto telnet -> connexion
KEYS+=(--type-keys "$c:1"); c=$((c + 4000000))   # accueil -> page HIRES logo
DUMPC=$c; STOP=$((c + 2000000))

SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy \
  "$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "modem:127.0.0.1:$PORT" --serial-buffer 512 \
    --headless --realtime \
    "${KEYS[@]}" \
    --dump-ram-at "$DUMPC:$DUMP" -c "$STOP" >/tmp/emu-hires.log 2>&1
echo "RAM dump -> $DUMP ($(stat -c%s "$DUMP" 2>/dev/null || echo 0) octets)"

# Comparaison au rasteriseur Go (test gaté).
ORIC_RAM_DUMP="$DUMP" ORIC_HIRES_JSON="$CONTENT" ORIC_HIRES_PAGE="logo" \
  go test "$ROOT/pcterm/internal/hires/" -run TestPixelExactVsEmulateur -v
