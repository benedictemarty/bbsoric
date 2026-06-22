# Studio Forge — éditeur de contenu du BBS Oric

Outil web **local** pour éditer le(s) `site*.json` du BBS (pages `menu` / `page` /
`applet`, porte d'auth), avec **aperçu OASCII couleur** et **validation par le même paquet
que le serveur** (`internal/content`) — donc zéro divergence de format. Plus tard :
déploiement par profils (dev / int / prod).

## Lancer

```bash
make studio                  # go run ./studio/cmd/forge -addr 127.0.0.1:8080
# ou
go run ./studio/cmd/forge -addr 127.0.0.1:8080 -content-dir content
```

Puis ouvrir <http://127.0.0.1:8080>. Outil de **développement** : il écoute sur `127.0.0.1`
uniquement (non exposé, pas d'authentification).

## Fonctions

- Charger un site, lister/ajouter/renommer/supprimer des pages.
- Éditer par formulaire selon le type : `menu` (entrées touche/libellé/cible),
  `page` (lignes texte + encre), `applet` (nom d'applet + page `next` + intro).
- Aperçu couleur 40 colonnes fidèle au rendu serveur.
- **Valider** (refuse un JSON incohérent) et **Enregistrer** (écriture atomique).

## Architecture

```
studio/
  cmd/forge/         serveur web (net/http, assets embarqués) + handlers API
  internal/store/    liste / charge / enregistre les site*.json (valide avant écriture)
  internal/preview/  rend une page en HTML coloré (réutilise oascii + content.Ink)
  web/               index.html, app.js, style.css (embed)
```

API : `GET /api/sites`, `GET /api/site?name=`, `POST /api/validate`,
`POST /api/save?name=`, `POST /api/preview?page=`.

Stdlib uniquement, aucune dépendance externe.
