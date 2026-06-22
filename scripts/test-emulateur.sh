#!/usr/bin/env bash
# Test d'intégration de bout en bout :
#   1. compile et lance le serveur BBS Oric
#   2. lance oric1-emu (Phosphoric) en headless, connecté en série TCP au BBS
#   3. prend une capture d'écran (PPM) du rendu OASCII
#
# Prérequis : Go, xa, et l'émulateur de référence Oric1/oric1-emu + sa ROM.
# La .tap du terminal doit être construite au préalable : client/build.sh
#
# Usage : scripts/test-emulateur.sh [sortie.ppm]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
TAP="$ROOT/client/term.tap"
PORT="${BBS_PORT:-6502}"
OUT="${1:-/tmp/oric.ppm}"

[ -f "$TAP" ] || { echo "term.tap manquant — lance d'abord client/build.sh"; exit 1; }

go build -o /tmp/bbsd "$ROOT/server/cmd/bbsd"
/tmp/bbsd -addr "127.0.0.1:$PORT" &
SRV=$!
trap 'kill $SRV 2>/dev/null || true' EXIT
sleep 0.5

# Boot ROM + fast-load injectent le terminal vers ~3 M cycles ; on capture à 6.5 M.
SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy \
  "$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "tcp:127.0.0.1:$PORT" --serial-buffer 512 \
    --headless --screenshot-at "6500000:$OUT" -c 7000000

echo "Screenshot -> $OUT"
command -v pnmtopng >/dev/null && pnmtopng "$OUT" > "${OUT%.ppm}.png" && echo "PNG -> ${OUT%.ppm}.png" || true
