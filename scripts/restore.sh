#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# restore.sh — restaure une sauvegarde de l'état du BBS Oric.
#
# Restaure une archive produite par backup.sh : comptes (users.json),
# bibliothèque de fichiers (files/) et contenu (site.json).
#
# Procédure sûre :
#   1. arrête le service (état figé) ;
#   2. écarte l'état courant en .pre-restore (annulable) ;
#   3. restaure depuis l'archive ;
#   4. redémarre le service — systemd, avec DynamicUser + StateDirectory,
#      réapproprie récursivement le StateDirectory à l'uid courant au
#      démarrage : les fichiers restaurés par root redeviennent lisibles
#      par le service. (Voir docs/backup.md.)
#
# Usage :
#   restore.sh <archive.tar.gz>     restaure cette archive
#   restore.sh latest               restaure la plus récente de BBS_BACKUP_DIR
#   restore.sh --list               liste les archives disponibles
#
# Variables d'environnement (toutes optionnelles) :
#   BBS_STATE_DIR    répertoire d'état RW      (défaut /var/lib/bbsoric)
#   BBS_CONTENT      fichier de contenu        (défaut /etc/bbsoric/site.json)
#   BBS_BACKUP_DIR   dossier des archives      (défaut /var/backups/bbsoric)
#   BBS_SERVICE      unité systemd à cycler    (défaut bbsoric ; vide = ne
#                                               touche pas à systemd — tests)
#   BBS_ASSUME_YES   1 = ne pas demander confirmation (équivaut à -y)
# ---------------------------------------------------------------------------
set -uo pipefail

STATE_DIR="${BBS_STATE_DIR:-/var/lib/bbsoric}"
CONTENT="${BBS_CONTENT:-/etc/bbsoric/site.json}"
BACKUP_DIR="${BBS_BACKUP_DIR:-/var/backups/bbsoric}"
SERVICE="${BBS_SERVICE-bbsoric}"
ASSUME_YES="${BBS_ASSUME_YES:-0}"

log() { echo "[restore] $*" >&2; }

ARG="${1:-}"
for a in "$@"; do
	[ "$a" = "-y" ] && ASSUME_YES=1
done

case "$ARG" in
	""|-h|--help)
		sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
		exit 0 ;;
	--list)
		ls -1t "$BACKUP_DIR"/bbsoric-backup-*.tar.gz 2>/dev/null \
			|| { log "aucune archive dans $BACKUP_DIR"; exit 1; }
		exit 0 ;;
	latest)
		ARCHIVE="$(ls -1t "$BACKUP_DIR"/bbsoric-backup-*.tar.gz 2>/dev/null | head -1)"
		[ -n "$ARCHIVE" ] || { log "ERREUR : aucune archive dans $BACKUP_DIR."; exit 1; } ;;
	*)
		ARCHIVE="$ARG" ;;
esac

[ -f "$ARCHIVE" ] || { log "ERREUR : archive introuvable : $ARCHIVE"; exit 1; }
log "archive : $ARCHIVE"

# Espace de travail temporaire (nettoyé en sortie).
WORK="$(mktemp -d)" || { log "ERREUR : mktemp impossible."; exit 1; }
trap 'rm -rf "$WORK"' EXIT

if ! tar -xzf "$ARCHIVE" -C "$WORK"; then
	log "ERREUR : extraction impossible (archive corrompue ?)."
	exit 1
fi
PAYLOAD="$(find "$WORK" -maxdepth 1 -mindepth 1 -type d | head -1)"
[ -n "$PAYLOAD" ] || { log "ERREUR : archive vide / format inattendu."; exit 1; }

if [ -f "$PAYLOAD/MANIFEST.txt" ]; then
	log "contenu de la sauvegarde :"
	sed 's/^/    /' "$PAYLOAD/MANIFEST.txt" >&2
fi

# Confirmation (sauf -y) — la restauration écrase l'état courant.
if [ "$ASSUME_YES" != "1" ]; then
	printf '[restore] Écraser l’état courant (%s, %s) ? [oui/NON] ' "$STATE_DIR" "$CONTENT" >&2
	read -r ans
	case "$ans" in
		oui|OUI|o|O|y|Y|yes) ;;
		*) log "abandon."; exit 1 ;;
	esac
fi

# 1. Arrêt du service (si systemd géré).
if [ -n "$SERVICE" ] && command -v systemctl >/dev/null 2>&1; then
	log "arrêt de $SERVICE"
	systemctl stop "$SERVICE" || log "ATTENTION : arrêt de $SERVICE en échec (on continue)."
fi

# 2. État RW (comptes + fichiers).
if [ -d "$PAYLOAD/state" ]; then
	if [ -e "$STATE_DIR" ]; then
		rm -rf "$STATE_DIR.pre-restore"
		mv "$STATE_DIR" "$STATE_DIR.pre-restore" \
			&& log "état courant écarté → $STATE_DIR.pre-restore"
	fi
	mkdir -p "$(dirname "$STATE_DIR")"
	cp -a "$PAYLOAD/state" "$STATE_DIR" || { log "ERREUR : restauration de l'état."; exit 1; }
	log "état restauré → $STATE_DIR"
else
	log "ATTENTION : pas d'état dans l'archive — $STATE_DIR inchangé."
fi

# 3. Contenu site.json.
if [ -f "$PAYLOAD/site.json" ]; then
	if [ -f "$CONTENT" ]; then
		cp -a "$CONTENT" "$CONTENT.pre-restore" && log "contenu courant → $CONTENT.pre-restore"
	fi
	mkdir -p "$(dirname "$CONTENT")"
	cp -a "$PAYLOAD/site.json" "$CONTENT" || { log "ERREUR : restauration du contenu."; exit 1; }
	log "contenu restauré → $CONTENT"
else
	log "ATTENTION : pas de site.json dans l'archive — $CONTENT inchangé."
fi

# 4. Redémarrage : systemd réapproprie le StateDirectory (DynamicUser).
if [ -n "$SERVICE" ] && command -v systemctl >/dev/null 2>&1; then
	log "redémarrage de $SERVICE"
	systemctl start "$SERVICE" || { log "ERREUR : démarrage de $SERVICE."; exit 1; }
	sleep 2
	if [ "$(systemctl is-active "$SERVICE" 2>/dev/null)" = "active" ]; then
		log "OK : $SERVICE actif."
	else
		log "ERREUR : $SERVICE n'est pas actif après restauration."
		exit 1
	fi
fi

log "restauration terminée."
exit 0
