# Déployer le catalogue en production

Le catalogue (Logiciels / Magazines / Livres) est un **contenu** (source DataWindow
+ pages) fusionné dans le `site.json` de production, plus des **fichiers**
téléchargeables copiés dans le répertoire `-files` du serveur.

> La prod sert **un seul** contenu (`-content /etc/bbsoric/site.json`) avec
> `-files /var/lib/bbsoric/files` et `-data /var/lib/bbsoric/dwdata` déjà configurés
> (voir `deploy/bbsoric.service`). Le catalogue est donc **greffé** dans ce site.json.

> **Source en production : tachibana.eu.** Le catalogue déployé provient de la base du site
> **tachibana.eu** (`--source tachibana`, voir la section dédiée plus bas), qui a remplacé
> OricProgramsLib. État courant : **1520 logiciels** (téléchargeables) / 701 magazines /
> 340 livres. La section « Fichiers » du BBS est désormais l'**espace personnel** par compte
> (`docs/userfiles.md`), distincte de ce catalogue public.

## Automatique (recommandé)

Prérequis : VPN mustang actif, `deploy/deploy.conf` rempli. Source **OricProgramsLib** :
`ORIC_LIB` défini (env ou `deploy.conf`). Source **tachibana** : rien de plus (snapshot
récupéré automatiquement par `pull-tachibana.sh`).

```bash
# répétition à blanc (ne pousse rien, valide le rendu) :
scripts/deploy-catalogue.sh --dry-run

# déploiement réel (rsync fichiers + site.json fusionné + restart service) :
scripts/deploy-catalogue.sh
```

Le script : récupère le `site.json` **de prod** (préserve les éditions à chaud), y
greffe le catalogue (+ entrée de menu `8` sur la page `main`), copie les fichiers
téléchargeables (petits `.tap`, ≤ 30 Ko) dans un staging, **valide** le site fusionné
(`internal/content`), puis rsync les fichiers, dépose le `site.json` et **redémarre**
le service (indispensable : une nouvelle source DataWindow est semée au démarrage).

## Manuel (étape par étape)

```bash
LIB="/media/bmarty/SP PHD U3/OricProgramsLib"
# 1. récupérer le site.json de prod (préserve les éditions à chaud)
ssh vps "cat /etc/bbsoric/site.json" > /tmp/site-prod.json
# 2. greffer le catalogue + copier les fichiers téléchargeables
python3 scripts/gen-catalogue.py --lib "$LIB" \
    --merge-into /tmp/site-prod.json --copy-files /tmp/bbsfiles --out /tmp/site.json
# 3. valider avant d'envoyer
go run ./tools/validate-content /tmp/site.json
# 4. envoyer
rsync -az /tmp/bbsfiles/ vps:/var/lib/bbsoric/files/
scp /tmp/site.json vps:/etc/bbsoric/site.json
ssh vps "systemctl restart bbsoric && systemctl is-active bbsoric"
```

## Source alternative : catalogue tachibana.eu

Le catalogue peut aussi être alimenté depuis la base du site **tachibana.eu**
(`tachibana.sqlite`, table `items`) au lieu d'`OricProgramsLib`. Le **format de
sortie et le déploiement sont identiques** — seule la SOURCE des lignes change.

Correspondance : `Logiciel` → Logiciels (téléchargeables), `Revue` → Magazines,
`Livre` + `Documentation` → Livres. Les noms de fichiers `.tap/.dsk/.rom` sont
extraits du HTML `body` des items (liens `/lib/...`), résolus contre un **miroir
local** du volume `/srv/oriclib` de tachibana.

```bash
# 1. snapshot local de la base + des seuls fichiers utiles (~20 Mo, via ssh/pct) :
scripts/pull-tachibana.sh                     # -> /tmp/tachibana/{tachibana.sqlite,oriclib/}

# 2a. déploiement automatique depuis cette source (rafraîchit le snapshot avec --pull) :
scripts/deploy-catalogue.sh --source tachibana [--pull] [--reseed]

# 2b. …ou génération manuelle (mêmes étapes 3-4 que ci-dessus) :
python3 scripts/gen-catalogue-tachibana.py \
    --db /tmp/tachibana/tachibana.sqlite --oriclib /tmp/tachibana/oriclib \
    --merge-into /tmp/site-prod.json --copy-files /tmp/bbsfiles --out /tmp/site.json
```

Config de `pull-tachibana.sh` (env) : `TACHI_SSH` (défaut `kiranerys`), `TACHI_CT`
(`716`), `TACHI_DB`, `TACHI_ORICLIB`, `OUT` (`/tmp/tachibana`), `MAX` (`65535`).
Volumétrie actuelle : **3983 logiciels** (~1520 téléchargeables ≤ 64 Ko),
**701 magazines**, **340 livres**.

## Notes

- **Redémarrage obligatoire** : le semis SQLite d'une source neuve se fait au boot
  (`InitialiserSource`). Un simple rechargement à chaud du JSON ne crée pas la table.
- **Semis idempotent** : au redémarrage suivant, si la table `catalogue` existe déjà
  et n'est pas vide, elle **n'est pas re-semée**. Pour republier un catalogue **modifié**,
  utiliser **`scripts/deploy-catalogue.sh --reseed`** : il arrête le service, **DROP** la
  table `catalogue` dans `/var/lib/bbsoric/dwdata/bbsoric.db`, puis redémarre — la table
  est recréée et re-semée depuis le nouveau `site.json`. (Sans `--reseed`, seul le contenu
  des pages est mis à jour, pas les données déjà semées.)
- **Taille** : seuls les fichiers ≤ `--max-file-size` (défaut 65535 o = 0xFFFF, borne de
  l'en-tête download 16 bits ; réception en streaming LOCI/Sedoric, cf. I2b) sont
  téléchargeables ; magazines/livres (PDF) sont consultables (fiche `V`).
- **Catalogue complet** (OricProgramsLib) : ~2600 logiciels (dont ~1900 téléchargeables),
  ~700 magazines, ~190 livres (~1,2 Mo, ~1900 fichiers). Non versionné (régénérable).
- **Source tachibana** : voir la section dédiée ci-dessus (3983 logiciels, ~1520
  téléchargeables). Choix de la source via `--source oriclib|tachibana`.
