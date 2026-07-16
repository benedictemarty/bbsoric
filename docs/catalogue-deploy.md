# Déployer le catalogue en production

Le catalogue (Logiciels / Magazines / Livres) est un **contenu** (source DataWindow
+ pages) fusionné dans le `site.json` de production, plus des **fichiers**
téléchargeables copiés dans le répertoire `-files` du serveur.

> La prod sert **un seul** contenu (`-content /etc/bbsoric/site.json`) avec
> `-files /var/lib/bbsoric/files` et `-data /var/lib/bbsoric/dwdata` déjà configurés
> (voir `deploy/bbsoric.service`). Le catalogue est donc **greffé** dans ce site.json.

## Automatique (recommandé)

Prérequis : VPN mustang actif, `deploy/deploy.conf` rempli, `ORIC_LIB` défini
(chemin de `OricProgramsLib`, dans l'environnement ou `deploy.conf`).

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

## Notes

- **Redémarrage obligatoire** : le semis SQLite d'une source neuve se fait au boot
  (`InitialiserSource`). Un simple rechargement à chaud du JSON ne crée pas la table.
- **Semis idempotent** : au redémarrage suivant, si la table `catalogue` existe déjà
  et n'est pas vide, elle **n'est pas re-semée**. Pour republier un catalogue **modifié**,
  utiliser **`scripts/deploy-catalogue.sh --reseed`** : il arrête le service, **DROP** la
  table `catalogue` dans `/var/lib/bbsoric/dwdata/bbsoric.db`, puis redémarre — la table
  est recréée et re-semée depuis le nouveau `site.json`. (Sans `--reseed`, seul le contenu
  des pages est mis à jour, pas les données déjà semées.)
- **Taille** : seuls les fichiers ≤ `--max-file-size` (défaut 30720 o = buffer terminal
  Oric) sont téléchargeables ; magazines/livres (PDF) sont consultables (fiche `V`).
- **Catalogue complet** : ~2600 logiciels (dont ~1900 téléchargeables), ~700 magazines,
  ~190 livres (~1,2 Mo, ~1900 fichiers). Non versionné (régénérable).
