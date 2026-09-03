#!/bin/bash
# Synchro des ACTUALITÉS du BBS depuis les articles de tachibana.eu.
# Tourne sur l'hôte Proxmox kiranerys (accès `pct` aux deux conteneurs) : lit la
# base tachibana (CT 716), génère le news.json, et ne le pousse au BBS (CT 510) +
# redémarre QUE si le contenu a changé (comparaison sémantique, pas de restart inutile).
# Déployé par systemd : oric-news-sync.service + .timer (quotidien).
#
# Config (env, valeurs par défaut adaptées à l'infra actuelle) :
#   CT_TACHI=716  CT_BBS=510  NEWS_LANG=fr
set -euo pipefail

CT_TACHI="${CT_TACHI:-716}"
CT_BBS="${CT_BBS:-510}"
NEWS_LANG="${NEWS_LANG:-fr}"
DB_REMOTE="/var/lib/tachibana/tachibana.sqlite"
NEWS_REMOTE="/var/lib/bbsoric/news.json"
GEN="/usr/local/bin/gen-news-tachibana.py"

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT

# 1. Copie de la base tachibana (lecture seule).
pct exec "$CT_TACHI" -- cat "$DB_REMOTE" > "$tmp/tachi.sqlite"
[ -s "$tmp/tachi.sqlite" ] || { echo "base tachibana vide/introuvable" >&2; exit 1; }

# 2. news.json courant du BBS (peut ne pas exister).
pct exec "$CT_BBS" -- cat "$NEWS_REMOTE" 2>/dev/null > "$tmp/news-cur.json" || true
[ -s "$tmp/news-cur.json" ] || echo "[]" > "$tmp/news-cur.json"

# 3. Génération (merge : préserve les annonces d'autres auteurs que tachibana.eu).
python3 "$GEN" --db "$tmp/tachi.sqlite" --lang "$NEWS_LANG" \
    --merge-into "$tmp/news-cur.json" --out "$tmp/news-new.json"

# 4. Pousse + redémarre uniquement si le contenu diffère (comparaison sémantique).
if python3 -c "import json,sys; a=json.load(open('$tmp/news-cur.json')); b=json.load(open('$tmp/news-new.json')); sys.exit(0 if a==b else 1)"; then
    echo "actualites inchangees ($(python3 -c "import json;print(len(json.load(open('$tmp/news-new.json'))))") annonces) — rien a faire"
    exit 0
fi

pct push "$CT_BBS" "$tmp/news-new.json" "$NEWS_REMOTE" --perms 0644
pct exec "$CT_BBS" -- systemctl restart bbsoric
echo "actualites mises a jour ($(python3 -c "import json;print(len(json.load(open('$tmp/news-new.json'))))") annonces) + restart bbsoric"
