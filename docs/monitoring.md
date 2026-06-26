# Monitoring & alerting — BBS Oric

> **Sprint 5.** The BBS is exposed on the Internet 24/7 (`pavi.3617.fr:6502`). This
> page describes the monitoring in place beyond `journald` + `Restart=on-failure`.

Monitoring is organized in **layers**, from the most local (the daemon recovers
on its own) to the most external (a probe alerts a human):

| Layer | Mechanism | Role |
|--------|-----------|------|
| 1. Self-healing | `Restart=on-failure`, `RestartSec=5` (systemd) | the daemon restarts on its own after a crash. |
| 2. Logging | `StandardOutput=journal` (slog) | `journalctl -u bbsoric`: connections, rejections, errors. |
| 3. Status endpoint | HTTP `/healthz` + `/metrics` (local) | liveness + usable metrics. |
| 4. Probe + alert | `bbsoric-monitor.timer` → `monitor.sh` | tests the status every 5 min, alerts if down. |

---

## 1. Monitoring endpoint (`/healthz`, `/metrics`)

The daemon exposes a small HTTP server **separate** from the BBS, enabled by the
`-metrics-addr` flag. In production it listens **locally only**:

```
bbsoric ... -metrics-addr 127.0.0.1:6510
```

> ⚠️ **Never expose `-metrics-addr` on `0.0.0.0`/Internet**: the endpoint
> has no authentication and discloses the server state. Only the BBS port
> (`6502`) is public.

### `GET /healthz`

Liveness probe. Responds `200 ok`. Usable by an external probe (systemd
timer, uptime-kuma, Caddy health check…).

```console
$ curl -s http://127.0.0.1:6510/healthz
ok
```

### `GET /metrics`

Metrics in **Prometheus text format** (`# HELP`/`# TYPE` + values):

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

| Metric | Type | Meaning |
|----------|------|------|
| `bbsoric_uptime_seconds` | gauge | seconds since startup. |
| `bbsoric_connections_total` | counter | accepted TCP connections (cumulative). |
| `bbsoric_connections_active` | gauge | sessions in progress. |
| `bbsoric_connections_rejected_total` | counter | connections rejected by a safeguard (global / per-IP limit). |

A rising `connections_rejected_total` = saturation or abuse (adjust
`-max-conns` / `-max-conns-per-ip`, or investigate an IP via the logs).

---

## 2. Probe + alert (`monitor.sh` + systemd timer)

`scripts/monitor.sh` probes the BBS and **alerts** if it does not respond:

1. tests `GET /healthz` (if `curl` is present);
2. otherwise, tests opening the **public telnet port** via `/dev/tcp`;
3. on failure: logs and sends an **email** (`mail` command) if
   `BBS_ALERT_EMAIL` is set.

Exit code: `0` = up, `1` = down. Variables: `BBS_HEALTH_URL`, `BBS_HOST`,
`BBS_PORT`, `BBS_ALERT_EMAIL`, `BBS_TIMEOUT` (see the script header).

### Deployment (automatic via `make deploy`)

`deploy/vps-deploy.sh` installs:

- `scripts/monitor.sh` → `/usr/local/bin/bbsoric-monitor.sh`
- `deploy/bbsoric-monitor.service` (oneshot) → `/etc/systemd/system/`
- `deploy/bbsoric-monitor.timer` (every 5 min) → `/etc/systemd/system/`

then enables the timer (`systemctl enable --now bbsoric-monitor.timer`).

To enable mail alerts, uncomment `BBS_ALERT_EMAIL` in
`bbsoric-monitor.service` (and configure a `mail`/MTA on the host).

### Operation

```console
# timer status and next run
systemctl status bbsoric-monitor.timer
systemctl list-timers bbsoric-monitor.timer

# last probe
journalctl -u bbsoric-monitor.service -n 20

# manual probe
systemctl start bbsoric-monitor.service
```

---

## 3. Possible evolutions (non-blocking)

- **systemd watchdog** (`Type=notify` + `WatchdogSec`): the daemon would issue
  `sd_notify(WATCHDOG=1)`, and systemd would kill/restart it if it froze (beyond
  the simple crash covered by `Restart=on-failure`).
- **Prometheus scraper + Grafana/Alertmanager**: `/metrics` is already compatible;
  it would suffice to point a Prometheus at `127.0.0.1:6510/metrics` (via tunnel).
- **Third-party external probe** (uptime-kuma, healthchecks.io) on the public port 6502
  for a "from the Internet" view independent of the host.

---

## References

- `server/internal/server/metrics.go` — `/healthz` + `/metrics` handler.
- `server/internal/server/server.go` — counters (`Stats`).
- `scripts/monitor.sh` — probe + alert.
- `deploy/bbsoric.service`, `deploy/bbsoric-monitor.{service,timer}` — systemd units.
