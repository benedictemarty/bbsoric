#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Génère le catalogue de téléchargement du BBS Oric depuis la base du site
**tachibana.eu** (`tachibana.sqlite`, table `items`) — source alternative à
OricProgramsLib (cf. `gen-catalogue.py`, même format de sortie).

Le format d'affichage BBS est STRICTEMENT le même (source DataWindow `catalogue`
en 3 vues filtrées + menu) : ce script ne fait que fournir les LIGNES depuis une
autre source, puis réutilise l'assemblage partagé de `gen-catalogue.py`.

Correspondance des catégories tachibana -> BBS :
    Logiciel                 -> Logiciel  (téléchargeable : X → XMODEM)
    Revue                    -> Magazine  (consultable, non téléchargeable)
    Livre + Documentation    -> Livre     (consultable, non téléchargeable)

Les fichiers téléchargeables (.tap/.dsk/.rom/.ort) ne sont pas dans une colonne
dédiée de `items` : leurs URL `/lib/...` sont dans le HTML du champ `body`. On les
en extrait, on résout `/lib/X` -> `<oriclib>/X` (un miroir local du volume
`/srv/oriclib` de tachibana), on choisit le plus petit fichier transférable qui
tient dans le plafond, et on le copie dans `-files` (comme gen-catalogue.py).

Usage :
    python3 scripts/gen-catalogue-tachibana.py \
        --db /chemin/tachibana.sqlite --oriclib /chemin/miroir/srv/oriclib \
        [--copy-files /tmp/bbsfiles] [--limit N] [--out catalogue.json] \
        [--merge-into site.json --menu-page main --menu-key 8]
"""
import argparse
import html
import importlib.util
import os
import re
import shutil
import sqlite3
import sys

# Réutilise l'assemblage partagé (format d'affichage BBS) de gen-catalogue.py.
_HERE = os.path.dirname(os.path.abspath(__file__))
_SPEC = importlib.util.spec_from_file_location("gencat", os.path.join(_HERE, "gen-catalogue.py"))
gc = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(gc)

# Extensions transférables (mêmes que gen-catalogue.py, ordre = préférence).
DOWNLOAD_EXTS = gc.DOWNLOAD_EXTS

# Liens de fichiers dans le body : href="/lib/.../nom.tap" (tap/dsk/rom/ort).
_HREF_RE = re.compile(r'href="(/lib/[^"?]+\.(?:tap|dsk|rom|ort))"', re.I)
# Premier paragraphe de contenu (description oric.org) et retrait des balises.
_P_RE = re.compile(r"<p\b[^>]*>(.*?)</p>", re.I | re.S)
_TAG_RE = re.compile(r"<[^>]+>")


def strip_html(s):
    """Texte brut d'un fragment HTML (retire balises, décode les entités)."""
    return html.unescape(_TAG_RE.sub("", s or "")).strip()


def description_of(body):
    """Description lisible : premier <p> « de contenu » du body (on saute les
    paragraphes techniques : note de source, étoiles de notation)."""
    for frag in _P_RE.findall(body or ""):
        txt = re.sub(r"\s+", " ", strip_html(frag)).strip()
        if not txt or txt.startswith("Source :") or txt.startswith("★") or txt.startswith("☆"):
            continue
        return txt
    return ""


def local_path(oriclib, url):
    """Résout une URL `/lib/X` en chemin local `<oriclib>/X` (le miroir du volume
    `/srv/oriclib` où `/lib/` est un alias). Renvoie None hors périmètre."""
    if not url.startswith("/lib/"):
        return None
    return os.path.join(oriclib, url[len("/lib/"):])


def pick_from_body(body, oriclib, maxsize):
    """Choisit le fichier téléchargeable d'un logiciel à partir des liens du body :
    plus petit fichier transférable présent localement et ≤ maxsize (cassette avant
    disque). Renvoie (path, name, size) ou (None, "", best_size)."""
    infos = []
    for url in dict.fromkeys(_HREF_RE.findall(body or "")):  # dédoublonne, garde l'ordre
        ext = "." + url.rsplit(".", 1)[1].lower()
        if ext not in DOWNLOAD_EXTS:
            continue
        p = local_path(oriclib, url)
        if not p or not os.path.isfile(p):
            continue  # fichier absent du miroir -> non téléchargeable
        sz = os.path.getsize(p)
        if sz == 0:
            continue  # fichier vide (coquille) -> non téléchargeable
        infos.append({"ext": ext, "size": sz, "name": os.path.basename(url), "path": p})
    dl, size = gc.pick_download(infos, maxsize)
    if dl:
        return dl["path"], dl["name"], size
    return None, "", size


def copy_download(src_path, title, name, copy_dest, used):
    """Copie le fichier dans copy_dest sous un nom court sûr (dérivé du titre, sinon
    du nom d'origine), dédoublonné par chemin source. Renvoie le nom court ou ""."""
    stem = title or name.rsplit(".", 1)[0]
    ext = name.rsplit(".", 1)[1] if "." in name else "tap"
    short = gc.sedoric_name(stem + "." + ext)
    cand, n = short, 1
    while cand in used and used[cand] != src_path:
        base, _, e = short.rpartition(".")
        cand = "%s%d.%s" % ((base or short)[:6], n, e) if e else "%s%d" % (short[:7], n)
        n += 1
    used[cand] = src_path
    try:
        dest = os.path.join(copy_dest, cand)
        if not os.path.exists(dest):
            shutil.copy(src_path, dest)
        return cand
    except OSError as e:
        print("copie ignoree (%s): %s" % (name, e), file=sys.stderr)
        return ""


def software_rows(con, oriclib, limit, maxsize, copy_dest, downloadable_only=True):
    """Lignes 'Logiciel' depuis items (category='Logiciel', published=1). `fichier`
    n'est renseigné que si un fichier transférable tient dans maxsize et existe dans
    le miroir ; il est alors copié dans copy_dest (nom court unique).

    downloadable_only (défaut) : n'émet QUE les logiciels réellement téléchargeables
    (BBS orienté download ; évite ~2/3 d'entrées non transférables et les 'taille 0').
    Passer False pour un catalogue navigable complet (métadonnées seules incluses)."""
    rows, used = [], {}
    q = ("SELECT title, publisher, programmer, year, kind, language, body "
         "FROM items WHERE category='Logiciel' AND published=1 ORDER BY title COLLATE NOCASE")
    for title, pub, prog, year, kind, lang, body in con.execute(q):
        title = gc.clean(title)
        path, name, size = pick_from_body(body, oriclib, maxsize)
        fichier = ""
        if path and copy_dest:
            fichier = copy_download(path, title, name, copy_dest, used)
        elif path:
            fichier = gc.sedoric_name((title or name.rsplit(".", 1)[0]) + "." + name.rsplit(".", 1)[-1])
        if downloadable_only and not fichier:
            continue  # non téléchargeable -> exclu de la vue Logiciels
        auteur = gc.clean(prog) or gc.clean(pub)
        rows.append({
            "titre": title[:40],
            "auteur": auteur[:40],
            "annee": year or 0,
            "taille": size or 0,
            "fichier": fichier,        # non vide = téléchargeable (présent dans -files)
            "genre": gc.clean(kind)[:20],
            "editeur": gc.clean(pub)[:40],
            "langue": gc.clean(lang)[:16],
            "joueurs": "",             # non porté par tachibana
            "ecran": "",               # les captures tachibana sont des URL, pas des fichiers BBS
            "description": description_of(body)[:200],
        })
        if limit and len(rows) >= limit:
            break
    return rows


def consult_rows(con, categories, limit):
    """Lignes consultables (Revue -> Magazine, Livre/Documentation -> Livre) :
    métadonnées seules, colonne fichier vide (PDF non transférables vers l'Oric)."""
    rows = []
    ph = ",".join("?" * len(categories))
    q = ("SELECT title, publisher, year, kind, body FROM items "
         "WHERE category IN (%s) AND published=1 ORDER BY title COLLATE NOCASE" % ph)
    for title, pub, year, kind, body in con.execute(q, categories):
        rows.append({
            "titre": gc.clean(title)[:40],
            "auteur": gc.clean(pub)[:40],
            "annee": year or 0,
            "fichier": "",
            "genre": gc.clean(kind)[:20],
            "editeur": gc.clean(pub)[:40],
            "description": description_of(body)[:200],
        })
        if limit and len(rows) >= limit:
            break
    return rows


def build_catalogue(db, oriclib, limit, maxsize, copy_dest, downloadable_only=True):
    """Construit le catalogue depuis tachibana.sqlite. Renvoie (source, pages, counts)."""
    con = sqlite3.connect("file:%s?mode=ro" % db, uri=True)
    try:
        logiciels = software_rows(con, oriclib, limit, maxsize, copy_dest, downloadable_only)
        magazines = consult_rows(con, ["Revue"], limit)
        livres = consult_rows(con, ["Livre", "Documentation"], limit)
    finally:
        con.close()
    return gc.assemble_catalogue(logiciels, magazines, livres)


def main():
    ap = argparse.ArgumentParser(description="Génère le catalogue BBS Oric depuis tachibana.sqlite.")
    ap.add_argument("--db", required=True, help="base SQLite tachibana (tachibana.sqlite)")
    ap.add_argument("--oriclib", required=True,
                    help="miroir local du volume /srv/oriclib (où /lib/X -> <oriclib>/X)")
    ap.add_argument("--limit", type=int, default=0, help="items max par catégorie (0 = tout)")
    ap.add_argument("--out", default="-", help="fichier de sortie (- = stdout)")
    ap.add_argument("--max-file-size", type=int, default=gc.DEFAULT_MAX_FILE,
                    help="taille max d'un fichier téléchargeable en octets (défaut %d)" % gc.DEFAULT_MAX_FILE)
    ap.add_argument("--copy-files", default="",
                    help="répertoire -files où copier les fichiers téléchargeables (vide = ne copie pas)")
    ap.add_argument("--merge-into", default="",
                    help="site.json existant où greffer le catalogue (au lieu d'un site autonome)")
    ap.add_argument("--menu-page", default="main", help="page de menu où ajouter l'entrée Catalogue")
    ap.add_argument("--menu-key", default="8", help="touche de l'entrée Catalogue")
    ap.add_argument("--all-software", action="store_true",
                    help="lister AUSSI les logiciels non téléchargeables (défaut : téléchargeables seulement)")
    args = ap.parse_args()

    if args.copy_files:
        os.makedirs(args.copy_files, exist_ok=True)

    catalogue = build_catalogue(args.db, args.oriclib, args.limit, args.max_file_size,
                                args.copy_files, downloadable_only=not args.all_software)
    if args.merge_into:
        import json
        with open(args.merge_into, encoding="utf-8") as f:
            site = json.load(f)
        site, (nl, nm, nv) = gc.graft_catalogue(site, catalogue, args.menu_page, args.menu_key)
    else:
        site, (nl, nm, nv) = gc.wrap_site(catalogue)

    dl = sum(1 for r in site["sources_donnees"]["catalogue"]["donnees"] if r.get("fichier"))
    import json
    data = json.dumps(site, ensure_ascii=False, indent=2)
    if args.out == "-":
        print(data)
    else:
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(data + "\n")
    print("Catalogue tachibana : %d logiciels (%d telechargeables <= %do), %d magazines, %d livres -> %s"
          % (nl, dl, args.max_file_size, nm, nv, args.out), file=sys.stderr)
    if args.copy_files:
        print("Fichiers copies dans %s" % args.copy_files, file=sys.stderr)


if __name__ == "__main__":
    main()
