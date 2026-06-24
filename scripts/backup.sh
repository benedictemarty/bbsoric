#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# backup.sh — sauvegarde de l'état persistant du BBS Oric.
#
# Archive, dans un .tar.gz horodaté et avec rotation, tout ce qui n'est pas
# reproductible depuis le dépôt :
#   - le StateDirectory du service  (comptes hachés users.json + bibliothèque
#     de fichiers files/) ;
#   - le contenu site.json (éditable à chaud sur le serveur, donc à préserver).
#
# Conçu pour tourner SUR le serveur de production, déclenché par
# bbsoric-backup.timer (quotidien). Sans dépendance autre que tar/gzip.
#
# La sauvegarde est « à chaud » : users.json et les fichiers de la
# bibliothèque sont écrits de façon atomique (write-temp + rename) par le
# serveur, donc l'archive ne capture jamais d'écriture partielle. Inutile
# d'arrêter le service.
#
# Variables d'environnement (toutes optionnelles) :
#   BBS_STATE_DIR    répertoire d'état RW       (défaut /var/lib/bbsoric)
#   BBS_CONTENT      fichier de contenu         (défaut /etc/bbsoric/site.json)
#   BBS_BACKUP_DIR   destination des archives   (défaut /var/backups/bbsoric)
#   BBS_BACKUP_KEEP  nombre d'archives gardées  (défaut 14 ; 0 = pas de purge)
#   BBS_BACKUP_STAMP horodatage forcé (tests)   (défaut : date courante)
#
# Sortie : chemin de l'archive créée (stdout). Journal sur stderr.
# Code de sortie : 0 = archive créée, 1 = échec.
# ---------------------------------------------------------------------------
set -uo pipefail

STATE_DIR="${BBS_STATE_DIR:-/var/lib/bbsoric}"
CONTENT="${BBS_CONTENT:-/etc/bbsoric/site.json}"
BACKUP_DIR="${BBS_BACKUP_DIR:-/var/backups/bbsoric}"
KEEP="${BBS_BACKUP_KEEP:-14}"
STAMP="${BBS_BACKUP_STAMP:-$(date +%Y%m%d-%H%M%S)}"

log() { echo "[backup] $*" >&2; }

# Au moins une source doit exister, sinon il n'y a rien à sauvegarder.
if [ ! -d "$STATE_DIR" ] && [ ! -f "$CONTENT" ]; then
	log "ERREUR : ni le StateDirectory ($STATE_DIR) ni le contenu ($CONTENT) n'existent."
	exit 1
fi

mkdir -p "$BACKUP_DIR" || { log "ERREUR : création de $BACKUP_DIR impossible."; exit 1; }

# Espace de travail temporaire (nettoyé en sortie).
WORK="$(mktemp -d)" || { log "ERREUR : mktemp impossible."; exit 1; }
trap 'rm -rf "$WORK"' EXIT

PAYLOAD="bbsoric-backup-$STAMP"
ROOT="$WORK/$PAYLOAD"
mkdir -p "$ROOT"

# 1. État RW (comptes + bibliothèque de fichiers).
if [ -d "$STATE_DIR" ]; then
	cp -a "$STATE_DIR" "$ROOT/state" || { log "ERREUR : copie de $STATE_DIR."; exit 1; }
	log "état inclus : $STATE_DIR"
else
	log "ATTENTION : $STATE_DIR absent — non inclus."
fi

# 2. Contenu site.json.
if [ -f "$CONTENT" ]; then
	cp -a "$CONTENT" "$ROOT/site.json" || { log "ERREUR : copie de $CONTENT."; exit 1; }
	log "contenu inclus : $CONTENT"
else
	log "ATTENTION : $CONTENT absent — non inclus."
fi

# 3. Manifeste (traçabilité de la restauration).
{
	echo "BBS Oric — sauvegarde"
	echo "horodatage : $STAMP"
	echo "hôte       : $(hostname 2>/dev/null || echo inconnu)"
	echo "state_dir  : $STATE_DIR"
	echo "content    : $CONTENT"
	if [ -f "$ROOT/state/users.json" ]; then
		echo "comptes    : $(grep -o '"name"' "$ROOT/state/users.json" 2>/dev/null | wc -l) (estimation)"
	fi
	if [ -d "$ROOT/state/files" ]; then
		echo "fichiers   : $(find "$ROOT/state/files" -type f 2>/dev/null | wc -l)"
	fi
} > "$ROOT/MANIFEST.txt"

# 4. Archive atomique : .tmp puis rename.
ARCHIVE="$BACKUP_DIR/$PAYLOAD.tar.gz"
if ! tar -czf "$ARCHIVE.tmp" -C "$WORK" "$PAYLOAD"; then
	log "ERREUR : création de l'archive."
	rm -f "$ARCHIVE.tmp"
	exit 1
fi
chmod 600 "$ARCHIVE.tmp"
mv -f "$ARCHIVE.tmp" "$ARCHIVE"
log "archive créée : $ARCHIVE ($(stat -c%s "$ARCHIVE" 2>/dev/null || echo '?') octets)"

# 5. Rotation : ne garder que les KEEP plus récentes.
if [ "$KEEP" -gt 0 ] 2>/dev/null; then
	mapfile -t OLD < <(ls -1t "$BACKUP_DIR"/bbsoric-backup-*.tar.gz 2>/dev/null | tail -n +"$((KEEP + 1))")
	for f in "${OLD[@]}"; do
		rm -f "$f" && log "purge (rotation) : $(basename "$f")"
	done
fi

echo "$ARCHIVE"
exit 0
