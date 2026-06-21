# CHANGELOG — BBS Oric

Toutes les modifications notables de ce projet sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) ;
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]

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
