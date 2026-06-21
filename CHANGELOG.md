# CHANGELOG — BBS Oric

Toutes les modifications notables de ce projet sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) ;
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]

### Ajouté
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
- `docs/test-emulateurs.md` : pipeline de test 100% local via les émulateurs Oric présents
  (oric1-emu/Phosphoric `--serial tcp:`, Oricutron, modem picowifi).

### Corrigé
- Adressage ACIA précisé : `$031C` (Telestrat/oric1-emu) et `$03A0-$03BF` (LOCI MIA), en remplacement
  de la valeur `0x380` initiale.
