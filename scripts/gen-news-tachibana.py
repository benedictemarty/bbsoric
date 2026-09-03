#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Génère le fichier d'actualités du BBS Oric (`-news`) depuis les **articles** du
site tachibana.eu (`tachibana.sqlite`, table `articles`).

Le store news du BBS (`server/internal/news`) persiste un **tableau JSON d'Item**
`{title, body, author, at}`, en ordre chronologique (le plus récent en dernier).
On importe les articles **publiés** d'une langue donnée (défaut `fr`), en :
- retirant le HTML du corps (les articles portent du HTML de confiance) ;
- **translittérant les accents** (é→e…) car l'affichage Oric est en ASCII pur
  (`oascii.SanitizeText` écarte le non-ASCII) ;
- bornant titre (≤38) et corps (≤400) comme le fait le store.

Author = « tachibana.eu ». Date = `created_at` de l'article (normalisée RFC3339).

Usage :
    python3 scripts/gen-news-tachibana.py --db /tmp/tachibana/tachibana.sqlite \
        [--lang fr] [--out news.json] [--merge-into /var/lib/bbsoric/news.json]

Avec `--merge-into`, les annonces existantes **non issues de tachibana** (autre
`author`) sont préservées ; celles de tachibana sont remplacées par l'import frais.
"""
import argparse
import html
import json
import re
import sqlite3
import sys
import unicodedata
from datetime import datetime, timezone

AUTHOR = "tachibana.eu"
MAX_TITLE = 38   # cf. news.MaxTitle
MAX_BODY = 400   # cf. news.MaxBody

_TAG = re.compile(r"<[^>]+>")
# Caractères qui ne se décomposent pas proprement en ASCII via NFKD.
_SPECIALS = {
    "œ": "oe", "Œ": "OE", "æ": "ae", "Æ": "AE", "ß": "ss",
    "’": "'", "‘": "'", "“": '"', "”": '"', "«": '"', "»": '"',
    "–": "-", "—": "-", "…": "...", " ": " ", "€": "EUR",
}


def deaccent(s):
    """Translittère en ASCII (é→e, œ→oe, guillemets typographiques…)."""
    for k, v in _SPECIALS.items():
        s = s.replace(k, v)
    s = unicodedata.normalize("NFKD", s)
    return "".join(c for c in s if not unicodedata.combining(c))


def ascii_clean(s, maxlen):
    """Texte ASCII propre borné à maxlen (balises HTML retirées, accents
    translittérés, entités décodées, espaces compactés). Miroir de news.sanitize
    côté serveur (mais en conservant les accents translittérés au lieu de les écarter)."""
    s = _TAG.sub(" ", s or "")
    s = deaccent(html.unescape(s))
    s = "".join(ch if 32 <= ord(ch) < 127 else " " for ch in s)
    s = re.sub(r"\s+", " ", s).strip()
    return s[:maxlen].strip()


def body_text(summary, body):
    """Corps lisible : le résumé s'il existe, sinon le texte du HTML du corps."""
    if (summary or "").strip():
        return summary
    return _TAG.sub(" ", body or "")


def to_rfc3339(created):
    """Normalise une date d'article en RFC3339 (ce que le store attend pour At).
    Accepte l'ISO 8601 (avec ou sans Z/offset) et 'YYYY-MM-DD HH:MM:SS'. Renvoie
    None si non interprétable (l'article est alors ignoré — on n'invente pas de date)."""
    c = (created or "").strip()
    if not c:
        return None
    iso = c.replace("Z", "+00:00").replace(" ", "T", 1)
    for cand in (iso, c):
        try:
            dt = datetime.fromisoformat(cand)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        except ValueError:
            continue
    return None


def articles_to_items(db, lang):
    """Lit les articles publiés de la langue et renvoie la liste d'Item BBS
    (ordre chronologique croissant : le plus récent en dernier, comme le store)."""
    con = sqlite3.connect("file:%s?mode=ro" % db, uri=True)
    try:
        rows = con.execute(
            "SELECT title, summary, body, created_at FROM articles "
            "WHERE published=1 AND lang=? ORDER BY created_at ASC, id ASC",
            (lang,),
        ).fetchall()
    finally:
        con.close()
    items, skipped = [], 0
    for title, summary, body, created in rows:
        at = to_rfc3339(created)
        corps = ascii_clean(body_text(summary, body), MAX_BODY)
        if at is None or not corps:
            skipped += 1
            continue  # pas de date exploitable ou corps vide -> on n'invente pas
        titre = ascii_clean(title, MAX_TITLE) or "Annonce"
        items.append({"title": titre, "body": corps, "author": AUTHOR, "at": at})
    return items, skipped


def main():
    ap = argparse.ArgumentParser(description="Génère les actualités BBS depuis les articles tachibana.")
    ap.add_argument("--db", required=True, help="base SQLite tachibana (tachibana.sqlite)")
    ap.add_argument("--lang", default="fr", help="langue des articles à importer (défaut fr)")
    ap.add_argument("--out", default="-", help="fichier de sortie (- = stdout)")
    ap.add_argument("--merge-into", default="",
                    help="news.json existant : préserve les annonces d'autres auteurs, remplace celles de tachibana")
    args = ap.parse_args()

    items, skipped = articles_to_items(args.db, args.lang)

    if args.merge_into:
        try:
            with open(args.merge_into, encoding="utf-8") as f:
                existing = json.load(f) or []
        except FileNotFoundError:
            existing = []
        kept = [it for it in existing if it.get("author") != AUTHOR]
        items = sorted(kept + items, key=lambda it: it.get("at", ""))

    data = json.dumps(items, ensure_ascii=False, indent=2)
    if args.out == "-":
        print(data)
    else:
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(data + "\n")
    print("Actualites tachibana (%s) : %d article(s) importe(s), %d ignore(s) -> %s"
          % (args.lang, len(items), skipped, args.out), file=sys.stderr)


if __name__ == "__main__":
    main()
