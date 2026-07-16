#!/bin/bash
# E2 — le terminal Oric (firmware, dans oric1-emu) se connecte au serveur de
# PRODUCTION via le modem émulé (chemin fidèle au matériel réel : Oric + modem
# WiFi qui compose l'hôte du ATD), puis navigue jusqu'au catalogue. Capture d'écran.
#
# C'est la meilleure validation possible sans Oric physique : firmware réel +
# séquence AT + chaîne réseau -> BBS déployé.
#
# Prérequis : oric1-emu + ROM (Oric1/). Réseau sortant vers l'hôte du BBS.
# Usage : scripts/test-emulateur-prod.sh [host] [port]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
HOST="${1:-pavi.3617.fr}"
PORT="${2:-6502}"
TAP="$ROOT/client/term.tap"
OUT="/tmp/e2-prod.ppm"

[ -x "$EMU" ] || { echo "émulateur introuvable : $EMU (définir ORIC_EMU)"; exit 1; }
[ -f "$TAP" ] || { echo "term.tap manquant — lancer 'make client'"; exit 1; }

# Le terminal démarre : menu modem (1=ACIA $031C) -> répertoire (1 = pavi.3617.fr:6502)
# -> ATD/connexion -> accueil BBS -> 3 (invité) -> espace -> menu principal ->
# 8 (Catalogue) -> 1 (Logiciels). Une touche par évènement --type-keys, espacées ;
# --realtime cadence la série réseau et rend le typing déterministe. Marge large
# pour la connexion internet + poignée de main AT.
KEYS=(
  --type-keys "7500000:1"    # menu modem -> ACIA $031C
  --type-keys "10000000:1"   # répertoire -> compose l'entrée 1 (prod)
  --type-keys "22000000:3"   # accueil BBS -> invité (après connexion)
  --type-keys "25000000: "   # « appuyez sur une touche » -> menu principal
  --type-keys "28000000:8"   # menu principal -> Catalogue
  --type-keys "31000000:1"   # Catalogue -> Logiciels
)
SHOT=34000000
STOP=35000000

echo "E2 : terminal Oric (émulé) -> $HOST:$PORT (modem), capture $OUT"
"$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "modem:$HOST:$PORT" --serial-buffer 512 \
    --realtime --headless \
    "${KEYS[@]}" \
    --screenshot-at "$SHOT:$OUT" -c "$STOP" 2>&1 | tail -5

if command -v pnmtopng >/dev/null; then
    pnmtopng "$OUT" > "${OUT%.ppm}.png" && echo "PNG -> ${OUT%.ppm}.png"
fi
