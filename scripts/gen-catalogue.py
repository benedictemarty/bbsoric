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
import shutil
import sys

# Extensions réellement transférables vers un Oric (le reste — PDF, PNG… — est
# consultable mais pas téléchargeable). Ordre = préférence (cassette avant disque).
DOWNLOAD_EXTS = (".tap", ".ort", ".rom", ".dsk")

# Buffer de réception du terminal Oric ($4000..~$B800). Au-delà, non téléchargeable.
DEFAULT_MAX_FILE = 30720


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


def pick_download(files, maxsize):
    """Choisit le fichier téléchargeable d'un programme : la plus petite entrée
    d'extension transférable qui tient dans maxsize (cassette préférée au disque).
    Renvoie (fileinfo, size) ou (None, best_size) si aucun ne tient (size la plus
    petite connue, pour information)."""
    cands = [f for f in (files or []) if f.get("ext", "").lower() in DOWNLOAD_EXTS]
    if not cands:
        return None, 0
    def key(f):
        return (DOWNLOAD_EXTS.index(f["ext"].lower()), f.get("size", 1 << 62))
    cands.sort(key=key)
    for f in cands:
        if f.get("size", 1 << 62) <= maxsize:
            return f, f["size"]
    return None, min(f.get("size", 0) for f in cands)


def sedoric_name(name):
    """Nom court sûr pour -files (majuscules, [A-Z0-9], 8.3), aligné sur ce que le
    terminal sauvera. Évite les caractères hors ASCII/espace."""
    base, _, ext = name.rpartition(".")
    base = base or name
    keep = lambda s: "".join(c for c in s.upper() if c.isalnum())[:8]
    b, e = keep(base), keep(ext)[:3]
    return (b + ("." + e if e else "")) or "FICHIER"


def software_rows(lib, limit, maxsize, copy_dest):
    """Lignes 'Logiciel' depuis data/catalog.json. `fichier` n'est renseigné que si
    un fichier transférable tient dans maxsize ; si copy_dest, ce fichier y est copié
    (nom court unique) et `fichier` prend ce nom. `taille` est la taille en octets."""
    path = os.path.join(lib, "data", "catalog.json")
    with open(path, encoding="utf-8") as f:
        cat = json.load(f)
    rows, used = [], {}
    for p in cat.get("programs", []):
        m = p.get("metadata") or {}
        dl, size = pick_download(p.get("files"), maxsize)
        fichier = ""
        if dl:
            fichier = basename(dl["name"])
            if copy_dest:
                short = sedoric_name(dl["name"])
                n, cand = 1, short
                while cand in used and used[cand] != dl.get("sha1"):
                    stem, _, ext = short.rpartition(".")
                    cand = "%s%d.%s" % ((stem or short)[:6], n, ext) if ext else "%s%d" % (short[:7], n)
                    n += 1
                used[cand] = dl.get("sha1")
                try:
                    if not os.path.exists(os.path.join(copy_dest, cand)):
                        shutil.copy(dl["path"], os.path.join(copy_dest, cand))
                    fichier = cand
                except OSError as e:
                    print("copie ignoree (%s): %s" % (dl["name"], e), file=sys.stderr)
                    fichier = ""
        auteur = clean(m.get("programmer")) or clean(m.get("publisher"))
        desc = clean(m.get("comment")) or clean(m.get("genre"))
        pf, pt = m.get("players_from"), m.get("players_to")
        joueurs = ""
        if pf:
            joueurs = str(pf) if not pt or pt == pf else "%s-%s" % (pf, pt)
        rows.append({
            "titre": (clean(p.get("title")) or clean(m.get("name")))[:40],
            "auteur": auteur[:40],
            "annee": m.get("year") or 0,
            "taille": size or 0,
            "fichier": fichier,        # non vide = téléchargeable (present dans -files)
            "genre": clean(m.get("genre"))[:20],
            "editeur": clean(m.get("publisher"))[:40],
            "langue": clean(m.get("language"))[:16],
            "joueurs": joueurs,
            "ecran": basename(m.get("screenshot")),   # référence capture d'écran
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


# Colonnes du catalogue (partagées standalone / merge).
CAT_COLONNES = {
    "id":          {"type": "INTEGER", "libelle": "ID", "cle_primaire": True, "auto_increment": True},
    "categorie":   {"type": "TEXT", "libelle": "Categorie", "longueur_max": 10},
    "titre":       {"type": "TEXT", "libelle": "Titre", "longueur_max": 40},
    "auteur":      {"type": "TEXT", "libelle": "Auteur", "longueur_max": 40},
    "annee":       {"type": "INTEGER", "libelle": "Annee"},
    "taille":      {"type": "INTEGER", "libelle": "Taille o"},
    "genre":       {"type": "TEXT", "libelle": "Genre", "longueur_max": 20},
    "editeur":     {"type": "TEXT", "libelle": "Editeur", "longueur_max": 40},
    "langue":      {"type": "TEXT", "libelle": "Langue", "longueur_max": 16},
    "joueurs":     {"type": "TEXT", "libelle": "Joueurs", "longueur_max": 8},
    "ecran":       {"type": "TEXT", "libelle": "Ecran", "longueur_max": 32},
    "fichier":     {"type": "TEXT", "libelle": "Fichier", "longueur_max": 16},
    "description": {"type": "TEXT", "libelle": "Description", "longueur_max": 200},
}


def build_catalogue(lib, limit, maxsize, copy_dest):
    """Construit la source `catalogue` (1 table, colonne categorie) et ses pages
    (menu + 3 vues filtrées). Renvoie (source, pages, (nl, nm, nv))."""
    logiciels = software_rows(lib, limit, maxsize, copy_dest)
    magazines = magazine_rows(lib, limit)
    livres = pdf_rows(os.path.join(lib, "library", "livres"), limit, recursive=False)
    for r in logiciels:
        r["categorie"] = "Logiciel"
    for r in magazines:
        r["categorie"] = "Magazine"
    for r in livres:
        r["categorie"] = "Livre"
    source = {
        "table": "catalogue",
        "tri_defaut": "titre ASC",
        "lignes_par_page": 15,
        "colonnes": CAT_COLONNES,
        "donnees": logiciels + magazines + livres,
    }
    pages = {
        "catalogue": {"title": "CATALOGUE", "entries": [
            {"key": "1", "label": "Logiciels (%d)" % len(logiciels), "target": "g_logiciels"},
            {"key": "2", "label": "Magazines (%d)" % len(magazines), "target": "g_magazines"},
            {"key": "3", "label": "Livres (%d)" % len(livres), "target": "g_livres"},
            {"key": "R", "label": "Retour", "target": "__back__"},
        ]},
        "g_logiciels": grille("LOGICIELS", "Logiciel", avec_fichier=True),
        "g_magazines": grille("MAGAZINES", "Magazine", avec_fichier=False),
        "g_livres":    grille("LIVRES", "Livre", avec_fichier=False),
    }
    return source, pages, (len(logiciels), len(magazines), len(livres))


def build_site(lib, limit, maxsize, copy_dest):
    """Site autonome dont la page de départ est le catalogue."""
    source, pages, counts = build_catalogue(lib, limit, maxsize, copy_dest)
    site = {
        "_comment": "Catalogue de telechargement BBS Oric (genere par scripts/gen-catalogue.py). "
                    "UN catalogue (colonne categorie) presente en 3 vues filtrees (filtre_fixe). "
                    "Logiciels telechargeables (touche X, XMODEM) si le fichier est petit et present "
                    "dans -files ; magazines/livres (PDF) consultables (fiche V). Filtre F, tri T.",
        "start": "catalogue",
        "sources_donnees": {"catalogue": source},
        "pages": pages,
    }
    return site, counts


def merge_into(site_path, lib, limit, maxsize, copy_dest, menu_page, menu_key):
    """Greffe le catalogue dans un site.json existant : ajoute la source `catalogue`,
    ses pages, et une entrée de menu (menu_key -> "catalogue") sur menu_page, insérée
    avant l'éventuelle sortie (__quit__). Idempotent sur l'entrée de menu."""
    with open(site_path, encoding="utf-8") as f:
        site = json.load(f)
    source, pages, counts = build_catalogue(lib, limit, maxsize, copy_dest)
    site.setdefault("sources_donnees", {})["catalogue"] = source
    site.setdefault("pages", {}).update(pages)

    mp = site.get("pages", {}).get(menu_page)
    if mp is None:
        raise SystemExit("page de menu %r introuvable dans %s" % (menu_page, site_path))
    entries = mp.setdefault("entries", [])
    entries = [e for e in entries if e.get("target") != "catalogue"]  # évite un doublon
    entry = {"key": menu_key, "label": "Catalogue", "target": "catalogue"}
    quit_idx = next((i for i, e in enumerate(entries)
                     if e.get("target") == "__quit__" or e.get("target") == "__back__"), len(entries))
    entries.insert(quit_idx, entry)
    mp["entries"] = entries
    return site, counts


def main():
    ap = argparse.ArgumentParser(description="Génère le catalogue BBS Oric depuis OricProgramsLib.")
    ap.add_argument("--lib", required=True, help="racine OricProgramsLib")
    ap.add_argument("--limit", type=int, default=0, help="items max par catégorie (0 = tout)")
    ap.add_argument("--out", default="-", help="fichier de sortie (- = stdout)")
    ap.add_argument("--max-file-size", type=int, default=DEFAULT_MAX_FILE,
                    help="taille max d'un fichier téléchargeable en octets (défaut %d)" % DEFAULT_MAX_FILE)
    ap.add_argument("--copy-files", default="",
                    help="répertoire -files où copier les fichiers téléchargeables (vide = ne copie pas)")
    ap.add_argument("--merge-into", default="",
                    help="site.json existant où greffer le catalogue (au lieu d'un site autonome)")
    ap.add_argument("--menu-page", default="main", help="page de menu où ajouter l'entrée Catalogue (avec --merge-into)")
    ap.add_argument("--menu-key", default="8", help="touche de l'entrée Catalogue (avec --merge-into)")
    args = ap.parse_args()

    if args.copy_files:
        os.makedirs(args.copy_files, exist_ok=True)

    if args.merge_into:
        site, (nl, nm, nv) = merge_into(args.merge_into, args.lib, args.limit,
                                        args.max_file_size, args.copy_files, args.menu_page, args.menu_key)
    else:
        site, (nl, nm, nv) = build_site(args.lib, args.limit, args.max_file_size, args.copy_files)
    dl = sum(1 for r in site["sources_donnees"]["catalogue"]["donnees"] if r.get("fichier"))
    data = json.dumps(site, ensure_ascii=False, indent=2)
    if args.out == "-":
        print(data)
    else:
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(data + "\n")
    print("Catalogue : %d logiciels (%d telechargeables <= %do), %d magazines, %d livres -> %s"
          % (nl, dl, args.max_file_size, nm, nv, args.out), file=sys.stderr)
    if args.copy_files:
        print("Fichiers copies dans %s" % args.copy_files, file=sys.stderr)


if __name__ == "__main__":
    main()
