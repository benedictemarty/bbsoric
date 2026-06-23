# Supervision & alerting — BBS Oric

> **Sprint 5.** Le BBS est exposé sur Internet 24/7 (`pavi.3617.fr:6502`). Cette
> page décrit la supervision en place au-delà de `journald` + `Restart=on-failure`.

La supervision s'organise en **couches**, de la plus locale (le démon se relève
seul) à la plus externe (une sonde alerte un humain) :

| Couche | Mécanisme | Rôle |
|--------|-----------|------|
| 1. Auto-réparation | `Restart=on-failure`, `RestartSec=5` (systemd) | le démon redémarre seul après un crash. |
| 2. Journalisation | `StandardOutput=journal` (slog) | `journalctl -u bbsoric` : connexions, refus, erreurs. |
| 3. Endpoint d'état | HTTP `/healthz` + `/metrics` (local) | vivacité + métriques exploitables. |
| 4. Sonde + alerte | `bbsoric-monitor.timer` → `monitor.sh` | teste l'état toutes les 5 min, alerte si down. |

---

## 1. Endpoint de supervision (`/healthz`, `/metrics`)

Le démon expose un petit serveur HTTP **séparé** du BBS, activé par le drapeau
`-metrics-addr`. En production il écoute **en local uniquement** :

```
bbsoric ... -metrics-addr 127.0.0.1:6510
```

> ⚠️ **Ne jamais exposer `-metrics-addr` sur `0.0.0.0`/Internet** : l'endpoint
> n'a pas d'authentification et divulgue l'état du serveur. Seul le port BBS
> (`6502`) est public.

### `GET /healthz`

Sonde de vivacité. Répond `200 ok`. Utilisable par un probe externe (timer
systemd, uptime-kuma, health check Caddy…).

```console
$ curl -s http://127.0.0.1:6510/healthz
ok
```

### `GET /metrics`

Métriques au **format texte Prometheus** (`# HELP`/`# TYPE` + valeurs) :

```console
$ curl -s http://127.0.0.1:6510/metrics
# HELP bbsoric_uptime_seconds Temps écoulé depuis le démarrage du serveur.
# TYPE bbsoric_uptime_seconds gauge
bbsoric_uptime_seconds 3600
# TYPE bbsoric_connections_total counter
bbsoric_connections_total 128
# TYPE bbsoric_connections_active gauge
bbsoric_connections_active 2
# TYPE bbsoric_connections_rejected_total counter
bbsoric_connections_rejected_total 4
```

| Métrique | Type | Sens |
|----------|------|------|
| `bbsoric_uptime_seconds` | gauge | secondes depuis le démarrage. |
| `bbsoric_connections_total` | counter | connexions TCP acceptées (cumul). |
| `bbsoric_connections_active` | gauge | sessions en cours. |
| `bbsoric_connections_rejected_total` | counter | connexions refusées par un garde-fou (limite globale / par IP). |

Un `connections_rejected_total` qui grimpe = saturation ou abus (ajuster
`-max-conns` / `-max-conns-per-ip`, ou investiguer une IP via les logs).

---

## 2. Sonde + alerte (`monitor.sh` + timer systemd)

`scripts/monitor.sh` sonde le BBS et **alerte** s'il ne répond pas :

1. teste `GET /healthz` (si `curl` présent) ;
2. à défaut, teste l'ouverture du **port telnet public** via `/dev/tcp` ;
3. en cas d'échec : journalise et envoie un **courriel** (commande `mail`) si
   `BBS_ALERT_EMAIL` est renseigné.

Code de sortie : `0` = up, `1` = down. Variables : `BBS_HEALTH_URL`, `BBS_HOST`,
`BBS_PORT`, `BBS_ALERT_EMAIL`, `BBS_TIMEOUT` (cf. en-tête du script).

### Déploiement (automatique via `make deploy`)

`deploy/vps-deploy.sh` installe :

- `scripts/monitor.sh` → `/usr/local/bin/bbsoric-monitor.sh`
- `deploy/bbsoric-monitor.service` (oneshot) → `/etc/systemd/system/`
- `deploy/bbsoric-monitor.timer` (toutes les 5 min) → `/etc/systemd/system/`

puis active le timer (`systemctl enable --now bbsoric-monitor.timer`).

Pour activer les alertes mail, décommenter `BBS_ALERT_EMAIL` dans
`bbsoric-monitor.service` (et configurer un `mail`/MTA sur l'hôte).

### Exploitation

```console
# état du timer et prochaine échéance
systemctl status bbsoric-monitor.timer
systemctl list-timers bbsoric-monitor.timer

# dernière sonde
journalctl -u bbsoric-monitor.service -n 20

# sonde manuelle
systemctl start bbsoric-monitor.service
```

---

## 3. Pistes d'évolution (non bloquantes)

- **Watchdog systemd** (`Type=notify` + `WatchdogSec`) : le démon ferait des
  `sd_notify(WATCHDOG=1)`, systemd le tuerait/relancerait s'il se fige (au-delà
  du simple crash couvert par `Restart=on-failure`).
- **Scraper Prometheus + Grafana/Alertmanager** : `/metrics` est déjà compatible ;
  il suffirait de pointer un Prometheus sur `127.0.0.1:6510/metrics` (via tunnel).
- **Probe externe tierce** (uptime-kuma, healthchecks.io) sur le port public 6502
  pour une vue « depuis Internet » indépendante de l'hôte.

---

## Références

- `server/internal/server/metrics.go` — handler `/healthz` + `/metrics`.
- `server/internal/server/server.go` — compteurs (`Stats`).
- `scripts/monitor.sh` — sonde + alerte.
- `deploy/bbsoric.service`, `deploy/bbsoric-monitor.{service,timer}` — unités systemd.
