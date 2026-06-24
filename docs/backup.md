# Sauvegarde & restauration — BBS Oric

Le BBS conserve, sur le serveur de production, un **état persistant non
reproductible** depuis le dépôt. Cette page décrit comment il est sauvegardé,
restauré, et comment vérifier que la chaîne fonctionne.

## 1. Ce qui est sauvegardé

| Donnée | Emplacement (prod) | Criticité | Reproductible ? |
| --- | --- | --- | --- |
| Comptes utilisateurs (hachés) | `/var/lib/bbsoric/users.json` | **Haute** | Non — perte = comptes perdus |
| Bibliothèque de fichiers (upload) | `/var/lib/bbsoric/files/` | Moyenne | Non |
| Contenu des pages | `/etc/bbsoric/site.json` | Moyenne | Oui (studio), mais éditable à chaud |

> `users.json` est le seul état **irrécupérable** : les mots de passe sont
> hachés (PBKDF2), donc non régénérables. C'est la cible n°1 de la sauvegarde.

Le binaire, les unités systemd et le code ne sont **pas** sauvegardés ici :
ils proviennent du dépôt Git et sont réinstallés par `deploy/vps-deploy.sh`.

## 2. Mécanisme

- **`scripts/backup.sh`** — crée une archive `tar.gz` horodatée dans
  `/var/backups/bbsoric/`, avec **rotation** (14 archives par défaut). La
  sauvegarde est **« à chaud »** : `users.json` et les fichiers sont écrits de
  façon atomique (write-temp + `rename`) par le serveur, donc l'archive ne
  capture jamais d'écriture partielle — inutile d'arrêter le BBS.
- **`deploy/bbsoric-backup.service` + `.timer`** — exécutent `backup.sh`
  **chaque jour à 03h30** (avec rattrapage `Persistent=true` si la machine
  était éteinte).
- **`scripts/restore.sh`** — restaure une archive (arrêt du service →
  restauration → redémarrage).

Structure d'une archive :

```
bbsoric-backup-AAAAMMJJ-HHMMSS/
├── state/          # copie de /var/lib/bbsoric (users.json + files/)
├── site.json       # copie de /etc/bbsoric/site.json
└── MANIFEST.txt    # horodatage, hôte, nb de comptes / fichiers
```

## 3. Déploiement

Le timer et les scripts sont installés automatiquement par
`deploy/vps-deploy.sh` (section *Sauvegardes*) :

- `scripts/backup.sh`  → `/usr/local/bin/bbsoric-backup.sh`
- `scripts/restore.sh` → `/usr/local/bin/bbsoric-restore.sh`
- `bbsoric-backup.{service,timer}` → `/etc/systemd/system/`, timer activé.

Vérifier après déploiement :

```sh
systemctl list-timers bbsoric-backup.timer     # prochaine échéance
systemctl start bbsoric-backup.service          # sauvegarde immédiate
ls -lt /var/backups/bbsoric/                    # archives présentes
```

## 4. Restauration

```sh
# Lister les sauvegardes disponibles
bbsoric-restore.sh --list

# Restaurer la plus récente (demande confirmation)
bbsoric-restore.sh latest

# Restaurer une archive précise, sans confirmation
bbsoric-restore.sh /var/backups/bbsoric/bbsoric-backup-20260624-033000.tar.gz -y
```

`restore.sh` écarte l'état courant en `*.pre-restore` (annulable) avant
d'écrire, puis redémarre le service.

### Propriété des fichiers (DynamicUser)

Le service tourne sous **`DynamicUser=yes`** : le `StateDirectory`
(`/var/lib/bbsoric`) appartient à un UID éphémère. systemd **réapproprie
récursivement** ce répertoire à l'UID courant **à chaque démarrage** ; les
fichiers restaurés par `root` redeviennent donc lisibles par le service dès le
`systemctl start` final. C'est pourquoi `restore.sh` redémarre toujours le
service en dernier — ne pas restaurer « à chaud » sans redémarrage.

## 5. Test

`scripts/test-backup.sh` valide le cycle complet dans un bac à sable
temporaire (sans systemd ni root) : sauvegarde → vérification du contenu →
restauration après corruption → `latest` → rotation. À lancer avant tout
commit touchant la sauvegarde :

```sh
bash scripts/test-backup.sh
```

## 6. Hors-site (recommandé)

Les archives vivent sur le même hôte que le service : une perte du LXC les
emporte aussi. Pour une vraie résilience, rapatrier périodiquement
`/var/backups/bbsoric/` ailleurs, par exemple depuis le poste d'admin :

```sh
rsync -avz --delete \
  "$VPS_USER@$VPS_HOST:/var/backups/bbsoric/" ~/sauvegardes/bbsoric/
```

> À planifier côté admin (cron local) — non automatisé côté serveur pour ne
> pas y stocker de secret d'accès distant.
