#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Tests unitaires de scripts/gen-catalogue-tachibana.py.

Vérifie l'extraction propre à la source tachibana (parsing du body HTML) sans
dépendre de la vraie base : description_of (1er paragraphe de contenu), strip_html,
local_path (résolution /lib/X -> miroir), pick_from_body (choix du fichier
téléchargeable présent, non vide, ≤ plafond, cassette avant disque) et le rendu
complet build_catalogue sur une base SQLite + un miroir de fichiers construits à la
volée dans un répertoire temporaire.

Usage :  python3 scripts/test-gen-catalogue-tachibana.py     (exit 0 = OK, 1 = échec)
"""
import importlib.util
import os
import sqlite3
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC = importlib.util.spec_from_file_location("gencat_t", os.path.join(HERE, "gen-catalogue-tachibana.py"))
gt = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gt)

PASS = 0
FAIL = 0


def check(cond, label):
    global PASS, FAIL
    if cond:
        print("  ok   :", label); PASS += 1
    else:
        print("  FAIL :", label); FAIL += 1


# --- strip_html / description_of ---------------------------------------------
check(gt.strip_html("<p>Salut&nbsp;<b>toi</b>&#x27;</p>") == "Salut\xa0toi'",
      "strip_html retire les balises et décode les entités")

body = ('<div class="article-body"><p>Un jeu d\'arcade rapide.</p>'
        '<p class="note">Source : LaunchBox (traduction).</p></div>')
check(gt.description_of(body) == "Un jeu d'arcade rapide.",
      "description_of prend le 1er paragraphe de contenu (saute 'Source :')")

body_rating = ('<p class="rating">★★★☆☆ <span>3/5</span></p><p>La vraie description.</p>')
check(gt.description_of(body_rating) == "La vraie description.",
      "description_of saute le paragraphe de notation (étoiles)")

check(gt.description_of("<h3>Fichiers</h3>") == "",
      "description_of renvoie vide si aucun paragraphe de contenu")


# --- local_path ---------------------------------------------------------------
check(gt.local_path("/srv/oriclib", "/lib/live/games/tap/x.tap") == "/srv/oriclib/live/games/tap/x.tap",
      "local_path résout /lib/X -> <oriclib>/X")
check(gt.local_path("/srv/oriclib", "/autre/x.tap") is None,
      "local_path ignore une URL hors /lib/")


# --- pick_from_body : fixtures fichiers dans un miroir temporaire --------------
def write(root, rel, size):
    p = os.path.join(root, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "wb") as f:
        f.write(b"\0" * size)
    return p


with tempfile.TemporaryDirectory() as lib:
    write(lib, "live/games/tap/small.tap", 1000)
    write(lib, "live/games/dsk/big.dsk", 50000)
    write(lib, "Programs/z/files/empty.tap", 0)      # coquille vide -> ignorée
    write(lib, "live/games/tap/huge.tap", 70000)     # > plafond -> ignorée

    # cassette (.tap) préférée au disque (.dsk) même présents tous deux
    body = ('<a href="/lib/live/games/dsk/big.dsk" download>D</a>'
            '<a href="/lib/live/games/tap/small.tap" download>T</a>')
    p, name, size = gt.pick_from_body(body, lib, 65535)
    check(name == "small.tap" and size == 1000, "pick_from_body : cassette avant disque")

    # fichier vide (0 o) ignoré -> non téléchargeable si c'est le seul
    p, name, size = gt.pick_from_body('<a href="/lib/Programs/z/files/empty.tap">E</a>', lib, 65535)
    check(p is None, "pick_from_body : fichier vide (0 o) ignoré")

    # au-delà du plafond -> non téléchargeable
    p, name, size = gt.pick_from_body('<a href="/lib/live/games/tap/huge.tap">H</a>', lib, 65535)
    check(p is None, "pick_from_body : fichier > plafond ignoré")

    # référence vers un fichier absent du miroir -> ignorée
    p, name, size = gt.pick_from_body('<a href="/lib/live/games/tap/nope.tap">N</a>', lib, 65535)
    check(p is None, "pick_from_body : référence absente du miroir ignorée")

    # repli sur le disque si la cassette dépasse le plafond
    body = ('<a href="/lib/live/games/tap/huge.tap">H</a>'
            '<a href="/lib/live/games/dsk/big.dsk">D</a>')
    p, name, size = gt.pick_from_body(body, lib, 65535)
    check(name == "big.dsk", "pick_from_body : repli sur .dsk si .tap dépasse")

    # --- build_catalogue de bout en bout sur une base SQLite jetable ----------
    db = os.path.join(lib, "t.sqlite")
    con = sqlite3.connect(db)
    con.execute("CREATE TABLE items (title TEXT, publisher TEXT, programmer TEXT, year INT, "
                "kind TEXT, category TEXT, language TEXT, body TEXT, published INT)")
    con.executemany(
        "INSERT INTO items (title,publisher,programmer,year,kind,category,language,body,published) "
        "VALUES (?,?,?,?,?,?,?,?,?)",
        [
            ("Petit Jeu", "EditPub", "AuteurX", 1984, "Arcade", "Logiciel", "French",
             '<p>Super jeu.</p><a href="/lib/live/games/tap/small.tap">T</a>', 1),
            ("Sans Fichier", "P", "", 0, "Utilitaire", "Logiciel", "", "<p>Rien à télécharger.</p>", 1),
            ("Brouillon", "", "", 0, "", "Logiciel", "", "<p>x</p>", 0),   # non publié -> exclu
            ("Ma Revue 3", "CEO", "", 1990, "Fanzine", "Revue", "", "<p>Un numéro.</p>", 1),
            ("Un Manuel", "Ed", "", 0, "Doc", "Documentation", "", "<p>Doc.</p>", 1),
            ("Un Livre", "Ed", "", 0, "", "Livre", "", "<p>Bouquin.</p>", 1),
        ])
    con.commit(); con.close()

    copy_dest = os.path.join(lib, "out")
    os.makedirs(copy_dest)
    source, pages, (nl, nm, nv) = gt.build_catalogue(db, lib, 0, 65535, copy_dest)
    check((nl, nm, nv) == (2, 1, 2), "build_catalogue : 2 logiciels publiés, 1 magazine, 2 livres (Livre+Doc)")

    data = source["donnees"]
    jeu = next(r for r in data if r["titre"] == "Petit Jeu")
    check(jeu["fichier"] and jeu["categorie"] == "Logiciel" and jeu["taille"] == 1000,
          "build_catalogue : logiciel téléchargeable renseigné (fichier + taille)")
    check(os.path.isfile(os.path.join(copy_dest, jeu["fichier"])),
          "build_catalogue : le fichier téléchargeable est bien copié dans -files")
    check(jeu["auteur"] == "AuteurX" and jeu["editeur"] == "EditPub" and jeu["description"] == "Super jeu.",
          "build_catalogue : métadonnées mappées (auteur=programmer, editeur=publisher, description)")

    sansf = next(r for r in data if r["titre"] == "Sans Fichier")
    check(sansf["fichier"] == "", "build_catalogue : logiciel sans fichier -> colonne fichier vide")

    check(all(r["fichier"] == "" for r in data if r["categorie"] in ("Magazine", "Livre")),
          "build_catalogue : magazines/livres non téléchargeables (fichier vide)")
    check(not any(r["titre"] == "Brouillon" for r in data),
          "build_catalogue : les items non publiés sont exclus")

print()
print("Bilan: %d ok, %d ko" % (PASS, FAIL))
sys.exit(1 if FAIL else 0)
