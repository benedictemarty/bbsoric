#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Tests unitaires de scripts/gen-news-tachibana.py (import articles -> news BBS)."""
import importlib.util
import json
import os
import sqlite3
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC = importlib.util.spec_from_file_location("gennews", os.path.join(HERE, "gen-news-tachibana.py"))
gn = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gn)

PASS = 0
FAIL = 0


def check(cond, label):
    global PASS, FAIL
    if cond:
        print("  ok   :", label); PASS += 1
    else:
        print("  FAIL :", label); FAIL += 1


# --- helpers purs -------------------------------------------------------------
check(gn.deaccent("actualités œuvre à côté") == "actualites oeuvre a cote",
      "deaccent translittère les accents (é→e, œ→oe)")
check(gn.ascii_clean("<b>Bonjour</b>&nbsp;à   tous", 400) == "Bonjour a tous",
      "ascii_clean décode entités, compacte espaces, translittère")
check(gn.ascii_clean("x" * 100, 38) == "x" * 38, "ascii_clean borne à maxlen")
check(gn.to_rfc3339("2026-09-03 10:20:30") == "2026-09-03T10:20:30Z", "to_rfc3339 accepte 'YYYY-MM-DD HH:MM:SS'")
check(gn.to_rfc3339("2026-09-03T10:20:30Z") == "2026-09-03T10:20:30Z", "to_rfc3339 accepte l'ISO/Z")
check(gn.to_rfc3339("pas une date") is None, "to_rfc3339 renvoie None si non interprétable")
check(gn.body_text("", "<p>Salut</p><p>toi</p>").strip().startswith("Salut"),
      "body_text prend le texte du HTML si résumé vide")
check(gn.body_text("Le resume", "<p>corps</p>") == "Le resume", "body_text préfère le résumé")


# --- build sur une base jetable -----------------------------------------------
with tempfile.TemporaryDirectory() as d:
    db = os.path.join(d, "t.sqlite")
    con = sqlite3.connect(db)
    con.execute("CREATE TABLE articles (id INTEGER PRIMARY KEY, lang TEXT, slug TEXT, title TEXT, "
                "summary TEXT, body TEXT, published INT, created_at TEXT, updated_at TEXT)")
    con.executemany(
        "INSERT INTO articles (lang,slug,title,summary,body,published,created_at,updated_at) "
        "VALUES (?,?,?,?,?,?,?,?)",
        [
            ("fr", "a2", "Deuxieme", "Résumé accentué", "<p>x</p>", 1, "2026-02-02 09:00:00", ""),
            ("fr", "a1", "Premier", "", "<p>Corps <b>HTML</b> à lire</p>", 1, "2026-01-01 09:00:00", ""),
            ("fr", "brouillon", "Brouillon", "s", "b", 0, "2026-03-01 09:00:00", ""),   # non publié -> exclu
            ("en", "en1", "English", "s", "b", 1, "2026-01-15 09:00:00", ""),           # autre langue -> exclu
            ("fr", "nodate", "Sans date", "s", "b", 1, "pas une date", ""),             # date KO -> ignoré
            ("fr", "nobody", "Sans corps", "", "", 1, "2026-04-01 09:00:00", ""),       # corps vide -> ignoré
        ])
    con.commit(); con.close()

    items, skipped = gn.articles_to_items(db, "fr")
    check(len(items) == 2, "articles_to_items : 2 articles fr publiés retenus")
    check(skipped == 2, "articles_to_items : 2 ignorés (date KO + corps vide)")
    # ordre chronologique croissant (a1 avant a2)
    check([it["title"] for it in items] == ["Premier", "Deuxieme"], "ordre chronologique (le plus récent en dernier)")
    check(items[0]["body"] == "Corps HTML a lire", "corps : HTML retiré + accents translittérés")
    check(items[1]["body"] == "Resume accentue", "corps : résumé préféré + translittéré")
    check(all(it["author"] == "tachibana.eu" for it in items), "author = tachibana.eu")
    check(items[0]["at"] == "2026-01-01T09:00:00Z", "date normalisée RFC3339")

    # --- merge-into : préserve les annonces d'autres auteurs ------------------
    existing = os.path.join(d, "news.json")
    json.dump([
        {"title": "Sysop", "body": "annonce locale", "author": "sysop", "at": "2026-01-05T00:00:00Z"},
        {"title": "Vieux tachi", "body": "a remplacer", "author": "tachibana.eu", "at": "2025-01-01T00:00:00Z"},
    ], open(existing, "w"))
    # simule la fusion (comme main --merge-into)
    kept = [it for it in json.load(open(existing)) if it.get("author") != gn.AUTHOR]
    merged = sorted(kept + items, key=lambda it: it["at"])
    titles = [it["title"] for it in merged]
    check("Sysop" in titles and "Vieux tachi" not in titles,
          "merge : annonce sysop préservée, ancienne tachibana remplacée")
    check(titles == sorted(titles, key=lambda t: {"Premier": 1, "Sysop": 2, "Deuxieme": 3}[t]),
          "merge : tri chronologique croissant (Premier<Sysop<Deuxieme)")

print()
print("Bilan: %d ok, %d ko" % (PASS, FAIL))
sys.exit(1 if FAIL else 0)
