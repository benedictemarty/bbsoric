#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Génère le catalogue de téléchargement du BBS Oric (Logiciels / Magazines /
Livres) au format `content` (sources_donnees + pages), depuis la bibliothèque
OricProgramsLib.

Trois sources DataWindow (une par catégorie) + trois grilles + un menu. Chaque
ligne porte titre / auteur / année et, pour les logiciels, le nom du fichier
téléchargeable (colonne `fichier`, action X → XMODEM côté BBS). Les magazines et
livres (PDF) sont LISTÉS pour la consultation (fiche détail via V) mais ne sont
pas téléchargeables vers un Oric (buffer terminal ~30 Ko) : leur colonne fichier
reste vide — c'est volontaire, pas un oubli.

Usage :
    python3 scripts/gen-catalogue.py --lib "/media/.../OricProgramsLib" \
        [--limit N] [--out catalogue.json]

--limit borne le nombre d'items par catégorie (utile pour une démo committable).
Le JSON produit est un Site complet, validable par `internal/content` et servable
via `./bbsd -content <out> -data <dir>` (+ `-files <dir>` pour les téléchargements).
"""
import argparse
import json
import os
import sys


def clean(v):
    """Nettoie une valeur de métadonnée (retire les '(Unknown)', None)."""
    if v is None:
        return ""
    s = str(v).strip()
    if s in ("(Unknown)", "Unknown", "None"):
        return ""
    return s


def basename(path):
    """Dernier segment d'un chemin (séparateurs Windows ou Unix)."""
    if not path:
        return ""
    return path.replace("\\", "/").rsplit("/", 1)[-1]


def titre_from_pdf(name):
    """Titre lisible depuis un nom de fichier PDF."""
    t = name.rsplit(".pdf", 1)[0].replace("_", " ").replace("-", " ").strip()
    return t[:60]


def software_rows(lib, limit):
    """Lignes 'Logiciel' depuis data/catalog.json."""
    path = os.path.join(lib, "data", "catalog.json")
    with open(path, encoding="utf-8") as f:
        cat = json.load(f)
    rows = []
    for p in cat.get("programs", []):
        m = p.get("metadata") or {}
        fichier = basename(m.get("filename"))
        auteur = clean(m.get("programmer")) or clean(m.get("publisher"))
        desc = clean(m.get("comment")) or clean(m.get("genre"))
        rows.append({
            "titre": (clean(p.get("title")) or clean(m.get("name")))[:40],
            "auteur": auteur[:40],
            "annee": m.get("year") or 0,
            "fichier": fichier,        # téléchargeable si présent dans -files et petit
            "description": desc[:200],
        })
        if limit and len(rows) >= limit:
            break
    return rows


def pdf_rows(root, limit, recursive):
    """Lignes depuis des PDF (magazines ou livres). fichier laissé vide (non
    téléchargeable vers l'Oric)."""
    rows = []
    if not os.path.isdir(root):
        return rows
    walker = os.walk(root) if recursive else [(root, [], os.listdir(root))]
    for dirpath, _dirs, filenames in walker:
        rubrique = os.path.basename(dirpath)
        for fn in sorted(filenames):
            if not fn.lower().endswith(".pdf"):
                continue
            rows.append({
                "titre": titre_from_pdf(fn)[:40],
                "auteur": rubrique[:40] if recursive and rubrique != os.path.basename(root) else "",
                "annee": 0,
                "fichier": "",
                "description": fn[:200],
            })
            if limit and len(rows) >= limit:
                return rows
    return rows


def magazine_rows(lib, limit):
    """Lignes 'Magazine' : chaque sous-dossier de library/ (sauf livres) = une
    revue ; ses PDF sont les numéros."""
    libdir = os.path.join(lib, "library")
    rows = []
    if not os.path.isdir(libdir):
        return rows
    for rubrique in sorted(os.listdir(libdir)):
        if rubrique == "livres":
            continue
        sub = os.path.join(libdir, rubrique)
        if not os.path.isdir(sub):
            continue
        for fn in sorted(os.listdir(sub)):
            if not fn.lower().endswith(".pdf"):
                continue
            rows.append({
                "titre": titre_from_pdf(fn)[:40],
                "auteur": rubrique[:40],
                "annee": 0,
                "fichier": "",
                "description": fn[:200],
            })
            if limit and len(rows) >= limit:
                return rows
    return rows


def grille(titre, categorie, avec_fichier):
    """Page grille : vue filtrée d'une catégorie du catalogue unique (filtre_fixe),
    sans que l'utilisateur ait à saisir de filtre. fichier_colonne pour les logiciels."""
    dw = {
        "source": "catalogue",
        "colonnes_affichees": ["titre", "auteur", "annee"],
        "largeurs": [17, 12, 4],  # 1 + 3(index) + (17+1)+(12+1)+(4+1) = 40
        "filtre_fixe": {"colonne": "categorie", "valeur": categorie},
    }
    if avec_fichier:
        dw["fichier_colonne"] = "fichier"
    return {"title": titre, "datawindow": dw}


def build_site(lib, limit):
    logiciels = software_rows(lib, limit)
    magazines = magazine_rows(lib, limit)
    livres = pdf_rows(os.path.join(lib, "library", "livres"), limit, recursive=False)
    for r in logiciels:
        r["categorie"] = "Logiciel"
    for r in magazines:
        r["categorie"] = "Magazine"
    for r in livres:
        r["categorie"] = "Livre"
    toutes = logiciels + magazines + livres

    site = {
        "_comment": "Catalogue de telechargement BBS Oric (genere par scripts/gen-catalogue.py). "
                    "UN catalogue (colonne categorie) presente en 3 vues filtrees (filtre_fixe). "
                    "Logiciels telechargeables (touche X, XMODEM) si le fichier est petit et present "
                    "dans -files ; magazines/livres (PDF) consultables (fiche V) mais non telechargeables "
                    "vers l'Oric. Filtre utilisateur F, tri T, pagination S/R.",
        "start": "catalogue",
        "sources_donnees": {
            "catalogue": {
                "table": "catalogue",
                "tri_defaut": "titre ASC",
                "lignes_par_page": 15,
                "colonnes": {
                    "id":          {"type": "INTEGER", "libelle": "ID", "cle_primaire": True, "auto_increment": True},
                    "categorie":   {"type": "TEXT", "libelle": "Categorie", "longueur_max": 10},
                    "titre":       {"type": "TEXT", "libelle": "Titre", "longueur_max": 40},
                    "auteur":      {"type": "TEXT", "libelle": "Auteur", "longueur_max": 40},
                    "annee":       {"type": "INTEGER", "libelle": "Annee"},
                    "fichier":     {"type": "TEXT", "libelle": "Fichier", "longueur_max": 16},
                    "description": {"type": "TEXT", "libelle": "Description", "longueur_max": 200},
                },
                "donnees": toutes,
            },
        },
        "pages": {
            "catalogue": {"title": "CATALOGUE", "entries": [
                {"key": "1", "label": "Logiciels (%d)" % len(logiciels), "target": "g_logiciels"},
                {"key": "2", "label": "Magazines (%d)" % len(magazines), "target": "g_magazines"},
                {"key": "3", "label": "Livres (%d)" % len(livres), "target": "g_livres"},
                {"key": "Q", "label": "Retour", "target": "__back__"},
            ]},
            "g_logiciels": grille("LOGICIELS", "Logiciel", avec_fichier=True),
            "g_magazines": grille("MAGAZINES", "Magazine", avec_fichier=False),
            "g_livres":    grille("LIVRES", "Livre", avec_fichier=False),
        },
    }
    return site, (len(logiciels), len(magazines), len(livres))


def main():
    ap = argparse.ArgumentParser(description="Génère le catalogue BBS Oric depuis OricProgramsLib.")
    ap.add_argument("--lib", required=True, help="racine OricProgramsLib")
    ap.add_argument("--limit", type=int, default=0, help="items max par catégorie (0 = tout)")
    ap.add_argument("--out", default="-", help="fichier de sortie (- = stdout)")
    args = ap.parse_args()

    site, (nl, nm, nv) = build_site(args.lib, args.limit)
    data = json.dumps(site, ensure_ascii=False, indent=2)
    if args.out == "-":
        print(data)
    else:
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(data + "\n")
    print("Catalogue : %d logiciels, %d magazines, %d livres -> %s"
          % (nl, nm, nv, args.out), file=sys.stderr)


if __name__ == "__main__":
    main()
