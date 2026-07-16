#!/bin/bash
# E2 — le terminal Oric (firmware, dans oric1-emu) se connecte au serveur de
# PRODUCTION via le modem émulé, puis navigue le catalogue. Deux chemins :
#   acia  : ACIA 6551 standard $031C  (--serial modem:H:P), menu modem « 1 »
#   loci  : LOCI + Pico W WiFi $0380  (--loci --serial picowifi), menu modem « 2 »
#           -> le plus fidèle au matériel réel (Oric + LOCI + WiFiModem)
#
# Meilleure validation possible sans Oric physique : firmware réel + séquence AT +
# chaîne réseau -> BBS déployé. Capture plusieurs écrans le long du parcours.
#
# Usage : scripts/test-emulateur-prod.sh [acia|loci] [host] [port]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
MODE="${1:-acia}"
HOST="${2:-pavi.3617.fr}"
PORT="${3:-6502}"
TAP="$ROOT/client/term.tap"
PREFIX="/tmp/e2-$MODE"

[ -x "$EMU" ] || { echo "émulateur introuvable : $EMU (définir ORIC_EMU)"; exit 1; }
[ -f "$TAP" ] || { echo "term.tap manquant — lancer 'make client'"; exit 1; }

case "$MODE" in
    acia) TRANSPORT=(--serial "modem:$HOST:$PORT"); MODEMKEY="1" ;;
    loci) TRANSPORT=(--loci --serial picowifi);     MODEMKEY="2" ;;
    *) echo "mode inconnu : $MODE (acia|loci)"; exit 2 ;;
esac

# Parcours : menu modem -> répertoire (1 = pavi.3617.fr:6502) -> ATD/connexion ->
# accueil -> 3 (invité) -> espace -> menu -> 8 (Catalogue) -> 1 (Logiciels) ->
# V (fiche détail). Une touche par évènement --type-keys, marges larges pour la
# connexion internet + poignée de main AT. --realtime rend le timing déterministe.
KEYS=(
  --type-keys "7500000:$MODEMKEY"   # menu modem -> backend série
  --type-keys "10000000:1"          # répertoire -> compose l'entrée 1 (prod)
  --type-keys "22000000:3"          # accueil BBS -> invité
  --type-keys "25000000: "          # « une touche » -> menu principal
  --type-keys "28000000:8"          # menu principal -> Catalogue
)
# Captures échelonnées : menu Catalogue, grille Logiciels, fiche détail.
SHOTS=(
  --screenshot-at "29500000:$PREFIX-1-catalogue.ppm"
)
KEYS+=(--type-keys "30500000:1")                                   # Catalogue -> Logiciels
SHOTS+=(--screenshot-at "33000000:$PREFIX-2-logiciels.ppm")
KEYS+=(--type-keys "34000000:v")                                   # Logiciels -> fiche détail
SHOTS+=(--screenshot-at "37000000:$PREFIX-3-fiche.ppm")
STOP=38000000

echo "E2 [$MODE] : terminal Oric (émulé) -> $HOST:$PORT"
"$EMU" -t "$TAP" -f -r "$ROM" \
    "${TRANSPORT[@]}" --serial-buffer 512 \
    --realtime --headless \
    "${KEYS[@]}" "${SHOTS[@]}" -c "$STOP" 2>&1 | tail -4

if command -v pnmtopng >/dev/null; then
    for f in "$PREFIX"-*.ppm; do
        [ -f "$f" ] && pnmtopng "$f" > "${f%.ppm}.png" && echo "PNG -> ${f%.ppm}.png"
    done
fi
