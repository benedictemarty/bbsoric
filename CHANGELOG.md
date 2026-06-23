# CHANGELOG — BBS Oric

Toutes les modifications notables de ce projet sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) ;
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]

### Modifié (Contenu — page de login par défaut en page « form »)
- `content/site.json` : l'accueil ne lance plus l'applet `login` directement ;
  l'entrée 1 cible une **page `login` dédiée de type `form`** (action `login`,
  champs pseudo/mot de passe, `next: main`). Démontre le modèle déclaratif sur le
  contenu de production. Routage validé end-to-end (accueil → CONNEXION → saisie).

### Ajouté (Pages de saisie déclaratives — applet « form »)
- **Modèle** (`internal/content`) : type **`Form`** (`action`, `fields`, `next`) +
  **`Field`** (`key`, `label`, `secret`) sur la page. `Validate` vérifie l'action
  (`login`/`register`), les champs requis (`login`+`password`, plus `confirm` pour
  l'inscription) et l'existence de `next`.
- **Moteur** (`server/internal/bbs`) : applet générique **`form`** (`form.go`) —
  affiche le décor (buffer raw composé OU bandeau de titre), saisit les champs
  déclarés, puis exécute l'action côté serveur (authentification / création de
  compte, hachage PBKDF2 inchangé). `runFormPage` route vers `Form.Next`. Un seul
  applet déclaratif remplace l'écriture de Go par écran de saisie ; `login`/
  `register`/`guest` historiques restent (compat). Tests `TestFormPageLogin` /
  `TestFormPageRegister`.
- **Studio** : éditeur de **formulaire** dans l'onglet « Édition » (`formEditor`) —
  action, liste de champs (clé/libellé/secret), `next` ; ajout auto du champ
  `confirm` en mode inscription. Une page à formulaire n'affiche pas d'éditeur de
  menu (le form pilote la page).

### Modifié (Studio — navigation raw : colonne « libellé » masquée)
- **Onglet « Écran », bloc Navigation** : la colonne **« Libellé »** est retirée.
  Sur un « menu sur fond d'écran » (page raw), le libellé est **dessiné dans le
  décor** et `e.label` est ignoré au rendu (`RawScreen`) — seul compte le mapping
  touche → cible/applet. `entriesEditor` reçoit une option `hideLabel` (la colonne
  reste affichée dans l'onglet « Édition » pour les menus normaux).

### Modifié (Studio — dépose de glyphe : charset alternatif auto)
- **Onglet « Écran »** : cliquer un glyphe BBS le **dépose** désormais directement
  au curseur (au lieu de seulement charger le pinceau) et **pose l'attribut
  charset alternatif (0x09) s'il n'est pas déjà actif** à cette position — un
  glyphe n'est rendu en police BBS que si l'alt est actif. `altActiveAt` calcule
  l'état par sérialisation depuis le début de ligne ; `dropGlyph` n'ajoute la case
  d'attribut que si nécessaire (pas de doublon si l'alt est déjà posé).

### Modifié (Studio — palette de glyphes à droite de l'écran)
- **Onglet « Écran »** : la palette de glyphes BBS passe **sous** le canvas à
  **droite** de celui-ci (conteneur flex `.screen-edit` : canvas + `.palette-side`).
  Plus de scroll vertical pour atteindre les glyphes ; layout responsive (la
  palette repasse dessous sur écran étroit).

### Ajouté (Menu sur fond d'écran — raw + entries combinés)
- **Moteur** (`server/internal/bbs/engine.go`) : une page « écran brut » (`raw`)
  se **combine** désormais avec des `entries`. Le buffer 40×28 sert de **fond
  d'écran** (décor composé case par case, libellés du menu dessinés dedans) et
  les `entries` assurent la **navigation** (touche → page, ou ▶ applet) — sans
  invite « Votre choix » ajoutée. Présentation (Screen) et logique (Entries)
  séparées. Le `switch` traite les entries en priorité, le rendu suit `raw`.
  Test `TestRawScreenMenu` (fond raw + navigation, sans invite).
- **Modèle** (`internal/content`) : documentation du mix `Raw`+`Entries` ;
  `Validate` l'acceptait déjà.
- **Studio, onglet « Écran »** : éditeur de **navigation** sous la grille
  (`renderScreenNav` + `#screen-nav`) — on compose le décor au-dessus et on câble
  les touches juste en dessous. `entriesEditor` rendu réutilisable (callback de
  rafraîchissement). L'enregistrement préserve les entrées (statut « raw + menu »).

### Corrigé (Studio — éditeur d'écran : sélecteur de page vide)
- **Onglet « Écran »** : le sélecteur de page restait **vide** quand le site ne
  contenait aucune page « écran brut » (`raw`), alors que `screenLoad` sait déjà
  charger **n'importe quelle** page (rendu serveur → buffer éditable).
  `refreshScreenPages` liste désormais **toutes** les pages (suffixe « (écran) »
  pour celles déjà en mode raw). Charger une page normale puis l'enregistrer la
  convertit en écran brut.

### Corrigé (Studio — éditeur d'écran : pose des attributs couleur)
- **Onglet « Écran »** : cliquer une pastille **encre/fond** (ou un bouton
  d'attribut texte alt/cli/norm) ne faisait que changer le pinceau **sans rien
  poser** — la couleur semblait « ne pas s'appliquer » et le clic **volait le
  focus** au canvas (frappe clavier inopérante ensuite). Désormais le clic
  **pose la case d'attribut** à la position du curseur (un attribut OCCUPE une
  case sur Oric : l'« espace » coloré), **avance le curseur**, et **rend le
  focus** au canvas pour enchaîner la frappe (`pickAttr`/`putByteAdvance` dans
  `studio/web/app.js`). Le pinceau reste réglé sur la valeur choisie (peinture
  au clic toujours possible).

### Ajouté (Sprint 5 — Conteneurisation Docker)
- **`Dockerfile`** multi-stage : build `golang:1.26-alpine` (binaire statique
  `CGO_ENABLED=0`, `-trimpath -ldflags='-s -w'`) → runtime `alpine:3.20`
  **non-root** (uid 10001), `site.json` par défaut intégré. Image **~18 Mo**,
  **`HEALTHCHECK`** sur `/healthz`. Aucune dépendance externe (stdlib only).
- **`docker-compose.yml`** : service `bbsoric` (port 6502, `restart:
  unless-stopped`, volume `bbsoric-state` pour les comptes, montage `site.json`
  optionnel). **`.dockerignore`** (contexte de build minimal).
- **Makefile** : cibles `docker-build`, `docker-up`, `docker-down`.
- **Doc** : `docs/docker.md` (image, démarrage, config, TLS, sécurité).
- Validé : `docker build` OK, conteneur démarré, BBS répond sur 6502 (bannière
  ASCII-art), healthcheck `ok`. Sprint 5 **terminé** (prod reste sur systemd).

### Ajouté (Sprint 5 — Monitoring/alerting + doc utilisateur)
- **Endpoint de supervision HTTP** (`server/internal/server/metrics.go`) :
  `GET /healthz` (sonde de vivacité « ok ») et `GET /metrics` (format texte
  Prometheus). Activé par le drapeau **`-metrics-addr`** (vide = désactivé ;
  **local-only** en prod, ex. `127.0.0.1:6510` — jamais exposé sur Internet).
  Arrêt propre sur SIGINT/SIGTERM.
- **Métriques** (`server/internal/server/server.go`) : compteurs atomiques et
  `Server.Stats()` — `bbsoric_uptime_seconds`, `bbsoric_connections_total`,
  `bbsoric_connections_active`, `bbsoric_connections_rejected_total`. Tests
  `TestHealthz` / `TestMetricsReflectsCounters`.
- **Sonde + alerting** : `scripts/monitor.sh` (teste `/healthz` puis le port
  telnet via `/dev/tcp`, alerte par courriel si down) déclenchée par
  `deploy/bbsoric-monitor.timer` → `deploy/bbsoric-monitor.service` (oneshot,
  toutes les 5 min). `bbsoric.service` ajoute `-metrics-addr 127.0.0.1:6510` ;
  `vps-deploy.sh` installe et active la supervision automatiquement.
- **Docs** : `docs/monitoring.md` (couches de supervision, endpoint, sonde,
  pistes watchdog/Prometheus) et `docs/guide-utilisateur.md` (connexion grand
  public depuis un Oric réel et depuis un PC, navigation, comptes, dépannage).

### Ajouté (Sprint 4 — Connexion matérielle réelle)
- **`docs/connexion-materielle.md`** : guide complet pour joindre le BBS depuis un
  **Oric réel**. Chaîne matérielle Oric→ACIA→modem WiFi→TCP ; adressage **ACIA
  `$031C`** (standard) et **LOCI `$03A0-$03BF`** avec table des registres 6551 ;
  modem WiFi (firmware Hayes / picowifi v0.2.0), réglages **9600 8N1** ; commandes
  AT émises par le client (`ATD`, `ATDT#` TLS, `AT$CA`/`AT$CV1`, `ATGET`) ;
  procédure pas à pas (`CLOAD"TERM"` → menu → numérotation), tableau de dépannage.
- **Procédure de recette matérielle** (checklist **T1–T9**, §7 du même doc) :
  chargement, backend ACIA, répertoire/CONNECT, bannière couleur, navigation
  clavier, saisie manuelle, TLS, déconnexion, stabilité. *Test physique en attente
  de matériel* — le pipeline est validé dans l'émulateur.
- **Écran d'accueil ASCII-art « ORIC »** (`server/internal/bbs/welcome.go`) :
  bannière enrichie d'un art 5 lignes assemblé par glyphes 5×5 (`buildOricArt`),
  centré et **conforme OASCII** (largeur ≤ 40 colonnes, 1 octet d'attribut/ligne),
  couleurs jaune (art) / cyan (sous-titre `B B S   O R I C`). Version applicative
  passée à « Sprint 4 ». Tests serveur verts.

### Ajouté (Éditeur d'écran plein 40×28 + page « écran brut »)
- **Page « écran brut »** : champ `raw` + buffer **`screen`** (40×28 octets, base64) dans
  `internal/content` — rendu **tel quel** sans barre de titre ni invite
  (`internal/render.RawScreen`/`screenRows`, pas de saut de ligne final). Repli sur `lines`
  si pas de buffer. Moteur : une touche pour sortir.
- **Studio, onglet « Écran »** : éditeur **caractère par caractère** sur la grille 40×28,
  **fidèle à l'ULA** — il travaille sur le **buffer écran d'octets** où les **attributs sont
  des cases** qu'on pose explicitement (encre/fond/texte), exactement comme l'Oric (un
  changement d'encre occupe une case et s'applique jusqu'au prochain attribut). Plus de
  coloration « par cellule » incohérente avec la sérialisation ; l'inverse reste par
  caractère (bit 7). Pinceau = octet à poser (caractère + inverse, ou attribut via pastilles
  encre/fond + boutons alt/cli/norm) ; clic = peindre, clavier = écrire (flèches/curseur,
  ⌫/Suppr) ; palette BBS pour le caractère. Créer/Charger/Enregistrer (buffer ↔ base64).
- `/api/screen` rend les pages `raw` via `RawScreen` ; rendu d'aperçu ULA partagé
  (`renderScreenBuf`, réutilisé par l'aperçu de page). Tests `render` ; validé serveur + studio.
- Onglet Édition : un **compositeur** assemble une ligne caractère par caractère en mêlant
  **texte normal** (champ « + texte ») et **glyphes BBS** (clic dans la palette), avec
  **aperçu ULA en direct**. « Insérer comme ligne » ajoute la ligne à la page courante,
  **regroupée en segments** selon le mode (alt vs normal). Remplace l'insertion glyphe-à-glyphe
  dans le champ focalisé.

### Ajouté (Police « BBS Oric » — charset alternatif redéfini, art BBS)
- L'Oric n'ayant pas de glyphes orientés BBS (contrairement à PETSCII/ATASCII), on
  **redéfinit son charset alternatif** : nouvelle **police BBS 6×8** (filets/cadres simples
  et doubles, blocs ▌▐▀▄█, trames ░▒▓, symboles ►◄▲▼★•✓…), 35 glyphes.
- `tools/genfont` : générateur (glyphes décrits en ASCII-art, **source unique**) produisant
  `studio/web/altcharset.js` (simulateur) et `client/altcharset.s` (données pour `$B800`).
  Cible `make genfont`.
- Studio : le simulateur ULA rend les cellules `altCharset` avec la police BBS (les filets se
  raccordent), et une **palette** (onglet Édition) insère les glyphes dans le champ courant.
- Accès via `altCharset: true` (ligne ou segment). Rendu validé (cadre `┌───┐`).
- **Terminal Oric** (`client/term.s`) : `load_altcharset` copie la police dans `$B800` au
  démarrage ; `client/build.sh` concatène `term.s` + `altcharset.s`. **Validé dans
  l'émulateur** : un Oric affiche un cadre dessiné en police BBS (rendu ULA réel).

### Ajouté (Studio — aperçu fidèle « simulateur ULA » + rendu partagé)
- **`internal/render`** : paquet **partagé** produisant le flux OASCII d'un écran de page
  (`Screen`) — **source unique** réutilisée par le serveur (`server/internal/bbs`) et le
  studio ; supprime la duplication serveur/aperçu.
- Studio : l'aperçu HTML approximatif est remplacé par un **simulateur ULA** en JS/canvas
  reproduisant `oric1-emu/src/video/video.c` (attributs encre/fond, double hauteur,
  clignotement, inverse, charset alt approximé) — **sans ROM ni émulateur au runtime** :
  la **police Oric standard** est extraite une fois du ROM (offset `0x3C78`, 96×8) et
  embarquée (`studio/web/charset.js`). Endpoint `GET/POST /api/screen` → octets OASCII ;
  rendu client (`240×224`, mise à l'échelle pixelisée).
- **Corrigé** : l'**inverse** est désormais **par caractère (bit 7)**, conforme à l'ULA
  (`InverseText`), et non un attribut sériel erroné (l'octet 29 réglait en fait le mode
  vidéo). `oascii.InverseAttr`/`Builder.Inverse` retirés.
- Tests : `render` (menu/contenu/segments/inverse bit 7), `oascii` (InverseText). Rendu de
  la police validé (ASCII-art « BIENVENUE »). Engine refactoré sur `render.Screen` (suite verte).

### Ajouté (Contenu — style Oric complet + multicolore par segments)
- `internal/content` : un **`Style`** (encre, fond, **clignotement**, **double hauteur**,
  **charset alternatif/semi-graphiques**, **inverse**) porté par une ligne **et** par chaque
  **`Span`** ; une `Line` peut être un texte simple ou une suite de `segments` stylés →
  **plusieurs couleurs/attributs sur une même ligne**.
- `internal/oascii` : `InverseAttr`/`Builder.Inverse` (vidéo inverse) ; `Builder.Attrs`
  (clignotement/double hauteur/charset alt en un octet).
- `server/internal/bbs` : rendu par **delta de style** (`writeLine`/`emitStyle`) — n'émet que
  les changements d'attribut le long de la ligne (économie de cases écran).
- Studio : éditeur de lignes **par carte** avec contrôles encre/fond + bascules C/H/A/I et
  **découpage en segments** ; l'aperçu rend fond, clignotement, double hauteur, inverse
  (échange encre/fond) et approxime les semi-graphiques.
- Docs `content.md`. Tests : oascii (Inverse), aperçu (segments/inverse/alt), moteur
  (multicolore). Octets Téletexte vérifiés (hexdump).

### Corrigé (Login : « Se connecter » revenait au menu via nc/clients ligne)
- Un client en mode ligne (nc…) envoie « 1\r\n » : le menu lisait `1` (touche unique) mais
  le `\r\n` résiduel était lu comme **ligne vide** par le premier `ReadLine` de l'applet
  login → annulation immédiate → retour au menu. `ReadKey` **draine désormais les CR/LF/NUL
  déjà bufferisés** derrière la touche (sans bloquer). Sans effet sur un terminal Oric en
  mode caractère (pas de résidu). Test `ReadKey` dédié ; flux inscription/login revérifié `nc`.

### Ajouté (Studio — édition des profils de déploiement)
- L'onglet Configuration permet d'**éditer les profils** (LOCAL, HOST, USER, PORT,
  CONTENT_PATH, SERVICE, RELOAD) et de les **enregistrer** dans
  `deploy/profiles/<site>/<env>.conf` — plus besoin d'éditer les `.conf` à la main.
- `studio/internal/deploy` : `Profile.Marshal` (format `.conf`) + `SaveProfile` (écriture
  atomique, anti-traversée) + tags JSON. `studio/cmd/forge` : `GET`/`POST /api/profile`.
- UI : sélecteur de profil → formulaire (champs masqués selon LOCAL) → « Enregistrer le
  profil » ; bloc Déployer (Simuler/Déployer) en dessous.
- Tests : `SaveProfile` aller-retour + refus de traversée (site/env). Vérifié via `curl`.

### Modifié (Contenu — fusion menu/page en un seul type de page)
- Suppression du champ `type` : une **page** a un titre et, optionnellement, du **texte**
  (`lines`) **et/ou** des **choix** (`entries`). Avec `entries` → écran interactif (le texte
  s'affiche au-dessus des choix) ; sans `entries` → écran de contenu. Permet **texte + choix
  sur le même écran** (impossible avant).
- `internal/content` : `Page` sans `Type` ; validation simplifiée. `server/internal/bbs` et
  `studio/internal/preview` : rendu/navigation basés sur la présence d'`entries`/`applet`.
- `content/site.json` : champs `type` retirés (le `type` reste ignoré s'il traîne dans un
  vieux JSON — compat lecture).
- Studio : plus de sélecteur de type ni de boutons `+ menu`/`+ page`/`+ applet` ; un seul
  **« + page »**, le formulaire édite texte **et** choix. Le graphe dérive l'étiquette
  (menu/page/applet) de la structure.
- Tests : page menu avec texte d'intro, validation, parsing. Vérifié via `nc`.

### Ajouté (Contenu — entrées-applet : un menu peut proposer plusieurs applets)
- `internal/content` : une `Entry` de menu peut désormais porter `applet` (+ `next`) **au
  lieu** de `target` — une page (menu) peut donc **contenir plusieurs applets**, présentés
  comme des choix. Validation adaptée (target **ou** applet requis).
- `server/internal/bbs/engine.go` : une entrée-applet lance l'applet via le registre puis,
  en cas de succès, navigue vers `next` (sinon reste sur le menu). Factorisation `runApplet`.
- `content/site.json` : la porte d'auth utilise des **entrées-applet** (`login`/`register`/
  `guest` directement sur le menu `accueil`) ; pages applet séparées supprimées.
- Studio : l'éditeur d'entrées propose le type **→ page** ou **▶ applet** (nom + `next`) ;
  le graphe de navigation relie les entrées-applet à leur `next` et affiche `▶applet`.
  Le studio ne crée plus de *page* de type applet (bouton « + applet » et option de type
  retirés) — les applets se lancent via une entrée de menu. Le moteur garde la compat des
  pages applet écrites à la main.
- Tests : entrée-applet (lancement + navigation `next`), validation. Validé via `nc`.

### Modifié (Studio Forge — profils PAR SITE + enregistrement indenté)
- Les profils de déploiement sont désormais **propres à chaque site** :
  `deploy/profiles/<site>/<env>.conf` (chaque site a son trio `dev`/`int`/`prod`), au lieu
  d'un jeu global. API : `GET /api/profiles?site=`, `POST /api/deploy?site=&profile=&dryRun=`.
  Exemples déplacés sous `deploy/profiles/site/` ; `.gitignore` couvre `deploy/profiles/**/*.conf`.
- L'enregistrement **ré-indente** le JSON (`json.Indent`, 2 espaces) : fichiers lisibles,
  diffs git stables, toutes les clés préservées (y compris `_comment`).
- Tests : `LoadSiteProfiles` (par site, répertoire absent toléré, refus de traversée), `SiteKey`.

### Ajouté (Studio Forge — incrément 2 : profils & déploiement dev/int/prod)
- **ADR-0003** (`docs/adr/0003-studio-forge.md`) : studio web Go, `internal/` partagé,
  déploiement par profils, studio = source de vérité (écrase + sauvegarde).
- `studio/internal/deploy` : profils `KEY=VALUE` (`deploy/profiles/<nom>.conf`, l'`.example`
  sert de défaut, le `.conf` réel gitignoré prime). Déploiement : **valide → sauvegarde
  horodatée → écrase → reload** ; **dry-run** (journal des actions). `dev` = local (copie,
  hot-reload) ; `int`/`prod` = ssh/scp (sans dépendance).
- `studio/cmd/forge` : API `GET /api/profiles`, `POST /api/deploy?profile=&dryRun=`.
- UI : sélecteur de profil, boutons **Simuler** / **Déployer** (confirmation), journal.
- `deploy/profiles/{dev,int,prod}.conf.example` ; `.gitignore` couvre les `.conf` réels.
- Tests : parsing/priorité des profils, déploiement local (backup+écrasement), refus d'un
  contenu invalide, dry-run sans effet. **Validé end-to-end** : forge → deploy `dev` →
  bbsd recharge à chaud (vérifié via `nc`).

### Ajouté (Studio Forge — incrément 1 : éditeur web + aperçu OASCII)
- Nouveau sous-projet **`studio/`** : app web **Go** locale (stdlib, assets embarqués) pour
  éditer le(s) `site*.json` (pages `menu`/`page`/`applet`, porte d'auth).
- `studio/internal/store` : liste/charge/enregistre les sites ; **valide via `internal/content`
  (même validation que le serveur)** avant écriture atomique ; refuse la traversée de chemin.
- `studio/internal/preview` : rend une page en **HTML coloré 40 colonnes**, fidèle au moteur
  (réutilise la palette `internal/oascii` + `content.Ink`).
- `studio/cmd/forge` : serveur `net/http` (bind **127.0.0.1**, sans auth) ; API
  `GET /api/sites|site`, `POST /api/validate|save|preview`.
- `studio/web` : éditeur vanilla JS (sélection site/page, formulaires par type, aperçu live,
  Valider/Enregistrer).
- Cible Make `make studio`. Tests : store (valide-avant-écriture, anti-traversée), preview
  (rendu menu/page/applet, échappement HTML), handlers HTTP (`httptest`). Smoke test `curl` OK.

### Modifié (Restructuration en 3 sous-projets : server / client / studio)
- Le dépôt s'organise en **`server/`** (serveur Go : `server/cmd/bbsd` + `server/internal/`
  bbs/server/user), **`client/`** (terminal Oric, ex `oric-client/`) et **`studio/`** (à venir).
- Les paquets **partagés** `content` et `oascii` restent dans l'**`internal/` racine** afin
  d'être réutilisables par le serveur **et** le studio (règle de visibilité Go) — zéro
  duplication de validation/rendu.
- Chemins d'import des paquets déplacés réécrits ; `Makefile` (`make client`, `make studio`),
  `scripts/test-emulateur.sh`, `deploy/vps-deploy.sh`, `.gitignore` et `docs/architecture.md`
  mis à jour. Déplacement pur : **suite de tests inchangée et verte**, `.tap` client identique.

### Ajouté (Login — incrément 3 : applets auth + câblage + déploiement)
- `internal/bbs/login.go` : applets **`login`**, **`register`**, **`guest`** (enregistrés
  via `init`). Pseudo + mot de passe saisis ligne par ligne (RETURN), **mot de passe visible
  à l'écran** (averti ; TLS couvre le transport), 3 tentatives, annulation par champ vide.
  Accueil personnalisé « Bonjour {pseudo} — Appel n°{N} » (façon BBS), accès invité en
  lecture seule. Tests bout-en-bout (login OK, mauvais mot de passe, invité, inscription
  persistée).
- `content/site.json` : **porte d'auth au CONNECT** — page de départ `accueil`
  (Se connecter / Créer un compte / Invité) menant aux applets, `next` vers `main`.
- `cmd/bbsd` : flag **`-users <fichier.json>`** (comptes persistés ; vide = mémoire).
- Déploiement : unité systemd `-users /var/lib/bbsoric/users.json` + **`StateDirectory=bbsoric`**
  (répertoire RW possédé par le DynamicUser, autorisé malgré `ProtectSystem=strict`).
- Validé end-to-end via `nc` : inscription → hachage persisté dans `users.json` →
  reconnexion et login (pseudo insensible à la casse) → compteur d'appels incrémenté.
- **Terminal Oric** : vérifié que `oric-client/term.s` émet **déjà** chaque frappe
  immédiatement (boucle `main` : `key_scan` → `ser_tx`, pas de tampon de ligne) — la saisie
  touche unique fonctionne **sans modifier le terminal** (ADR-0002 corrigé). `.tap`
  réassemblée à l'identique (non-régression) ; l'émulateur confirme le pipeline
  clavier→numérotation→CONNECT→réception.

### Ajouté (Login — incrément 2 : moteur d'applets + saisie touche unique)
- **ADR-0002** (`docs/adr/0002-modele-de-saisie.md`) : terminal Oric en **mode caractère**,
  `ReadKey` (menus, « appuyez sur une touche ») + `ReadLine` (champs texte).
- **ADR-0001 révisé** : le login devient un **applet** lancé par une page de type `applet`
  (porte au CONNECT via la page de départ JSON), au lieu de cibles spéciales par fonction.
- `server.ReadKey()` : lit une touche unique (filtre IAC, ignore CR/LF/NUL résiduels). Tests.
- `internal/content` : nouveau **type de page `applet`** (champs `applet` + `next`) +
  validation. La page reste du JSON ; elle référence un applet Go par son nom.
- `internal/bbs` : **registre d'applets** (`Register`/`Applet`/`AppContext`/`Outcome`/
  `SessionState`). Le moteur (`engine.go`) navigue désormais **menus et pages à la touche
  unique** (ReadKey) et **dispatche les pages applet** (succès → page `next`, applet inconnu
  géré proprement). Tests : dispatch + navigation `next`, applet inexistant non bloquant,
  navigation menu touche unique validée (+ démo `nc`).

### Ajouté (Login — incrément 1 : modèle utilisateur + store haché)
- **ADR-0001** (`docs/adr/0001-login-composant-page.md`) : le login sera un **composant
  interactif isolé** appelé par une **page via une cible spéciale** (`__login__`,
  `__register__`, `__guest__`, `__logout__`), dans le prolongement de
  `__quit__`/`__back__`/`__home__`. La page reste du JSON pur. Persistance hachée, mot de
  passe en clair à l'écran assumé (TLS couvre le transport), no-echo repoussé.
- `internal/user` : modèle `User` (`Handle`, `PassHash`, `Created`, `LastLogin`, `Calls`)
  et `Store` fichier JSON avec **verrou** (accès concurrents) et **écriture atomique**
  (fichier temporaire + `rename`). API : `Register`, `Authenticate`, `Get`, `Count`.
- Mots de passe **jamais en clair** : hachage **PBKDF2-HMAC-SHA256** (`crypto/pbkdf2`,
  **stdlib** Go 1.24+, **aucune dépendance ajoutée**), sel aléatoire par compte, format
  encodé auto-descriptif `pbkdf2$sha256$<iter>$<sel>$<hash>`, comparaison à temps constant.
- Validation pseudo (2–16 caractères ASCII alphanum + `-`/`_`) et mot de passe (≥ 4).
- Tests : hash/verify, sel aléatoire, rejet des hachages malformés, validation, doublons
  (insensible à la casse), incrément des appels + `LastLogin`, persistance après
  réouverture, fichier absent, **accès concurrent** (suite verte avec `-race`).

### Ajouté (Contenu dynamique — flux de pages JSON rechargé à chaud)
- `internal/content` : modèle `Site`/`Page`/`Entry`/`Line` + parsing/validation JSON +
  `Store` qui **recharge le fichier à chaud** (surveillance mtime ; en cas d'erreur,
  l'ancienne version est conservée). Cibles de navigation `__quit__`/`__back__`/`__home__`,
  couleurs d'encre par nom. Contenu intégré par défaut si aucun fichier.
- `internal/bbs/engine.go` : moteur générique piloté par le `Site` (rendu menus/pages +
  navigation par pile) — remplace le menu codé en dur.
- `cmd/bbsd` : flag `-content <fichier.json>`.
- `content/site.json` : flux de pages éditable (menus, pages, sous-menu `Services`).
- `docs/content.md` : format JSON documenté.
- Déploiement : unité systemd `-content /etc/bbsoric/site.json` ; le script **sème** le JSON
  à l'initialisation seulement (les éditions à chaud sur le serveur ne sont jamais écrasées).
- Tests : parsing/validation, rechargement à chaud, conservation sur fichier invalide,
  validité de `content/site.json`. Rechargement validé end-to-end (ajout d'une entrée au
  menu visible sans redémarrage).

### Validé (TLS vérifié bout-en-bout — AT$CV1)
- Test émulateur (`--serial picowifi`) de l'entrée 5 du répertoire **avec vérification du
  certificat** : le terminal charge le CA racine **ISRG Root X1** (`AT$CA=` → « CA stored:
  1939 bytes »), active `AT$CV1`, puis dial `ATDT#pavi.3617.fr:6992`. Résultat :
  **`TLS session up (TLSv1.3, verified)`** → `CONNECT` → bannière BBS servie à travers le
  tunnel TLS vérifié (`docs/img/tls-verified-atcv1.png`).
- Confirme que le cert Let's Encrypt servi par Caddy est de confiance et que la chaîne
  (leaf → YE1 → Root YE → ISRG Root X2 → ISRG Root X1) valide côté Pico W.
- Détail d'upload : le picowifi segmente la capture `AT$CA=` sur **LF** (`\n`), le `\r` est
  ignoré — le PEM doit donc être envoyé en lignes terminées par `\n`.

### Modifié (Production — terminaison TLS par Caddy + Let's Encrypt)
- Le TLS de `pavi.3617.fr:6992` est désormais **terminé par Caddy** (CT 130, module
  `caddy-l4`/layer4) avec un **vrai cert Let's Encrypt** (`subject=CN=pavi.3617.fr`), au lieu
  du cert auto-signé de bbsd. Caddy déchiffre et proxifie le telnet en clair vers `bbsd`
  (`.2:6502`). NAT MikroTik `:6992` redirigé vers Caddy (`.130`). Chaîne et config
  versionnées dans `deploy/caddy-tls.md`.
- Le Pico W peut maintenant **vérifier le certificat** (`AT$CV1` + CA Let's Encrypt).
- `bbsd -tls-addr 6992` **retiré** de l'unité systemd : bbsd ne sert plus que le telnet clair
  (`.2:6502`) vers Caddy. Sites web de CT 130 (meteolib/3617/lamatronne…) vérifiés sans régression.

### Ajouté (Production — écoute TLS 6992, accès public ouvert)
- **Déployé** : `bbsd -tls-addr 0.0.0.0:6992` sur le LXC pavi3617 (en plus du telnet 6502).
  Les deux ports écoutent (même process). Unité systemd mise à jour.
- **Forward NAT ouvert** sur le routeur MikroTik (dst-nat `:6992` → `192.168.1.2:6992`,
  calqué sur la règle 6502). **`pavi.3617.fr:6992` est joignable publiquement en TLS** :
  testé (`openssl s_client` → bannière + navigation `1` → écran Informations).
- L'entrée 5 du répertoire (`BBS Oric TLS`) est désormais opérationnelle depuis un vrai
  Oric sur Internet (`ATDT#pavi.3617.fr:6992`).

### Ajouté (Terminal Oric — dial TLS réel + build autonome)
- **Numérotation TLS** : le protocole 2 (TLS) compose désormais **`ATDT#hôte:port`** (le `#`
  ouvre un appel TLS-terminé côté Pico W, firmware v0.2.0) au lieu d'un `ATD` simple.
  **Validé de bout en bout** dans l'émulateur (build OpenSSL, `--serial picowifi`) : TX
  `ATDT#127.0.0.1:6510` → `TLS session up (TLSv1.3)` → bannière BBS rendue à travers un
  proxy TLS de test (`docs/img/tls-dial.png`).
- `oric-client/bin2tap.py` : générateur `.tap` autonome (Python) — le build ne dépend plus
  du `bin2tap` externe de l'émulateur (qui peut être nettoyé). `build.sh` l'utilise.

### Ajouté (Terminal Oric — multi-modem + saisie manuelle host/port/protocole)
- **Abstraction des E/S série** via `ACIAPTR` (pointeur ZP sur la base de l'ACIA) +
  primitives `ser_tx`/`ser_rx_ready`/`ser_rx`. Un seul `.tap` gère 2 backends, sélectionnés
  par un **menu modem** au démarrage :
  - **1 = ACIA 6551 direct** (`$031C`)
  - **2 = LOCI / Pico W** (`$03A0`) — même interface 6551, base différente.
  (DTL2000 exclu : V23/Minitel, sans AT ni TCP moderne. Les deux backends validés
  end-to-end → `CONNECT to pavi.3617.fr:6502`.)
- **Saisie manuelle** (option `M` du répertoire) : champs **hôte**, **port**, **protocole**
  (1=telnet/raw fonctionnel, 2=TLS), avec écho. Le terminal compose `ATD hôte:port`.
  Routine `input_line` (saisie de ligne avec écho + anti-rebond `wait_release`).
- **TLS** : assuré par le **modem** (Pico W) — l'Oric ne fait pas de crypto ; côté Oric le
  protocole choisit la commande. *(Le dial TLS `ATDT#` a depuis été implémenté et validé via
  le backend `--serial picowifi` — voir l'entrée « dial TLS réel ».)*
- Captures : `docs/img/modem-menu.png`, `docs/img/manual-entry.png`.
- Note de test : `--type-keys` maintient une touche enfoncée jusqu'à une touche identique
  ou la fin de chaîne, ce qui rend la navigation multi-écrans difficile à automatiser
  (artefact de l'outil, pas du terminal) ; chaque étape validée séparément.

### Ajouté (Terminal Oric — répertoire + numérotation AT autonome)
- `oric-client/term.s` : **répertoire (phonebook)** au démarrage + **numérotation Hayes
  autonome**. Le terminal affiche une liste de BBS (BBS Oric prod, ParticlesBBS, Altair,
  Heatwave), l'utilisateur choisit (1-4), et le terminal compose lui-même `ATD<hôte:port>`
  vers le modem, puis bascule en mode terminal — plus besoin de configurer le modem.
- Routines ajoutées : `print_string`, `send_string`, `get_key`, `reset_cursor` ; données
  répertoire + table d'adresses des cibles de numérotation.
- Validé sur émulateur (`--serial modem`) : sélection → `ATD` composé → `CONNECT` vers
  `pavi.3617.fr:6502` (`docs/img/phonebook-dial.png`).

### Ajouté (Déploiement production)
- `deploy/` : mécanisme de déploiement repris du projet telenet — `deploy.conf` (cible LXC
  pavi3617 via VPN mustang), `bbsoric.service` (unité systemd durcie, port 6502), `vps-deploy.sh`
  (compile linux/amd64 statique → copie → installe l'unité → restart + vérifie l'écoute).
- `Makefile` : cibles `build`, `test`, `vet`, `run`, `oric-client`, `deploy`, `deploy-build`.
- Déploiement comme service systemd dédié `bbsoric` sur le port 6502 (libre ; le service
  `telenet-bbs` préexistant était inactif), sans impacter `telenet-serveur`/`telenet-compagnon`.
- **MISE EN PRODUCTION** : service `bbsoric` `enabled`+`active` sur le LXC pavi3617, exposé
  publiquement et validé sur **`pavi.3617.fr:6502`** (bannière + navigation `1`/CR depuis
  l'Internet public). `DynamicUser=yes` (l'utilisateur `bmarty` n'existe pas sur le LXC ;
  évite de tourner en root).

### Ajouté (Sprint 2 — émission clavier / BBS interactif)
- `oric-client/term.s` : **émission clavier (TX)**. Scan complet de la matrice 8×8
  (protocole PSG-via-VIA repris de `Oric asteroids/src/asm/input.s`), table ASCII par
  position (depuis `src/io/keyboard.c`), anti-rebond (1 caractère par appui), envoi ACIA et
  **écho local** à l'écran. Le terminal lit/affiche (RX) et envoie les frappes (TX).
- `internal/server/session.go` : `ReadLine` termine désormais la ligne sur **CR seul** (`$0D`,
  ce qu'envoie l'Oric sur RETURN) en plus de LF/CRLF, sans lecture bloquante.
- Test `TestCROnlyLineTermination`.
- `docs/img/sprint2-keyboard-nav.png` : **preuve d'interactivité** — saisie `1` + RETURN depuis
  l'Oric émulé → écran « Informations système » affiché (écho local + navigation serveur).
- Validation : touches espacées toutes transmises (`a/b/c/RETURN`) ; navigation menu de bout en
  bout via `--type-keys`. (Caveat : `--type-keys` en rafale très rapide pendant le chargement de
  la bannière peut perdre des frappes — non représentatif d'une frappe humaine.)

### Ajouté (Sprint 2 — moteur de menus BBS)
- `internal/bbs/menu.go` : moteur de menus. Menu principal coloré (OASCII) + écrans
  **Informations système**, **À propos**, **Livre d'or** (placeholder), navigation par choix
  (1/2/3/Q), retour au menu via RETURN, sortie propre. Helper `firstKey` (routage des choix).
- `internal/bbs/welcome.go` : `WelcomeHandler` enchaîne désormais bannière + boucle de menu
  (remplace l'écho « hello world »).
- Tests : `TestBannerAndMenu`, `TestMenuNavigationAndQuit`, `TestFirstKey` (intégration via
  socket réelle, lecture par octets robuste aux invites sans `\n`).
- `docs/img/sprint2-menu.png` : menu coloré rendu dans l'émulateur (validation visuelle).
- Note : la navigation depuis l'Oric nécessitera l'émission clavier (TX) du terminal — prochaine étape.

### Ajouté (Sprint 1 — terminal Oric + validation émulateur)
- `oric-client/term.s` : terminal Oric minimal en 6502 (assemblage `xa`). Lit l'ACIA 6551
  `$031C` (9600 8N1, polling) et écrit **directement en VRAM `$BB80`** pour rendre les
  attributs sériels OASCII ; gère CR/LF/scroll, clamp 40 colonnes ; chargé/exécuté en `$1000`.
- `oric-client/build.sh` : assemblage + génération `.tap` autorun (via `bin2tap`).
- `scripts/test-emulateur.sh` : test d'intégration headless (serveur + `oric1-emu` en série
  TCP + capture d'écran PPM/PNG).
- `docs/img/sprint1-banner.png` : **preuve visuelle** — la bannière colorée (jaune/cyan/vert/blanc)
  s'affiche correctement dans l'émulateur, validant la table d'attributs et toute la chaîne réseau.
- `oric-client/README.md` : doc du terminal Oric.
- `docs/test-emulateurs.md` : procédure de test validée et automatisée (ROM obligatoire,
  fast-load, FIFO RX, timings de capture).

### Ajouté (Sprint 1 — couche OASCII)
- `internal/oascii` : couche d'affichage Oric. `Builder` chaînable
  (`Ink/Paper/Blink/DoubleHeight/AltCharset/Text/Newline`), mode `Sticky` (ré-émission
  des attributs par ligne), encodeurs bas niveau `InkAttr/PaperAttr/TextAttr`, constantes
  de couleurs (palette Oric) et de dimensions (`Cols=40`, `Rows=28`).
- Table d'attributs Téletexte sériels **extraite du décodeur ULA de l'émulateur de
  référence** (`Oric1/oric1-emu`, `src/video/video.c`) → fiabilité garantie. 7 tests
  unitaires comparant les octets émis aux valeurs de l'émulateur.
- `internal/bbs` : bannière d'accueil **colorée** via OASCII (titre jaune, sous-titre
  cyan, infos vert/blanc). Flux d'octets d'attribut vérifié au hexdump.
- `docs/oascii.md` : spécification de la couche (nature réelle d'« OASCII », table
  d'attributs, palette, API, pièges de mise en page 40 colonnes).

### Ajouté (Sprint 0)
- Initialisation du projet (Sprint 0).
- `README.md` : présentation, cibles, spécificités Oric.
- `ROADMAP.md` : plan agile par sprints (0 à 5) et décisions ouvertes.
- `docs/etat-de-l-art.md` : analyse des serveurs BBS rétro existants (PETSCII BBS, RetroBBS, Magnetar, TheOldNet) et de l'écosystème de connexion (LOCI, WiFiModem, Oricutron).
- `docs/architecture.md` : architecture technique cible et couche de rendu « OASCII » (attributs Téletexte sériels).
- `docs/agile/backlog.md` : backlog produit initial (user stories).
- `.gitignore`.

### Modifié
- Prise en compte de la contrainte **serveur Internet public** : section « Exposition Internet »
  dans `docs/architecture.md` (port public, absence de TLS, surface d'attaque, hébergement, dispo) ;
  sécurité/exposition/hébergement remontées comme préoccupations transverses dans `ROADMAP.md`.

### Décidé
- **Langage serveur** : Go 1.26.
- **Hébergement** : VPS cloud avec IP fixe.
- **Port public** : `6502`.

### Ajouté (code — Sprint 0, story A3)
- Module Go `github.com/bmarty/bbsoric`.
- `internal/server` : serveur TCP (1 tâche/connexion) avec garde-fous d'exposition Internet
  (limite globale + par IP, timeout d'inactivité, journalisation slog), API `Session`
  (`Write/Println/ReadLine`) avec filtrage minimal des commandes telnet IAC. Méthode publique
  `Serve(ctx, listener)` (tests / activation socket).
- `internal/bbs` : `WelcomeHandler` — écran d'accueil 40 colonnes + boucle de commandes (HELP/QUIT).
- `cmd/bbsd` : démon configurable (`-addr`, `-max-conns`, `-max-conns-per-ip`, `-idle`, env `BBS_ADDR`),
  arrêt propre sur SIGINT/SIGTERM.
- Tests : `internal/server` (écho, filtrage IAC, limite par IP), `internal/bbs` (bannière + QUIT, centrage).
  Tous verts (`go test ./...`).
- `docs/test-emulateurs.md` : pipeline de test 100% local via l'émulateur **unique**
  `Oric1/oric1-emu` (Phosphoric) en `--serial tcp:`, + modem picowifi émulé.

### Corrigé
- Adressage ACIA précisé : `$031C` (Telestrat/oric1-emu) et `$03A0-$03BF` (LOCI MIA), en remplacement
  de la valeur `0x380` initiale.
- **Émulateur de test unique** : toute la documentation (README, architecture, roadmap, état de l'art,
  test-emulateurs) pointe désormais exclusivement vers `/home/bmarty/Oric1/oric1-emu`. Suppression des
  références à `oric2/Phosphoric` et d'Oricutron comme outils de test (Oricutron reste cité au seul
  titre du catalogue « état de l'art » des émulateurs publics).
