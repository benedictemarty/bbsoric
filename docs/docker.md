# Docker containerization — BBS Oric

> **Sprint 5 (optional).** An alternative to systemd deployment (`deploy/`):
> run the BBS in a container. Production deployment remains on
> **systemd** (`make deploy`); Docker serves portable deployments, test
> environments, or a container-oriented host.

## Image

Multi-stage `Dockerfile`:

1. **build** (`golang:1.26-alpine`): compiles a **static** binary
   (`CGO_ENABLED=0`, `-trimpath -ldflags='-s -w'`). No external dependencies
   (stdlib only, no `go.sum`).
2. **runtime** (`alpine:3.20`): binary + default `site.json`, non-root user
   `bbsoric` (uid 10001), `wget`/`ca-certificates` for the healthcheck
   and optional TLS.

Result: **~18 MB** image.

### Healthcheck

The image embeds a `HEALTHCHECK` that queries the local `/healthz` endpoint
(`-metrics-addr 127.0.0.1:6510`, see `monitoring.md`). `docker ps` then displays
the `healthy`/`unhealthy` status.

## Quick start

```console
# build + run (docker compose)
make docker-up            # = docker compose up -d --build
docker compose logs -f    # logs
make docker-down          # stop

# or directly
make docker-build
docker run -d --name bbsoric -p 6502:6502 -v bbsoric-state:/var/lib/bbsoric bbsoric:latest
```

The BBS then listens on port **6502** of the host. Test:

```console
nc 127.0.0.1 6502
```

## Configuration

| Aspect | Detail |
|--------|--------|
| **Public port** | `6502` (telnet). Mapped via the compose `ports:`. |
| **Accounts** | persisted in the `bbsoric-state` volume (`/var/lib/bbsoric/users.json`). |
| **Content** | default `site.json` embedded in the image; overridable by mounting a file on `/etc/bbsoric/site.json` (see the commented line in the compose). |
| **Monitoring** | `/healthz` + `/metrics` **local to the container** (`127.0.0.1:6510`) — not published (not in `EXPOSE`/`ports`). |
| **Restart** | `restart: unless-stopped`. |

### Enabling TLS (port 6992)

Add `-tls-addr 0.0.0.0:6992` to the command and publish the port:

```yaml
    command: ["-addr","0.0.0.0:6502","-tls-addr","0.0.0.0:6992",
              "-content","/etc/bbsoric/site.json","-users","/var/lib/bbsoric/users.json",
              "-metrics-addr","127.0.0.1:6510"]
    ports:
      - "6502:6502"
      - "6992:6992"
```

Without `-tls-cert`/`-tls-key`, a self-signed certificate is generated at startup.

## Security

- The container runs as **non-root** (`USER bbsoric`).
- Only port **6502** (and possibly 6992) is exposed; monitoring
  stays internal.
- The server's Internet safeguards (global/per-IP connection limit,
  inactivity timeout) apply just as in native mode.

## See also

- `deploy/` + `vps-deploy.sh` — production systemd deployment.
- `docs/monitoring.md` — monitoring endpoint and probe.
- `docs/architecture.md` §5 — Internet exposure.
