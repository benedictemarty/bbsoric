# Contenu dynamique — flux de pages en JSON

L'enchaînement des écrans du BBS (menus, pages de contenu, navigation) est
**piloté par un fichier JSON rechargé à chaud** : le modifier met à jour le BBS
sans recompiler ni redémarrer (prise en compte sous ~2 s, à la navigation suivante
des sessions en cours).

- Fichier de référence versionné : [`../content/site.json`](../content/site.json)
- En production : `/etc/bbsoric/site.json` (édité directement sur le serveur ;
  le déploiement ne l'écrase jamais — il ne le sème qu'à l'initialisation).
- Lancement : `bbsd -content /etc/bbsoric/site.json` (sans `-content`, contenu
  intégré par défaut).

## Format

```json
{
  "start": "main",
  "pages": {
    "main": { "title": "MENU PRINCIPAL", "entries": [ ... ] },
    "info": { "title": "INFOS", "lines": [ ... ] }
  }
}
```

- **`start`** : identifiant de la page de départ.
- **`pages`** : dictionnaire `identifiant → page`.

### La page (type unique)
Une page a un **titre** et, **optionnellement**, du **texte** (`lines`) et/ou des
**choix** (`entries`) :
- **avec `entries`** → écran **interactif** : une touche route vers l'entrée choisie ;
  le texte (`lines`) éventuel s'affiche **au-dessus** des choix ;
- **sans `entries`** → écran de **contenu** : une touche revient en arrière
  (mode caractère, cf. ADR-0002).

**Lignes de texte** (`lines`) — attributs Oric par ligne :
```json
{ "text": " ALERTE ", "ink": "white", "paper": "red", "blink": true, "doubleHeight": false }
```
- `ink` (optionnel) : couleur du texte — `black red green yellow blue magenta cyan white` (défaut blanc).
- `paper` (optionnel) : couleur de **fond** (mêmes noms ; défaut noir si absent).
- `blink` (optionnel) : **clignotement**.
- `doubleHeight` (optionnel) : **double hauteur**.

> Rappel Oric : chaque attribut **occupe une case écran** (un changement de couleur « mange »
> une colonne) et l'ULA réinitialise encre/fond à chaque début de ligne — d'où l'application
> des attributs **par ligne**. Pour des effets plus poussés (couleurs multiples sur une ligne,
> ASCII-art, animation, interaction), écrire un **applet** (cf. `studio/README.md` / ADR-0001).

**Choix** (`entries`) — une entrée **navigue** (`target`) **ou lance un applet** (`applet`
+ `next`). Un menu peut donc proposer plusieurs applets au choix.

Entrée de navigation :
```json
{ "key": "1", "label": "Informations", "target": "info" }
```
- `key` : touche (insensible à la casse).
- `target` : identifiant de page **ou** cible spéciale :
  - `__quit__` : termine la session,
  - `__back__` : page précédente (pile),
  - `__home__` : page de départ.

Entrée-applet :
```json
{ "key": "1", "label": "Se connecter", "applet": "login", "next": "main" }
```
- `applet` : nom de l'applet à lancer quand l'entrée est choisie (`login`, `register`,
  `guest`…). **Ajouter un applet** = écrire une petite fonction Go et l'enregistrer.
- `next` (optionnel) : page où aller **après succès** de l'applet (vide = on reste).

> Compat : une page peut aussi porter `applet` (+ `next`) au niveau **page** (applet
> auto-lancé à l'arrivée). Mécanisme historique conservé pour les JSON écrits à la main ;
> préférez une **entrée-applet**.

## Rendu (rappel OASCII)

Titres en jaune, règles 40 colonnes, touches en cyan, libellés en blanc, invites
en vert. Un octet d'attribut couleur occupe une case écran (cf. `oascii.md`) — éviter
les libellés trop longs pour rester dans les 40 colonnes.

## Validation

Un JSON invalide (syntaxe, `start` introuvable, cible inexistante, type inconnu)
est **refusé** : l'ancienne version reste en service et l'erreur est journalisée.
Le test `internal/content` vérifie aussi que `content/site.json` du dépôt est valide.
