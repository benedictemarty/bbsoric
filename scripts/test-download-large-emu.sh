#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-download-large-emu.sh — preuve runtime de l'histoire I2b-c :
# TELECHARGER un fichier > 64 Kio.
#
# Jusqu'ici l'en-tête de download codait la taille réelle sur 2 octets (≤ 64 Kio)
# et le firmware manipulait xdrem sur 16 bits. L'en-tête v4 porte la taille sur
# 3 octets (24 bits) et le firmware décrémente xdrem sur 24 bits : le plafond
# passe de 64 Kio à ~8 Mio (borné par la jauge, cf. server maxDownloadSize).
#
# Ce script réutilise le harnais de streaming LOCI (test-stream-loci-emu.sh, qui
# initialise xdrem sur 3 octets) mais avec une taille de 80000 o (> 65535), et
# vérifie que le fichier reçu sur la carte SD est OCTET-IDENTIQUE à la source.
# C'est la preuve que le décrément 24 bits de xdrem (> 512 blocs) est correct.
#
# Dépendances (ignorées proprement si absentes) : xa (xa65), go, oric1-emu + ROM.
# Code de sortie : 0 = vert (ou skip), 1 = échec.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "== I2b-c : download > 64 Kio (80000 o, en-tête v4 / xdrem 24 bits) =="
# Taille > 64 Kio, budget cycles doublé (2× de données vs le test 40000 o), port
# dédié pour ne pas heurter une instance parallèle du test de streaming.
STREAM_TEST_SIZE=80000 \
STREAM_TEST_PORT="${LARGE_TEST_PORT:-6547}" \
STREAM_TEST_CYCLES="${LARGE_TEST_CYCLES:-200000000}" \
    exec bash "$ROOT/scripts/test-stream-loci-emu.sh"
