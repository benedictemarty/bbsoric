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
    "main":  { "title": "MENU PRINCIPAL", "type": "menu", "entries": [ ... ] },
    "info":  { "title": "INFOS", "type": "page", "lines": [ ... ] }
  }
}
```

- **`start`** : identifiant de la page de départ.
- **`pages`** : dictionnaire `identifiant → page`.

### Page `type: "menu"`
Liste de choix (`entries`). Chaque entrée :
```json
{ "key": "1", "label": "Informations", "target": "info" }
```
- `key` : touche (insensible à la casse).
- `target` : identifiant de page **ou** cible spéciale :
  - `__quit__` : termine la session,
  - `__back__` : page précédente (pile),
  - `__home__` : page de départ.

### Page `type: "page"`
Écran de contenu (`lines`) ; **une touche** revient en arrière (mode caractère,
cf. ADR-0002). Chaque ligne :
```json
{ "text": " Bonjour", "ink": "yellow" }
```
- `ink` (optionnel) : `black red green yellow blue magenta cyan white` (défaut blanc).

### Page `type: "applet"`
La page (texte/JSON) délègue un **comportement interactif** à un applet Go
enregistré par son nom — sans coder de page entière. Idéal pour le login, un jeu, etc.
```json
{ "type": "applet", "applet": "login", "next": "main", "lines": [ ... ] }
```
- `applet` : nom de l'applet enregistré côté serveur (ex. `login`, `register`, `guest`).
- `next` (optionnel) : page où aller **après succès** de l'applet.
- `lines` (optionnel) : texte d'intro affiché **avant** de lancer l'applet.

Un menu pointe vers une page applet comme vers n'importe quelle page
(`{ "key": "1", "label": "Se connecter", "target": "login" }`). **Ajouter un applet**
= écrire une petite fonction Go et l'enregistrer ; **le placer** = éditer ce JSON.
Applets disponibles : `login`, `register`, `guest`.

## Rendu (rappel OASCII)

Titres en jaune, règles 40 colonnes, touches en cyan, libellés en blanc, invites
en vert. Un octet d'attribut couleur occupe une case écran (cf. `oascii.md`) — éviter
les libellés trop longs pour rester dans les 40 colonnes.

## Validation

Un JSON invalide (syntaxe, `start` introuvable, cible inexistante, type inconnu)
est **refusé** : l'ancienne version reste en service et l'erreur est journalisée.
Le test `internal/content` vérifie aussi que `content/site.json` du dépôt est valide.
