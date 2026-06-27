#!/usr/bin/env bash
# Capture visuelle de la grille DataWindow rendue dans oric1-emu (terminal Oric
# réel -> série TCP -> BBS local servant le contenu démo `repertoire`).
#
#   1. compile et lance le BBS avec le contenu démo + `-data` (moteur DataWindow)
#   2. boote le terminal dans oric1-emu (--realtime pour la série réseau)
#   3. tape « 1 » (accueil -> grille) puis « + » (descend la sélection)
#   4. prend une capture (PPM -> PNG) de la grille
#
# Prérequis : Go, l'émulateur Oric1/oric1-emu + ROM, et client/term.tap (client/build.sh).
# Usage : scripts/test-emulateur-grille.sh [sortie.ppm]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EMU="${ORIC_EMU:-$HOME/Oric1/oric1-emu}"
ROM="${ORIC_ROM:-$HOME/Oric1/roms/basic11b.rom}"
TAP="$ROOT/client/term.tap"
CONTENT="${BBS_CONTENT:-$ROOT/docs/examples/datawindow-demo.json}"
PORT="${BBS_PORT:-6502}"
OUT="${1:-/tmp/oric-grille.ppm}"
DATA="$(mktemp -d)"

[ -f "$TAP" ] || { echo "term.tap manquant — lance d'abord client/build.sh"; exit 1; }
[ -x "$EMU" ] || { echo "émulateur introuvable : $EMU (ORIC_EMU pour surcharger)"; exit 1; }

go build -o /tmp/bbsd "$ROOT/server/cmd/bbsd"
/tmp/bbsd -addr "127.0.0.1:$PORT" -content "$CONTENT" -data "$DATA" -idle 60s &
SRV=$!
trap 'kill $SRV 2>/dev/null || true; rm -rf "$DATA"' EXIT
sleep 0.5

# Le terminal démarre par : menu modem (1=ACIA $031C, 2=LOCI) -> répertoire
# (1-6/M) -> numérotation (ATD) -> session BBS. Le transport `modem:` émule un
# modem Hayes qui compose l'hôte du ATD ; les entrées 1-6 du répertoire pointent
# sur des BBS distants, donc on passe par la SAISIE MANUELLE (« m ») pour viser
# notre BBS LOCAL 127.0.0.1. La saisie clavier (input_line) consomme une touche
# à la fois (get_key + wait_release), donc on injecte UNE touche par évènement
# --type-keys, espacées ; Entrée = saut de ligne. Séquence :
#   « 1 »  menu modem  -> ACIA standard $031C
#   « m »  répertoire  -> saisie manuelle
#   host « 127.0.0.1 » + Entrée ; port « PORT » + Entrée ; proto « 1 » (telnet)
#   (modem -> connexion BBS local) -> accueil -> « 1 » (grille démo) -> « + »
# --realtime cadence la série réseau et rend --type-keys déterministe.
# IMPORTANT : input_line (saisie host/port) consomme une touche puis fait un
# wait_release. Sous --type-keys, une touche est MAINTENUE jusqu'à l'évènement
# suivant ; il faut donc une pause \p pour la relâcher AVANT la touche suivante
# et surtout AVANT l'Entrée (sinon le dernier caractère télescope le CR et la
# validation est perdue). Le host passe en une-touche-par-évènement (relâche
# naturel entre évènements espacés) ; le port en un évènement \p-pausé terminé
# par \p puis newline. Les menus (proto/accueil/grille) prennent une touche par
# évènement espacé (get_key sans saisie multi-caractères).
KEYS=(--type-keys "7500000:1" --type-keys "9500000:m")
c=11000000
addkey() { KEYS+=(--type-keys "$c:$1"); c=$((c + 700000)); }
host="127.0.0.1"; for ((i=0; i<${#host}; i++)); do addkey "${host:$i:1}"; done
KEYS+=(--type-keys "$c:"$'\n'); c=$((c + 1500000))                  # Entrée host -> Port
# port : un évènement, chaque chiffre suivi de \p (pause = relâche), \p avant le CR
portseq=""; for ((i=0; i<${#PORT}; i++)); do portseq+="${PORT:$i:1}\\p1"; done
KEYS+=(--type-keys "$c:"$portseq$'\n'); c=$((c + 7000000))          # Entrée port -> proto
KEYS+=(--type-keys "$c:1"); c=$((c + 3000000))                      # proto telnet -> dial/connexion
KEYS+=(--type-keys "$c:1"); c=$((c + 3000000))                      # accueil -> grille
KEYS+=(--type-keys "$c:+"); SHOT=$((c + 2500000)); STOP=$((SHOT + 500000))  # sélection

SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy \
  "$EMU" -t "$TAP" -f -r "$ROM" \
    --serial "modem:127.0.0.1:$PORT" --serial-buffer 512 \
    --headless --realtime \
    "${KEYS[@]}" \
    --screenshot-at "$SHOT:$OUT" -c "$STOP"

echo "Screenshot -> $OUT"
command -v pnmtopng >/dev/null && pnmtopng "$OUT" > "${OUT%.ppm}.png" && echo "PNG -> ${OUT%.ppm}.png" || true
