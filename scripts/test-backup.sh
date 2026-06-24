#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# test-backup.sh — test bout-en-bout de backup.sh / restore.sh.
#
# Vérifie, dans un bac à sable temporaire (sans systemd ni root) :
#   1. une sauvegarde capture comptes + fichiers + contenu ;
#   2. une restauration rétablit fidèlement un état modifié/effacé ;
#   3. la rotation ne garde que les N archives les plus récentes.
#
# Code de sortie : 0 = tous les cas verts, 1 = au moins un échec.
# ---------------------------------------------------------------------------
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
BACKUP="$HERE/backup.sh"
RESTORE="$HERE/restore.sh"

SANDBOX="$(mktemp -d)"
trap 'rm -rf "$SANDBOX"' EXIT

export BBS_STATE_DIR="$SANDBOX/state"
export BBS_CONTENT="$SANDBOX/etc/site.json"
export BBS_BACKUP_DIR="$SANDBOX/backups"
export BBS_SERVICE=""        # pas de systemd dans le test
export BBS_ASSUME_YES=1

PASS=0
FAIL=0
ok()   { echo "  ok   : $*"; PASS=$((PASS + 1)); }
ko()   { echo "  FAIL : $*"; FAIL=$((FAIL + 1)); }

# --- Fixture : un état initial réaliste -----------------------------------
mkdir -p "$BBS_STATE_DIR/files" "$(dirname "$BBS_CONTENT")"
printf '{"users":[{"name":"alice"},{"name":"bob"}]}' > "$BBS_STATE_DIR/users.json"
printf 'BONJOUR ORIC' > "$BBS_STATE_DIR/files/readme.txt"
printf '{"pages":{}}' > "$BBS_CONTENT"

USERS_REF="$(cat "$BBS_STATE_DIR/users.json")"
FILE_REF="$(cat "$BBS_STATE_DIR/files/readme.txt")"
CONTENT_REF="$(cat "$BBS_CONTENT")"

echo "== 1. Sauvegarde =="
ARCHIVE="$(BBS_BACKUP_STAMP=20260624-000000 bash "$BACKUP")"
if [ -f "$ARCHIVE" ]; then ok "archive créée : $(basename "$ARCHIVE")"; else ko "pas d'archive"; fi
tar -tzf "$ARCHIVE" | grep -q 'state/users.json'      && ok "users.json archivé"   || ko "users.json manquant"
tar -tzf "$ARCHIVE" | grep -q 'state/files/readme.txt' && ok "fichier archivé"      || ko "fichier manquant"
tar -tzf "$ARCHIVE" | grep -q 'site.json'             && ok "site.json archivé"    || ko "site.json manquant"
tar -tzf "$ARCHIVE" | grep -q 'MANIFEST.txt'          && ok "manifeste présent"    || ko "manifeste manquant"

echo "== 2. Restauration après corruption =="
# On casse / efface l'état courant.
echo 'CORROMPU' > "$BBS_STATE_DIR/users.json"
rm -f "$BBS_STATE_DIR/files/readme.txt"
echo 'CORROMPU' > "$BBS_CONTENT"

bash "$RESTORE" "$ARCHIVE" >/dev/null 2>&1
[ "$(cat "$BBS_STATE_DIR/users.json" 2>/dev/null)" = "$USERS_REF" ]   && ok "users.json restauré"  || ko "users.json non restauré"
[ "$(cat "$BBS_STATE_DIR/files/readme.txt" 2>/dev/null)" = "$FILE_REF" ] && ok "fichier restauré"   || ko "fichier non restauré"
[ "$(cat "$BBS_CONTENT" 2>/dev/null)" = "$CONTENT_REF" ]              && ok "site.json restauré"   || ko "site.json non restauré"
[ -d "$BBS_STATE_DIR.pre-restore" ]                                  && ok "état précédent gardé (.pre-restore)" || ko "pas de sauvegarde .pre-restore"

echo "== 3. Restauration 'latest' =="
bash "$RESTORE" latest >/dev/null 2>&1 && ok "restore latest OK" || ko "restore latest en échec"

echo "== 4. Rotation (KEEP=3) =="
rm -f "$BBS_BACKUP_DIR"/bbsoric-backup-*.tar.gz
for i in 1 2 3 4 5; do
	BBS_BACKUP_STAMP="2026010$i-000000" BBS_BACKUP_KEEP=3 bash "$BACKUP" >/dev/null 2>&1
done
COUNT="$(ls -1 "$BBS_BACKUP_DIR"/bbsoric-backup-*.tar.gz 2>/dev/null | wc -l)"
[ "$COUNT" -eq 3 ] && ok "rotation : 3 archives gardées (sur 5)" || ko "rotation : $COUNT archives (attendu 3)"
# Les 3 plus récentes (03,04,05) doivent rester ; 01,02 purgées.
ls "$BBS_BACKUP_DIR"/bbsoric-backup-20260105-000000.tar.gz >/dev/null 2>&1 && ok "la plus récente conservée" || ko "la plus récente absente"
ls "$BBS_BACKUP_DIR"/bbsoric-backup-20260101-000000.tar.gz >/dev/null 2>&1 && ko "la plus ancienne aurait dû être purgée" || ok "la plus ancienne purgée"

echo "--------------------------------------------"
echo "Résultat : $PASS ok, $FAIL échec(s)"
[ "$FAIL" -eq 0 ] || exit 1
echo "TOUS LES TESTS PASSENT."
