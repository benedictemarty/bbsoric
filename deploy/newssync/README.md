# Synchro automatique des actualités BBS ← articles tachibana.eu

Timer systemd qui, chaque jour, importe les articles publiés de **tachibana.eu**
dans les **actualités** du BBS Oric. Tourne sur l'**hôte Proxmox `kiranerys`** (seul
à pouvoir lire la base de CT 716 *et* écrire dans CT 510 via `pct`), donc sans
dépendre du VPN ni d'un poste de dev.

Idempotent : ne pousse le `news.json` et ne redémarre `bbsoric` **que si le contenu
a changé** (comparaison sémantique) — pas de redémarrage inutile.

## Fichiers

| Fichier | Destination sur kiranerys |
|---|---|
| `../../scripts/gen-news-tachibana.py` | `/usr/local/bin/gen-news-tachibana.py` |
| `oric-news-sync.sh` | `/usr/local/bin/oric-news-sync.sh` |
| `oric-news-sync.service` | `/etc/systemd/system/` |
| `oric-news-sync.timer` | `/etc/systemd/system/` |

## Installation (depuis ce dépôt)

```bash
scp scripts/gen-news-tachibana.py deploy/newssync/oric-news-sync.sh kiranerys:/usr/local/bin/
scp deploy/newssync/oric-news-sync.{service,timer} kiranerys:/etc/systemd/system/
ssh kiranerys 'chmod +x /usr/local/bin/{gen-news-tachibana.py,oric-news-sync.sh} && \
  systemctl daemon-reload && systemctl enable --now oric-news-sync.timer'
```

## Exploitation

```bash
ssh kiranerys 'systemctl list-timers oric-news-sync.timer'   # prochaine exécution
ssh kiranerys 'systemctl start oric-news-sync.service'       # forcer une synchro
ssh kiranerys 'journalctl -u oric-news-sync.service -n 20'   # journal
```

Config (env, surchargée dans l'unité si besoin) : `CT_TACHI` (716), `CT_BBS` (510),
`NEWS_LANG` (fr). Mapping article → annonce : voir `docs/news-tachibana.md`.
