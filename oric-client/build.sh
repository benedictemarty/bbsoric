#!/usr/bin/env bash
# Assemble le terminal Oric (term.s) et produit une .tap autorun (term.tap).
#
# Dépendances : xa (xa65), et l'outil bin2tap de l'émulateur Oric1.
# Surcharge possible : BIN2TAP=/chemin/bin2tap ./build.sh
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
BIN2TAP="${BIN2TAP:-$HOME/Oric1/bin2tap}"
LOAD=0x1000   # adresse de chargement / exécution

xa "$HERE/term.s" -o "$HERE/term.bin"
"$BIN2TAP" "$HERE/term.bin" --start "$LOAD" --exec "$LOAD" -o "$HERE/term.tap" --name TERM

echo "OK -> $HERE/term.tap  (load/exec $LOAD)"
