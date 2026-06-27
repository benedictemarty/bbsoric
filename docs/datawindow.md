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

## Navigation (clavier Oric, sans flèches)

| Touche | Action |
|--------|--------|
| `+` / `-` | descendre / monter la sélection (déborde sur la page voisine) |
| `S` / `R` | page suivante / précédente |
| `V` | fiche détail de la ligne sélectionnée |
| `F` / `C` | poser un filtre LIKE / l'effacer |
| `T` | cycler le tri : défaut → colonne 1 ASC → DESC → colonne 2 ASC → … (libellé au pied) |
| `N` / `E` / `D` | créer / éditer / supprimer (si `editable` **et** session identifiée) |
| `Q` ou ESC | quitter la grille |

La ligne sélectionnée (et le bandeau titre) sont en **vidéo inverse** (bit 7 par
caractère, propre à l'Oric). Le rendu utilise le **buffer différentiel**
`oascii.Screen` : déplacer la sélection ne réémet que les deux lignes changées.

## Contraintes

- **Budget 40 colonnes** vérifié au chargement : `1 (couleur) + 3 (index) + Σ(largeur+1) ≤ 40`.
  Une grille trop large est refusée par `Site.Validate()`.
- **Sécurité** : noms de table/colonnes validés (liste blanche d'identifiants), valeurs
  toujours passées en paramètres `?` (jamais interpolées). Voir `ValiderNomSQL`/`ValiderTypeSQL`.
- **Sessions** : un verrou par base couvre les écritures concurrentes (1 goroutine/session).

## Pour aller plus loin (incréments suivants)

- Édition des sources/données depuis le studio Forge.
- Tri interactif par colonne, recherche par préfixe, sources API REST (comme telenet).
