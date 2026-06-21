# CHANGELOG — BBS Oric

Toutes les modifications notables de ce projet sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) ;
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]

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
