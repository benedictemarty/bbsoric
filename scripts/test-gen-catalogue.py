#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Tests unitaires de scripts/gen-catalogue.py (sélection du fichier téléchargeable).

Fonctions pures, sans OricProgramsLib : on vérifie que `pick_download` respecte le
plafond de taille et l'ordre de préférence d'extension, et que le plafond par défaut
a bien été relevé à 64 Ko (0xFFFF) pour exploiter la réception en streaming (I2b).

Usage :  python3 scripts/test-gen-catalogue.py        (exit 0 = OK, 1 = échec)
"""
import importlib.util
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC = importlib.util.spec_from_file_location("gencat", os.path.join(HERE, "gen-catalogue.py"))
gc = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gc)

PASS = 0
FAIL = 0


def check(cond, label):
    global PASS, FAIL
    if cond:
        print("  ok   :", label); PASS += 1
    else:
        print("  FAIL :", label); FAIL += 1


def tap(size):
    return {"ext": ".tap", "size": size, "name": "prog.tap"}


def dsk(size):
    return {"ext": ".dsk", "size": size, "name": "prog.dsk"}


# --- Plafond par défaut relevé à 0xFFFF (64 Ko) suite au streaming I2b ---------
check(gc.DEFAULT_MAX_FILE == 0xFFFF, "DEFAULT_MAX_FILE = 65535 (0xFFFF, aligne sur maxDownloadSize serveur)")

# --- Un fichier de 40000 o (> ancien plafond 30720, <= nouveau 65535) est
#     desormais telechargeable ; il ne l'etait PAS a 30720 ---------------------
f, size = gc.pick_download([tap(40000)], 65535)
check(f is not None and size == 40000, "40000 o telechargeable au plafond 65535")

f, _ = gc.pick_download([tap(40000)], 30720)
check(f is None, "40000 o NON telechargeable a l'ancien plafond 30720 (prouve l'effet du plafond)")

# --- Bornes exactes autour de 0xFFFF ------------------------------------------
f, _ = gc.pick_download([tap(65535)], 65535)
check(f is not None, "65535 o (limite exacte) telechargeable")
f, _ = gc.pick_download([tap(65536)], 65535)
check(f is None, "65536 o (au-dela) NON telechargeable")

# --- Preference d'extension : cassette (.tap) avant disque (.dsk) meme si plus
#     grosse, tant qu'elle tient (ordre DOWNLOAD_EXTS) --------------------------
f, _ = gc.pick_download([dsk(5000), tap(6000)], 65535)
check(f is not None and f["ext"] == ".tap", "preference .tap sur .dsk (cassette avant disque)")

# --- Si la cassette ne tient pas mais le disque oui, on prend le disque --------
f, _ = gc.pick_download([tap(70000), dsk(50000)], 65535)
check(f is not None and f["ext"] == ".dsk", "repli sur .dsk quand .tap depasse le plafond")

# --- Aucune extension transferable -> non telechargeable -----------------------
f, size = gc.pick_download([{"ext": ".pdf", "size": 100}], 65535)
check(f is None, "extension non transferable (.pdf) -> non telechargeable")

print()
print("Bilan: %d ok, %d ko" % (PASS, FAIL))
sys.exit(1 if FAIL else 0)
