#!/bin/bash
# Déploie le BBS Oric (bbsoric) sur le serveur de production (LXC pavi3617),
# en reprenant le mécanisme du projet telenet :
#   1. compile le binaire Go (linux/amd64, statique)
#   2. le copie sur le serveur
#   3. installe/met à jour l'unité systemd bbsoric.service
#   4. redémarre le service et vérifie qu'il écoute sur le port BBS
#
# Prérequis : VPN WireGuard mustang actif
#   sudo wg-quick up ~/.wireguard/mustang.conf
#
# Usage : deploy/vps-deploy.sh [--no-restart] [--build-only]
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ ! -f "$SCRIPT_DIR/deploy.conf" ]; then
    echo "ERREUR : deploy/deploy.conf manquant."
    echo "  cp deploy/deploy.conf.example deploy/deploy.conf  puis renseignez-le."
    exit 1
fi
source "$SCRIPT_DIR/deploy.conf"

SSH="ssh -p $VPS_PORT -o ConnectTimeout=8 $VPS_USER@$VPS_HOST"
SCP="scp -P $VPS_PORT -o ConnectTimeout=8"

DO_RESTART=true
BUILD_ONLY=false
for arg in "$@"; do
    case $arg in
        --no-restart) DO_RESTART=false ;;
        --build-only) BUILD_ONLY=true ;;
    esac
done

echo "=== Déploiement BBS Oric sur $VPS_HOST (telnet $BBS_PORT) ==="

# 0. Refuser de déployer du code non commité (traçabilité)
if [ -n "$(cd "$ROOT" && git status --porcelain)" ]; then
    echo "ERREUR : modifications non commitées. Commitez avant de déployer."
    exit 1
fi

# 1. Compilation
echo "--- Compilation (linux/amd64, statique) ---"
( cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /tmp/bbsoric ./server/cmd/bbsd )
echo "  binaire : $(stat -c%s /tmp/bbsoric) octets"
$BUILD_ONLY && { echo "(--build-only) arrêt avant déploiement"; exit 0; }

# 2. Connectivité
if ! $SSH "echo OK" >/dev/null 2>&1; then
    echo "ERREUR : $VPS_HOST injoignable."
    echo "VPN WireGuard mustang actif ? : sudo wg-quick up ~/.wireguard/mustang.conf"
    exit 1
fi

# 3. Copie du binaire (atomique : .new puis mv)
echo "--- Copie du binaire ---"
$SCP /tmp/bbsoric "$VPS_USER@$VPS_HOST:$BINAIRE_REMOTE.new"
$SSH "chmod 755 $BINAIRE_REMOTE.new && mv -f $BINAIRE_REMOTE.new $BINAIRE_REMOTE"

# 3b. Contenu JSON — SEMÉ UNE SEULE FOIS. Les éditions à chaud faites directement
# sur le serveur (/etc/bbsoric/site.json) ne sont jamais écrasées par le déploiement.
echo "--- Contenu site.json ---"
$SSH "mkdir -p /etc/bbsoric"
if $SSH "test -f /etc/bbsoric/site.json"; then
    echo "  /etc/bbsoric/site.json existe déjà — conservé (éditions à chaud préservées)"
else
    $SCP "$ROOT/content/site.json" "$VPS_USER@$VPS_HOST:/etc/bbsoric/site.json"
    echo "  /etc/bbsoric/site.json installé (contenu initial)"
fi

# 4. Unité systemd
echo "--- Unité systemd $SERVICE.service ---"
$SCP "$SCRIPT_DIR/bbsoric.service" "$VPS_USER@$VPS_HOST:/etc/systemd/system/$SERVICE.service"
$SSH "systemctl daemon-reload && systemctl enable $SERVICE >/dev/null 2>&1 || true"

# 4b. Supervision : script de sonde + timer systemd (healthz/TCP + alerte)
echo "--- Supervision ($SERVICE-monitor.timer) ---"
$SCP "$ROOT/scripts/monitor.sh" "$VPS_USER@$VPS_HOST:/usr/local/bin/bbsoric-monitor.sh"
$SSH "chmod 755 /usr/local/bin/bbsoric-monitor.sh"
$SCP "$SCRIPT_DIR/bbsoric-monitor.service" "$VPS_USER@$VPS_HOST:/etc/systemd/system/$SERVICE-monitor.service"
$SCP "$SCRIPT_DIR/bbsoric-monitor.timer" "$VPS_USER@$VPS_HOST:/etc/systemd/system/$SERVICE-monitor.timer"
$SSH "systemctl daemon-reload && systemctl enable --now $SERVICE-monitor.timer >/dev/null 2>&1 || true"

# 4c. Sauvegardes : script de backup + timer systemd (quotidien, rotation)
echo "--- Sauvegardes ($SERVICE-backup.timer) ---"
$SCP "$ROOT/scripts/backup.sh" "$VPS_USER@$VPS_HOST:/usr/local/bin/bbsoric-backup.sh"
$SSH "chmod 755 /usr/local/bin/bbsoric-backup.sh"
$SCP "$SCRIPT_DIR/bbsoric-backup.service" "$VPS_USER@$VPS_HOST:/etc/systemd/system/$SERVICE-backup.service"
$SCP "$SCRIPT_DIR/bbsoric-backup.timer" "$VPS_USER@$VPS_HOST:/etc/systemd/system/$SERVICE-backup.timer"
# Script de restauration disponible sur le serveur (exécution manuelle).
$SCP "$ROOT/scripts/restore.sh" "$VPS_USER@$VPS_HOST:/usr/local/bin/bbsoric-restore.sh"
$SSH "chmod 755 /usr/local/bin/bbsoric-restore.sh"
$SSH "systemctl daemon-reload && systemctl enable --now $SERVICE-backup.timer >/dev/null 2>&1 || true"

# 5. Redémarrage + vérification
if $DO_RESTART; then
    echo "--- Restart de $SERVICE ---"
    $SSH "systemctl restart $SERVICE"
    sleep 2
    ETAT=$($SSH "systemctl is-active $SERVICE" 2>/dev/null || true)
    if [ "$ETAT" = "active" ]; then
        echo "  OK : service $SERVICE actif"
        $SSH "ss -tlnp 2>/dev/null | grep ':$BBS_PORT' >/dev/null \
            && echo '  OK : écoute sur $BBS_PORT' \
            || echo '  ATTENTION : pas d écoute sur $BBS_PORT'"
    else
        echo "  ERREUR : service $SERVICE état=$ETAT"
        $SSH "journalctl -u $SERVICE --no-pager -n 20"
        exit 1
    fi
fi

echo ""
echo "=== Déploiement terminé : BBS Oric sur $VPS_HOST:$BBS_PORT (telnet) ==="
