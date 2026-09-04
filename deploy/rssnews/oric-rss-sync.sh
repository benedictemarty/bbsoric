#!/bin/bash
# Synchro des ACTUALITÉS du BBS depuis un flux RSS/Atom externe.
# Tourne sur l'hôte Proxmox kiranerys (accès `pct` au conteneur bbsoric) : va
# chercher le flux, le convertit en annonces (binaire rssnews), fusionne dans le
# news.json du BBS (CT 510) en préservant les autres auteurs, et ne pousse +
# redémarre QUE si le contenu a changé (idempotent, comme la synchro tachibana).
#
# Prérequis : binaire rssnews déployé sur l'hôte (make rssnews puis scp).
#
# Config (env — À RENSEIGNER, pas de valeur par défaut inventée pour l'URL) :
#   FEED_URL=https://exemple.org/rss.xml   (obligatoire)
#   FEED_AUTHOR=exemple.org                (auteur affiché ; défaut = titre du flux)
#   CT_BBS=510  MAX_ITEMS=20
set -euo pipefail

FEED_URL="${FEED_URL:?définir FEED_URL (URL du flux RSS/Atom)}"
FEED_AUTHOR="${FEED_AUTHOR:-}"
CT_BBS="${CT_BBS:-510}"
MAX_ITEMS="${MAX_ITEMS:-20}"
NEWS_REMOTE="/var/lib/bbsoric/news.json"
RSSNEWS="/usr/local/bin/rssnews"

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT

# 1. news.json courant du BBS (peut ne pas exister).
pct exec "$CT_BBS" -- cat "$NEWS_REMOTE" 2>/dev/null > "$tmp/news-cur.json" || true
[ -s "$tmp/news-cur.json" ] || echo "[]" > "$tmp/news-cur.json"

# 2. Import du flux, fusionné avec l'existant (préserve les autres auteurs).
"$RSSNEWS" -feed "$FEED_URL" ${FEED_AUTHOR:+-author "$FEED_AUTHOR"} \
    -max "$MAX_ITEMS" --merge-into "$tmp/news-cur.json" --out "$tmp/news-new.json"

# 3. Pousse + redémarre uniquement si le contenu diffère (comparaison sémantique).
if python3 -c "import json,sys; a=json.load(open('$tmp/news-cur.json')); b=json.load(open('$tmp/news-new.json')); sys.exit(0 if a==b else 1)"; then
    echo "actualites inchangees ($(python3 -c "import json;print(len(json.load(open('$tmp/news-new.json'))))") annonces) — rien a faire"
    exit 0
fi

pct push "$CT_BBS" "$tmp/news-new.json" "$NEWS_REMOTE" --perms 0644
pct exec "$CT_BBS" -- systemctl restart bbsoric
echo "actualites mises a jour ($(python3 -c "import json;print(len(json.load(open('$tmp/news-new.json'))))") annonces) + restart bbsoric"
