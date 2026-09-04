# Synchro des actualités BBS ← flux RSS/Atom externe

Timer systemd qui importe un **flux RSS 2.0 / Atom** quelconque dans les
**actualités** du BBS Oric. Pendant du pont tachibana (`deploy/newssync`), mais
pour une source RSS au lieu de la base SQLite tachibana.

Tourne sur l'**hôte Proxmox `kiranerys`** (accès `pct` au CT 510 *et* réseau
sortant vers le flux). Idempotent : ne pousse le `news.json` et ne redémarre
`bbsoric` **que si le contenu a changé**.

> ⚠️ **Non installé** : aucune URL de flux n'est configurée par défaut (on
> n'invente pas de source). Renseigner `FEED_URL` avant d'installer.

## Fichiers

| Fichier | Destination sur kiranerys |
|---|---|
| binaire `rssnews` (`make rssnews`, cross-compilé linux/amd64) | `/usr/local/bin/rssnews` |
| `oric-rss-sync.sh` | `/usr/local/bin/oric-rss-sync.sh` |
| `oric-rss-sync.service` | `/etc/systemd/system/` |
| `oric-rss-sync.timer` | `/etc/systemd/system/` |

## Installation (depuis ce dépôt)

```bash
# 1. Compiler le binaire pour l'hôte (Linux/amd64) et l'outil de synchro.
GOOS=linux GOARCH=amd64 go build -o /tmp/rssnews ./server/cmd/rssnews
scp /tmp/rssnews deploy/rssnews/oric-rss-sync.sh kiranerys:/usr/local/bin/
scp deploy/rssnews/oric-rss-sync.{service,timer} kiranerys:/etc/systemd/system/

# 2. Renseigner l'URL du flux dans l'unité (FEED_URL=…), puis activer.
ssh kiranerys 'chmod +x /usr/local/bin/{rssnews,oric-rss-sync.sh} && \
  systemctl daemon-reload && systemctl enable --now oric-rss-sync.timer'
```

## Test hors ligne (sans installer)

```bash
make rssnews
./rssnews -file un-flux-telecharge.xml -author exemple.org -out -   # aperçu JSON
./rssnews -feed https://exemple.org/rss.xml -author exemple.org \
          --merge-into news.json --out news.json                    # fusion réelle
```

## Exploitation

```bash
ssh kiranerys 'systemctl list-timers oric-rss-sync.timer'   # prochaine exécution
ssh kiranerys 'systemctl start oric-rss-sync.service'       # forcer une synchro
ssh kiranerys 'journalctl -u oric-rss-sync.service -n 20'   # journal
```

Config (env, surchargée dans l'unité) : `FEED_URL` (obligatoire), `FEED_AUTHOR`,
`CT_BBS` (510), `MAX_ITEMS` (20). Mapping flux → annonce : voir `docs/rss.md`.
