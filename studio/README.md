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
  Une entrée de menu « ▶ applet » se câble via une liste déroulante des applets
  connus (`login`, `register`, `guest`, `download`, `upload`, `who`, `chat`),
  avec une infobulle décrivant chacun. À garder aligné sur les applets
  enregistrés côté serveur (`bbs.Register`).
- **Aperçu « simulateur ULA »** (canvas 240×224) : rend le flux OASCII de la page comme la
  puce vidéo Oric (police Oric embarquée, attributs, inverse, double hauteur, clignotement ;
  semi-graphiques approximés). Sans ROM ni émulateur. Le rendu provient de `internal/render`
  (même flux d'octets que le serveur).
- **Valider** (refuse un JSON incohérent) et **Enregistrer** (écriture atomique).
- **Déployer** vers un environnement via un **profil** (Simuler / Déployer).

## Déploiement par profils (dev / int / prod)

Les profils sont **propres à chaque site** : `deploy/profiles/<site>/<env>.conf` où `<site>`
est le nom du fichier sans `.json`. Chaque site a son trio `dev` / `int` / `prod`.
Format `KEY=VALUE`. Un `.conf.example` sert de **défaut** ; copier en `.conf` pour l'infra
réelle (le `.conf` est gitignoré et **prime** sur l'exemple) :

```bash
# profils du site « site.json »
cp deploy/profiles/site/prod.conf.example deploy/profiles/site/prod.conf   # puis renseigner
```

Le studio (source de vérité) **valide → sauvegarde (horodatée) → écrase → reload**. Le
bouton **Simuler** (dry-run) montre les actions sans rien exécuter ; **Déployer** demande
confirmation. `dev` = **local** (copie de fichier, le bbsd recharge à chaud) ; `int`/`prod`
= **ssh/scp**. Champs : `LOCAL HOST USER PORT CONTENT_PATH SERVICE RELOAD`
(`RELOAD` = `none|reload|restart`).

API : `GET /api/profiles?site=`, `POST /api/deploy?site=&profile=&dryRun=`.

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
