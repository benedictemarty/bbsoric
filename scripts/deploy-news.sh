#!/bin/bash
# Déploie les ACTUALITÉS du BBS depuis les articles de tachibana.eu :
#   1. (au besoin) snapshot de la base tachibana (pull-tachibana.sh) ;
#   2. récupère le news.json de prod (préserve les annonces d'autres auteurs) ;
#   3. y fusionne les articles tachibana (gen-news-tachibana.py --merge-into) ;
#   4. valide le JSON, (sauf --dry-run) le dépose sur prod et REDÉMARRE le service
#      (le store news est chargé au démarrage, pas relu à chaud comme le site.json).
#
# Prérequis : VPN mustang actif, deploy/deploy.conf rempli, service lancé avec -news.
# Config (env) : TACHI_OUT (défaut /tmp/tachibana), NEWS_LANG (défaut fr).
#
# Usage : scripts/deploy-news.sh [--dry-run] [--pull]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/deploy/deploy.conf"
: "${VPS_HOST:?deploy.conf incomplet}" "${VPS_USER:?}" "${VPS_PORT:?}" "${SERVICE:?}"

DRY=false; PULL=false
while [ $# -gt 0 ]; do
    case "$1" in
        --dry-run) DRY=true ;;
        --pull) PULL=true ;;
        *) echo "option inconnue : $1" >&2; exit 2 ;;
    esac
    shift
done

TACHI_OUT="${TACHI_OUT:-/tmp/tachibana}"
NEWS_LANG="${NEWS_LANG:-fr}"
SSH="ssh -p $VPS_PORT -o ConnectTimeout=8 $VPS_USER@$VPS_HOST"
REMOTE_NEWS="/var/lib/bbsoric/news.json"

STAGE="$(mktemp -d)"; trap 'rm -rf "$STAGE"' EXIT

echo "=== Actualités (articles tachibana) -> $VPS_HOST ($SERVICE) ==="

if $PULL || [ ! -s "$TACHI_OUT/tachibana.sqlite" ]; then
    echo "--- snapshot tachibana (pull-tachibana.sh) ---"
    OUT="$TACHI_OUT" "$ROOT/scripts/pull-tachibana.sh"
fi

echo "--- 1. Récupération du news.json de prod (préserve les autres auteurs) ---"
$SSH "cat $REMOTE_NEWS 2>/dev/null" > "$STAGE/news-prod.json" || true
[ -s "$STAGE/news-prod.json" ] || echo "[]" > "$STAGE/news-prod.json"

echo "--- 2. Fusion des articles tachibana ($NEWS_LANG) ---"
python3 "$ROOT/scripts/gen-news-tachibana.py" --db "$TACHI_OUT/tachibana.sqlite" \
    --lang "$NEWS_LANG" --merge-into "$STAGE/news-prod.json" --out "$STAGE/news.json"

echo "--- 3. Validation JSON ---"
python3 -c "import json,sys; d=json.load(open('$STAGE/news.json')); assert isinstance(d,list); print('  %d annonce(s)'%len(d))"

if $DRY; then
    trap - EXIT
    echo "--- [dry-run] rien envoyé. news fusionné : $STAGE/news.json ---"
    exit 0
fi

echo "--- 4. Dépôt (atomique) + redémarrage ---"
scp -P "$VPS_PORT" -o ConnectTimeout=8 "$STAGE/news.json" "$VPS_USER@$VPS_HOST:$REMOTE_NEWS.new"
$SSH "chmod 644 $REMOTE_NEWS.new && mv $REMOTE_NEWS.new $REMOTE_NEWS && systemctl restart $SERVICE"
sleep 2
ETAT=$($SSH "systemctl is-active $SERVICE" 2>/dev/null || true)
echo "  service : $ETAT"
$SSH "ss -ltn 2>/dev/null | grep -q ':${BBS_PORT:-6502} '" && echo "  OK : écoute sur ${BBS_PORT:-6502}" || {
    echo "  ATTENTION : le service n'écoute pas — vérifier journalctl -u $SERVICE" >&2; exit 1; }
echo "=== Actualités déployées ==="
