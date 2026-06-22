#!/usr/bin/env bash
# Assemble le terminal Oric (term.s) et produit une .tap autorun (term.tap).
#
# Dépendances : xa (xa65) + python3. Le générateur .tap (bin2tap.py) est
# embarqué dans le dépôt → build autonome (pas de dépendance externe).
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
LOAD=0x1000   # adresse de chargement / exécution

xa "$HERE/term.s" -o "$HERE/term.bin"
python3 "$HERE/bin2tap.py" "$HERE/term.bin" "$LOAD" TERM "$HERE/term.tap"
