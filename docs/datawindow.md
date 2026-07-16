# DataWindow — grilles de données (SQLite + CRUD)

Une **DataWindow** présente une *source de données* (table SQLite typée) sous forme
de **grille paginée** navigable au clavier, avec **CRUD** complet et validation. Le
concept est porté du projet telenet ; voir `docs/adr/0004-datawindow-sqlite.md`.

## Activation

```bash
bbsd -content docs/examples/datawindow-demo.json -data /tmp/dwdata
```

- `-data <dir>` : répertoire des bases SQLite (une base `bbsoric.db`). Vide = désactivé.
- Au démarrage, chaque source déclarée dans le contenu est créée et amorcée (seed) si
  sa table est vide ; les colonnes manquantes sont ajoutées (auto-migration).

## Déclarer une source (`sources_donnees`)

Au niveau racine du `site.json` :

```json
"sources_donnees": {
  "repertoire": {
    "table": "repertoire",
    "tri_defaut": "nom ASC",
    "lignes_par_page": 15,
    "colonnes": {
      "id":    { "type": "INTEGER", "libelle": "ID",    "cle_primaire": true, "auto_increment": true },
      "nom":   { "type": "TEXT",    "libelle": "Nom",   "requis": true, "longueur_max": 16 },
      "ville": { "type": "TEXT",    "libelle": "Ville", "longueur_max": 10 },
      "note":  { "type": "INTEGER", "libelle": "Note" }
    },
    "donnees": [ { "nom": "Alice", "ville": "Lyon", "note": 5 } ]
  }
}
```

**Colonne** (`ColonneDef`) : `type` (TEXT/INTEGER/REAL/… liste blanche), `libelle`,
`cle_primaire`, `auto_increment`, `requis`, `longueur_max`, `pattern` (regex),
`valeur_defaut`, `auto_date`. `donnees` est un seed importé une seule fois.

## Source REST (API) — lecture seule

Une source peut être alimentée par un **endpoint JSON** au lieu de SQLite
(`type_source: "api"`). La grille est alors **en lecture seule** (pas de CRUD), avec
filtre/tri/pagination appliqués **côté serveur** sur les données récupérées, et un
**cache** (TTL configurable, 60 s par défaut). Idéal pour des données vivantes
(météo, actualités…).

```json
"meteo": {
  "type_source": "api",
  "tri_defaut": "ville ASC",
  "api": { "url": "https://exemple/meteo.json", "racine": "results", "ttl_sec": 300 },
  "colonnes": {
    "ville": { "type": "TEXT",    "libelle": "Ville" },
    "temp":  { "type": "INTEGER", "libelle": "Temp" }
  }
}
```

L'endpoint renvoie soit un **tableau d'objets**, soit un objet dont la clé `racine`
contient le tableau. Chaque objet mappe ses champs sur les colonnes **par nom**.
Pas de table SQLite ; le flag `-data` reste requis (le moteur porte aussi l'API).

## Présenter en grille (page `datawindow`)

```json
"grille": {
  "title": "REPERTOIRE",
  "datawindow": {
    "source": "repertoire",
    "colonnes_affichees": ["nom", "ville", "note"],
    "largeurs": [16, 10, 3],
    "couleur_entete": "yellow",
    "editable": true
  }
}
```

La page est atteinte par une entrée `{ "target": "grille" }` ou en page de départ.

![Grille DataWindow dans oric1-emu](img/datawindow-grid.png)

*Grille « repertoire » rendue dans `oric1-emu` (terminal Oric réel) : entête jaune,
6 enregistrements, la ligne sélectionnée (« Bob Durand ») en vidéo inverse, pied de
pagination et légende des touches.*

Interactions pilotées de bout en bout dans l'émulateur (saisie manuelle `127.0.0.1`
→ modem émulé → BBS local, via `scripts/test-emulateur-grille.sh`) :

| Tri par colonne (`T`) | Fiche détail (`V`) |
|---|---|
| ![Tri](img/datawindow-grid-emu-tri.png) | ![Fiche](img/datawindow-grid-emu-fiche.png) |

*À gauche : après `T`, le pied indique `tri Nom+` et les lignes sont triées par Nom.
À droite : après `V`, la carte `FICHE` affiche l'enregistrement sélectionné (le rendu
différentiel laisse apparaître le pied de la grille au-dessus).*

## Navigation (clavier Oric, sans flèches)

| Touche | Action |
|--------|--------|
| `+` / `-` | descendre / monter la sélection (déborde sur la page voisine) |
| `S` / `R` | page suivante / précédente |
| `V` | fiche détail de la ligne sélectionnée |
| `F` / `C` | poser un filtre LIKE / l'effacer |
| `T` | cycler le tri : défaut → colonne 1 ASC → DESC → colonne 2 ASC → … (libellé au pied) |
| `N` / `E` / `D` | créer / éditer / supprimer (si `editable` **et** session **admin**) |
| `X` | **télécharger** le fichier de la ligne (si `fichier_colonne` défini + `-files`) — catalogue |
| `Q` ou ESC | quitter la grille |

**Catalogue de téléchargement** (Epic J). Un descriptif DataWindow peut porter
`"fichier_colonne": "<colonne>"` : la valeur de cette colonne pour la ligne
sélectionnée est un nom de fichier de la bibliothèque (`-files`) ; la touche `X`
l'envoie via XMODEM (même chemin que l'applet `download`). La légende affiche
`X=DL` uniquement si une colonne fichier est définie et la bibliothèque active.
Contrainte : le buffer terminal Oric (~30 Ko, garde serveur 64 Ko) limite les
téléchargements aux petits fichiers (ex. `.tap`) — un PDF de magazine/livre se
**consulte** (fiche `V`) mais ne se télécharge pas vers l'Oric. Le générateur
`scripts/gen-catalogue.py` produit un catalogue (Logiciels/Magazines/Livres) depuis
la bibliothèque OricProgramsLib ; démo : `docs/examples/catalogue-demo.json`.

La ligne sélectionnée (et le bandeau titre) sont en **vidéo inverse** (bit 7 par
caractère, propre à l'Oric). Le rendu utilise le **buffer différentiel**
`oascii.Screen` : déplacer la sélection ne réémet que les deux lignes changées.

## Contraintes

- **Budget 40 colonnes** vérifié au chargement : `1 (couleur) + 3 (index) + Σ(largeur+1) ≤ 40`.
  Une grille trop large est refusée par `Site.Validate()`.
- **Sécurité** : noms de table/colonnes validés (liste blanche d'identifiants), valeurs
  toujours passées en paramètres `?` (jamais interpolées). Voir `ValiderNomSQL`/`ValiderTypeSQL`.
- **Écriture réservée aux admins** (S11.5) : la lecture est ouverte à tous (invités inclus),
  mais le CRUD (`N`/`E`/`D`) exige un compte **sysop** (`User.Admin`). Le premier compte
  enregistré devient sysop ; le flag `admin` reste éditable dans le JSON des comptes pour en
  promouvoir d'autres. Un non-admin ne voit pas les touches `N/E/D` dans la légende.
- **Sessions** : un verrou par base couvre les écritures concurrentes (1 goroutine/session).

## Édition dans le studio Forge

Tout le modèle DataWindow s'édite désormais visuellement dans le studio (`forge`),
sans toucher au JSON :

- **Onglet « Données »** — gère les `sources_donnees`. On crée/charge/supprime une
  source, on choisit son **type** (SQLite CRUD ou **API REST** lecture seule), puis :
  - *SQLite* : nom de table, **colonnes typées** (clé, type, libellé, clé primaire,
    auto-incrément, requis, longueur max, pattern, valeur par défaut, auto-date) et
    une grille de **données initiales** (seed) ;
  - *API* : `url`, clé `racine`, cache `ttl_sec` + colonnes mappées par nom ;
  - communs : tri par défaut, lignes par page.

  Le renommage d'une source ou d'une colonne reporte les références (les pages grille
  suivent) et **préserve l'ordre** des colonnes.

- **Onglet « Édition »** — sur une page, le bouton **« + grille de données »** la
  convertit en page grille. L'éditeur de descripteur règle la **source**, les
  **colonnes affichées** (ajout/retrait, **ordre** par ↑/↓, **largeur** par colonne
  avec un compteur de **budget /40** en direct), les **couleurs** (entête/lignes/
  sélection), les **lignes par écran** et le drapeau **éditable** (N/E/D).

`Valider` / `Enregistrer` passent par le même `content.Parse` que le serveur : un
contenu invalide (budget dépassé, colonne inconnue, source API sans URL…) est refusé
avant écriture.

## Pour aller plus loin (incréments suivants)

- Recherche par préfixe (en plus du filtre LIKE), masques de saisie.
- Sources API avec en-têtes/authentification, pagination côté endpoint.
