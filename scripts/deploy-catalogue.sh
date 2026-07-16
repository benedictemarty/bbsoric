#!/bin/bash
# Déploie le CATALOGUE (Logiciels / Magazines / Livres) sur le serveur de prod :
#   1. récupère le site.json de prod (préserve les éditions à chaud) ;
#   2. y greffe le catalogue (source + pages + entrée de menu) et copie localement
#      les fichiers téléchargeables (petits .tap) dans un staging ;
#   3. valide le site fusionné avec internal/content ;
#   4. (sauf --dry-run) rsync des fichiers vers -files, dépose le site.json fusionné,
#      REDÉMARRE le service (obligatoire : une nouvelle source DataWindow est semée
#      au démarrage) et vérifie qu'il écoute.
#
# Prérequis :
#   - VPN WireGuard mustang actif (comme deploy/vps-deploy.sh)
#   - deploy/deploy.conf rempli
#   - ORIC_LIB = chemin de la bibliothèque OricProgramsLib (env ou deploy.conf)
#
# Usage : scripts/deploy-catalogue.sh [--dry-run] [--limit N]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/deploy/deploy.conf"
: "${VPS_HOST:?deploy.conf incomplet}" "${VPS_USER:?}" "${VPS_PORT:?}" "${SERVICE:?}"
LIB="${ORIC_LIB:?definir ORIC_LIB=chemin/vers/OricProgramsLib (env ou deploy.conf)}"

DRY=false
LIMIT=0
while [ $# -gt 0 ]; do
    case "$1" in
        --dry-run) DRY=true ;;
        --limit) LIMIT="$2"; shift ;;
        *) echo "option inconnue : $1" >&2; exit 2 ;;
    esac
    shift
done

SSH="ssh -p $VPS_PORT -o ConnectTimeout=8 $VPS_USER@$VPS_HOST"
REMOTE_FILES="/var/lib/bbsoric/files"
REMOTE_CONTENT="/etc/bbsoric/site.json"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
FILES="$STAGE/files"
mkdir -p "$FILES"

echo "=== Catalogue -> $VPS_HOST ($SERVICE) ==="

echo "--- 1. Récupération du site.json de prod (préserve les éditions à chaud) ---"
if $SSH "test -f $REMOTE_CONTENT"; then
    $SSH "cat $REMOTE_CONTENT" > "$STAGE/site-prod.json"
else
    echo "  (aucun site.json distant : on part de content/site.json local)"
    cp "$ROOT/content/site.json" "$STAGE/site-prod.json"
fi

echo "--- 2. Fusion du catalogue + copie des fichiers téléchargeables ---"
ARGS=(--lib "$LIB" --merge-into "$STAGE/site-prod.json" --copy-files "$FILES" --out "$STAGE/site.json")
[ "$LIMIT" -gt 0 ] && ARGS+=(--limit "$LIMIT")
python3 "$ROOT/scripts/gen-catalogue.py" "${ARGS[@]}"
NFILES=$(find "$FILES" -type f | wc -l | tr -d ' ')
echo "  fichiers téléchargeables copiés en staging : $NFILES"

echo "--- 3. Validation du site fusionné (internal/content) ---"
( cd "$ROOT" && go run ./tools/validate-content "$STAGE/site.json" )

if $DRY; then
    echo "--- [dry-run] rien envoyé. Staging conservé n'est PAS supprimé : ---"
    trap - EXIT
    echo "  site fusionné : $STAGE/site.json"
    echo "  fichiers      : $FILES ($NFILES)"
    echo "  À faire pour de vrai : relancer sans --dry-run."
    exit 0
fi

echo "--- 4. Envoi des fichiers -> $REMOTE_FILES ---"
$SSH "mkdir -p $REMOTE_FILES"
rsync -az --info=stats1 -e "ssh -p $VPS_PORT -o ConnectTimeout=8" "$FILES/" "$VPS_USER@$VPS_HOST:$REMOTE_FILES/"

echo "--- 5. Dépôt du site.json fusionné (atomique) ---"
scp -P "$VPS_PORT" -o ConnectTimeout=8 "$STAGE/site.json" "$VPS_USER@$VPS_HOST:$REMOTE_CONTENT.new"
$SSH "mv $REMOTE_CONTENT.new $REMOTE_CONTENT"

echo "--- 6. Redémarrage du service (semis de la nouvelle source au boot) ---"
$SSH "systemctl restart $SERVICE"
sleep 2
ETAT=$($SSH "systemctl is-active $SERVICE" 2>/dev/null || true)
echo "  service : $ETAT"
if $SSH "ss -ltn 2>/dev/null | grep -q ':${BBS_PORT:-6502} '"; then
    echo "  OK : écoute sur ${BBS_PORT:-6502}"
else
    echo "  ATTENTION : le service n'écoute pas sur ${BBS_PORT:-6502} — vérifier journalctl -u $SERVICE" >&2
    exit 1
fi
echo "=== Catalogue déployé ==="
