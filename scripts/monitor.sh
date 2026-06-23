#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# monitor.sh — sonde de disponibilité du BBS Oric + alerte.
#
# Vérifie successivement :
#   1. l'endpoint de supervision HTTP /healthz (si exposé localement) ;
#   2. à défaut, l'ouverture du port telnet public (6502) via /dev/tcp.
#
# En cas d'échec : journalise (stderr → journald si lancé par systemd) et, si
# une adresse d'alerte est configurée, envoie un courriel (commande `mail`).
#
# Variables d'environnement (toutes optionnelles) :
#   BBS_HEALTH_URL   URL /healthz à sonder      (défaut http://127.0.0.1:6510/healthz)
#   BBS_HOST         hôte du port telnet        (défaut 127.0.0.1)
#   BBS_PORT         port telnet                (défaut 6502)
#   BBS_ALERT_EMAIL  destinataire de l'alerte   (vide = pas de courriel)
#   BBS_TIMEOUT      timeout par sonde, secondes (défaut 5)
#
# Code de sortie : 0 = BBS up, 1 = BBS down (alerte émise).
# ---------------------------------------------------------------------------
set -uo pipefail

HEALTH_URL="${BBS_HEALTH_URL:-http://127.0.0.1:6510/healthz}"
HOST="${BBS_HOST:-127.0.0.1}"
PORT="${BBS_PORT:-6502}"
ALERT_EMAIL="${BBS_ALERT_EMAIL:-}"
TIMEOUT="${BBS_TIMEOUT:-5}"

log() { echo "[monitor] $*" >&2; }

# 1. Sonde HTTP /healthz (si curl disponible).
check_health() {
	command -v curl >/dev/null 2>&1 || return 2
	local body
	body="$(curl -fsS --max-time "$TIMEOUT" "$HEALTH_URL" 2>/dev/null)" || return 1
	[ "$(echo "$body" | tr -d '[:space:]')" = "ok" ]
}

# 2. Sonde TCP du port public (repli universel, sans dépendance).
check_tcp() {
	timeout "$TIMEOUT" bash -c "exec 3<>/dev/tcp/${HOST}/${PORT}" 2>/dev/null
}

alert() {
	local msg="$1"
	log "ALERTE : $msg"
	if [ -n "$ALERT_EMAIL" ] && command -v mail >/dev/null 2>&1; then
		printf 'BBS Oric INDISPONIBLE\n\n%s\n\nSonde : %s\nHôte : %s:%s\n' \
			"$msg" "$HEALTH_URL" "$HOST" "$PORT" \
			| mail -s "[ALERTE] BBS Oric down" "$ALERT_EMAIL"
		log "courriel envoyé à $ALERT_EMAIL"
	fi
}

if check_health; then
	log "OK (healthz)"
	exit 0
fi
rc=$?
if [ "$rc" -eq 2 ]; then
	# curl absent : on se rabat directement sur la sonde TCP.
	log "curl absent, repli TCP ${HOST}:${PORT}"
fi

if check_tcp; then
	log "OK (port TCP ${HOST}:${PORT} ouvert ; healthz indisponible)"
	exit 0
fi

alert "Ni /healthz ni le port telnet ${HOST}:${PORT} ne répondent."
exit 1
