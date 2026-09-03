# Espace fichiers personnels (« Mes fichiers »)

Chaque **compte identifié** dispose d'un **espace de fichiers privé**, distinct de la
bibliothèque publique du **Catalogue**. C'est la section « Fichiers » du BBS. Décision
d'architecture : [`docs/adr/0006-personal-file-space.md`](adr/0006-personal-file-space.md).

> **Public vs privé.** Le **Catalogue** (`datawindow`, cf. `docs/datawindow.md`) sert la
> bibliothèque **publique** en téléchargement seul. « Mes fichiers » (`mesfichiers`) est
> l'espace **privé** de l'utilisateur : il y **téléverse** et **télécharge** ses propres
> fichiers. Les deux sont indépendants (répertoires séparés).

## Activation

```bash
bbsd ... -userfiles /var/lib/bbsoric/userfiles \
         -userfiles-max-files 20 -userfiles-max-bytes 524288 -max-upload 65536
```

- `-userfiles <dir>` : répertoire **racine** des espaces personnels (créé si absent).
  Vide = fonctionnalité **désactivée** (l'applet répond « Espace personnel indisponible »).
- `-userfiles-max-files <n>` : **quota** en nombre de fichiers par utilisateur (défaut 20 ;
  0 = illimité).
- `-userfiles-max-bytes <n>` : **quota** en octets au total par utilisateur (défaut 512 Ko ;
  0 = illimité).
- `-max-upload <n>` : borne la taille **d'un** fichier (partagée avec le Catalogue, défaut 64 Ko).

Un dépassement de quota est **refusé proprement** (message clair), sans écriture partielle.

## Modèle de stockage

- Un **sous-répertoire par compte** : `<dir>/<pseudo-normalisé>/`. Le pseudo est déjà validé à
  l'inscription (`[A-Za-z0-9_-]`, 2–16) et **re-validé** (`[a-z0-9_-]`) côté store — défense en
  profondeur contre toute sortie du répertoire (path traversal).
- Réutilise `internal/files.Library` par utilisateur (validation des noms, écriture atomique,
  lecture bornée). Package : `server/internal/userfiles`.
- Persistance = système de fichiers (comme `-files`). **Sauvegarde** = copie du répertoire
  `-userfiles`.

## Câbler l'applet dans le contenu

Un seul applet : **`mesfichiers`**. Il se pose comme entrée de menu :

```json
"fichiers": { "title": "MES FICHIERS", "entries": [
  { "key": "1", "label": "Mon espace personnel", "applet": "mesfichiers", "next": "fichiers" },
  { "key": "R", "label": "Retour", "target": "__back__" }
]}
```

L'applet **liste** les fichiers du compte (avec compteur d'usage). Touches :
- `1`–`9` : **télécharger** (XMODEM, même chemin que le Catalogue) ;
- `T` : **téléverser** (XMODEM → écriture avec quota) ;
- `E` : **effacer** un fichier (sélection par numéro + confirmation `O/N`) ;
- `R` : **renommer** un fichier (numéro + nouveau nom ; refus si la cible existe) ;
- `Q` : revenir.

## Copie depuis le Catalogue (touche `M`)

Depuis une grille **Catalogue** (`datawindow` avec `fichier_colonne`), un membre identifié copie le
fichier de la ligne sélectionnée dans son espace personnel avec la touche **`M`** (« Mon espace ») :
le fichier public (`-files`) est lu puis écrit dans `<userfiles>/<pseudo>/` (quota appliqué, refus
propre si dépassement). La touche n'apparaît (légende « M=perso ») que pour un membre avec `-userfiles`.

## Règles d'accès

- **Réservé aux comptes identifiés** : un **invité** est refusé (« Reserve aux membres
  identifies »). La garde est `State.User != nil` **et** `UserFiles != nil`.
- Un utilisateur ne voit **que** son propre répertoire (aucune énumération des autres).

## Transfert

Download et upload utilisent le **protocole XMODEM** décrit dans `docs/transfer.md`
(en-tête de download 16 bits, réception en streaming LOCI/Sedoric). Côté serveur, le download
réutilise `sendFileDownload` ; l'upload passe par `Store.Write` (quota appliqué).

## Déploiement

Le service de production active `-userfiles /var/lib/bbsoric/userfiles` avec les quotas ci-dessus
(cf. `deploy/bbsoric.service`). La page `fichiers` du contenu de prod lance `mesfichiers` (l'ancien
`download`/`upload` public à plat a été retiré, superseded par le Catalogue). Vérifié en prod :
inscription → téléversement (fichier privé sur disque) → téléchargement **byte-exact**.

## Tests

- `server/internal/userfiles` : isolation entre comptes, pseudo insensible à la casse, quotas
  (fichiers / octets, remplacement recompté), rejet d'un pseudo hors périmètre.
- `server/internal/bbs` : `TestMesFichiersRefuseNonIdentifie` (garde invité).
