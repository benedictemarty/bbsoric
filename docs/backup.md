# Backup & restore — BBS Oric

The BBS keeps, on the production server, a **persistent state that cannot be
reproduced** from the repository. This page describes how it is backed up,
restored, and how to verify that the chain works.

## 1. What is backed up

| Data | Location (prod) | Criticality | Reproducible? |
| --- | --- | --- | --- |
| User accounts (hashed) | `/var/lib/bbsoric/users.json` | **High** | No — loss = accounts lost |
| File library (uploads) | `/var/lib/bbsoric/files/` | Medium | No |
| Page content | `/etc/bbsoric/site.json` | Medium | Yes (studio), but editable live |

> `users.json` is the only **unrecoverable** state: passwords are
> hashed (PBKDF2), hence not regenerable. It is the #1 backup target.

The binary, the systemd units and the code are **not** backed up here:
they come from the Git repository and are reinstalled by `deploy/vps-deploy.sh`.

## 2. Mechanism

- **`scripts/backup.sh`** — creates a timestamped `tar.gz` archive in
  `/var/backups/bbsoric/`, with **rotation** (14 archives by default). The
  backup is **"hot"**: `users.json` and the files are written
  atomically (write-temp + `rename`) by the server, so the archive never
  captures a partial write — no need to stop the BBS.
- **`deploy/bbsoric-backup.service` + `.timer`** — run `backup.sh`
  **every day at 03:30** (with `Persistent=true` catch-up if the machine
  was off).
- **`scripts/restore.sh`** — restores an archive (stop the service →
  restore → restart).

Structure of an archive:

```
bbsoric-backup-AAAAMMJJ-HHMMSS/
├── state/          # copy of /var/lib/bbsoric (users.json + files/)
├── site.json       # copy of /etc/bbsoric/site.json
└── MANIFEST.txt    # timestamp, host, number of accounts / files
```

## 3. Deployment

The timer and the scripts are installed automatically by
`deploy/vps-deploy.sh` (*Backups* section):

- `scripts/backup.sh`  → `/usr/local/bin/bbsoric-backup.sh`
- `scripts/restore.sh` → `/usr/local/bin/bbsoric-restore.sh`
- `bbsoric-backup.{service,timer}` → `/etc/systemd/system/`, timer enabled.

Verify after deployment:

```sh
systemctl list-timers bbsoric-backup.timer     # next due time
systemctl start bbsoric-backup.service          # immediate backup
ls -lt /var/backups/bbsoric/                    # archives present
```

## 4. Restore

```sh
# List available backups
bbsoric-restore.sh --list

# Restore the most recent (asks for confirmation)
bbsoric-restore.sh latest

# Restore a specific archive, without confirmation
bbsoric-restore.sh /var/backups/bbsoric/bbsoric-backup-20260624-033000.tar.gz -y
```

`restore.sh` sets the current state aside as `*.pre-restore` (undoable) before
writing, then restarts the service.

### File ownership (DynamicUser)

The service runs under **`DynamicUser=yes`**: the `StateDirectory`
(`/var/lib/bbsoric`) belongs to an ephemeral UID. systemd **recursively
reassigns** this directory to the current UID **on every start**; files
restored by `root` therefore become readable again by the service as of the
final `systemctl start`. This is why `restore.sh` always restarts the
service last — do not restore "hot" without a restart.

## 5. Test

`scripts/test-backup.sh` validates the full cycle in a temporary
sandbox (without systemd or root): backup → content verification →
restore after corruption → `latest` → rotation. Run it before any
commit touching the backup:

```sh
bash scripts/test-backup.sh
```

## 6. Off-site (recommended)

The archives live on the same host as the service: a loss of the LXC
takes them too. For real resilience, periodically pull
`/var/backups/bbsoric/` elsewhere, for example from the admin workstation:

```sh
rsync -avz --delete \
  "$VPS_USER@$VPS_HOST:/var/backups/bbsoric/" ~/sauvegardes/bbsoric/
```

> To schedule on the admin side (local cron) — not automated on the server
> side so as not to store a remote-access secret there.
