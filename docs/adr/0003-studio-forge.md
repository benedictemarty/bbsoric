# ADR-0003 — Studio « Forge » : app web Go, internal/ partagé, déploiement par profils

- **Statut** : Accepté
- **Date** : 2026-06-22
- **Décideurs** : bmarty
- **Lié à** : ADR-0001/0002 (contenu piloté par JSON, type de page `applet`)

## Contexte

Le contenu du BBS (`content/site.json` : pages `menu`/`page`/`applet`, porte d'auth) était
édité **à la main**. L'utilisateur veut un **3ᵉ sous-projet** dédié — un **studio « forge »**
— pour générer/éditer le(s) site(s) **et les déployer** sur plusieurs environnements
(**prod / int / dev**) via des **profils**.

## Décision

1. **Trois sous-projets** : `server/` (serveur Go), `client/` (terminal Oric), `studio/`
   (le forge). Les paquets **partagés** `content` et `oascii` restent dans l'`internal/`
   **racine** : la règle de visibilité Go interdirait à `studio/` d'importer
   `server/internal/...`, donc on place le code réutilisé à la racine pour que le studio
   utilise **exactement** la même validation et la même palette que le serveur (zéro
   divergence). Le code propre au serveur (`bbs`, `server`, `user`) est sous `server/internal/`.

2. **Studio = app web Go** (`studio/cmd/forge`), **stdlib uniquement**, assets **embarqués**
   (`embed`). Outil de **développement local** : bind `127.0.0.1`, **sans authentification**.
   - `studio/internal/store` : liste/charge/**enregistre après validation** (`content.Parse`),
     écriture atomique, anti-traversée de chemin.
   - `studio/internal/preview` : rend une page en **HTML coloré 40 colonnes** fidèle au
     moteur (réutilise `oascii` + `content.Ink`).
   - `studio/internal/deploy` : déploiement par **profils**.

3. **Studio = source de vérité ; le déploiement ÉCRASE** la cible (abandon de la règle
   « semer une seule fois »), **après validation** et **sauvegarde horodatée**
   (`<cible>.bak.<horodatage>`). **Dry-run** par défaut dans l'UI ; **confirmation** avant
   un déploiement réel.

4. **Profils PAR SITE** : chaque site a son trio `dev`/`int`/`prod` dans
   `deploy/profiles/<site>/<env>.conf` (où `<site>` = nom du fichier sans `.json`, ex.
   `deploy/profiles/site/dev.conf`). Format `KEY=VALUE`. Un `<env>.conf.example` sert de
   **défaut** ; le `<env>.conf` réel (gitignoré) **prime**. `dev` = **local** (copie de
   fichier, le bbsd recharge à chaud) ; `int`/`prod` = **ssh/scp** (réutilise le mécanisme de
   `deploy/vps-deploy.sh`, sans dépendance). Champs : `LOCAL HOST USER PORT CONTENT_PATH
   SERVICE RELOAD` (`RELOAD` = `none|reload|restart`).

5. **Enregistrement indenté** : le studio ré-indente le JSON à l'écriture (`json.Indent`,
   2 espaces) — fichiers lisibles, diffs git stables, toutes les clés préservées (`_comment`).

## Conséquences

**Positives**
- Édition assistée + aperçu couleur, **validation identique** au serveur (réutilise `content`).
- Déploiement multi-environnements **traçable** (validation, sauvegarde, dry-run, journal).
- Zéro dépendance externe ; trois sous-projets clairs.

**Négatives / à surveiller**
- Le studio n'a **pas d'authentification** : il doit rester **local** (`127.0.0.1`).
- Le déploiement **écrase** la cible : les éditions à chaud faites directement sur un serveur
  ne sont plus la source de vérité (mais sauvegardées avant écrasement).
- L'aperçu est **fidèle mais approché** (la sémantique exacte des cases-attributs Téletexte
  pourra être affinée).

## Alternatives écartées
- **Studio Python/Flask** (comme les studios telenet) : duplique la validation hors du paquet
  Go et ajoute une dépendance Python.
- **Tout sous `server/internal/`** : empêcherait le studio de réutiliser `content`/`oascii`
  (visibilité Go) → duplication.
- **Déploiement « semer une seule fois »** : viderait de son sens un studio de déploiement.
