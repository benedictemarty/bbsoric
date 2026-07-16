# CHANGELOG — BBS Oric

All notable changes to this project are recorded here.
Format inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning [SemVer](https://semver.org/lang/en/).

## [Unreleased]

### Verified (E2 — terminal réel contre la production, 17/07/2026)
- **Le firmware terminal Oric (dans `oric1-emu`) se connecte au serveur de PRODUCTION**
  via le modem émulé et atteint le catalogue en direct. Nouveau `scripts/test-emulateur-prod.sh` :
  boote `client/term.tap`, compose `pavi.3617.fr:6502` (répertoire), navigue accueil → invité →
  menu → Catalogue → Logiciels, capture l'écran (`docs/img/e2-prod-catalogue.png` : grille
  LOGICIELS, « Page 1/131 2604 enreg. », légende `X=DL`). Meilleure validation possible sans
  Oric physique (firmware réel + séquence AT + réseau → BBS déployé). Reste le matériel réel (E2).

### Added (Epic J — Catalogue de téléchargement, 16/07/2026)
- **J5 — Outillage de déploiement du catalogue.** `gen-catalogue.py --merge-into <site.json>`
  greffe le catalogue (source + pages + entrée de menu, clé `8` sur `main`) dans un site.json
  existant, sans écraser le reste. Nouveau `scripts/deploy-catalogue.sh` : récupère le site.json
  **de prod** (préserve les éditions à chaud), fusionne, copie les fichiers téléchargeables,
  **valide** (`internal/content`), puis rsync `-files` + dépose le site.json + redémarre le
  service (avec `--dry-run`). Nouveau validateur `tools/validate-content` (valide un site.json
  avec le paquet du serveur). Doc `docs/catalogue-deploy.md`, `ORIC_LIB` dans
  `deploy.conf.example`. Testé en dry-run de bout en bout (fusion + validation OK).
- **J2 — Générateur conscient des tailles + peuplement de `-files`.** `gen-catalogue.py`
  n'indique plus un fichier `téléchargeable` que si un fichier réellement transférable
  (`.tap`/`.ort`/`.rom`/`.dsk`, le plus petit qui **tient**, cassette préférée) est ≤
  `--max-file-size` (défaut **30720 o**, le buffer terminal Oric) ; colonne `taille` ajoutée
  au catalogue. Option `--copy-files <dir>` : copie les fichiers téléchargeables dans le
  répertoire `-files` sous un nom court sûr (8.3, unique). Sur la bibliothèque complète :
  **2604 logiciels dont 1911 téléchargeables**, 710 magazines, 192 livres (catalogue ~1,2 Mo,
  1874 fichiers copiés). Vérifié dans le serveur réel : filtre + `X` initient le vrai
  téléchargement de `CENTI.TAP` (Centipede) présent dans `-files`.
- **J4 — Fiche détail enrichie.** La fiche `V` **replie** désormais les valeurs longues
  (auteur, description) sur plusieurs lignes au lieu de les tronquer à 22 colonnes
  (`wrapValeur`, au plus 4 lignes, marquage `...` si tronqué). Le générateur ajoute au
  catalogue des colonnes de métadonnées OricProgramsLib — **genre, éditeur, langue, joueurs,
  référence de capture d'écran** — qui apparaissent sur la fiche. *(Le « nombre de
  téléchargements » par item n'existe pas dans les données de la bibliothèque : non ajouté
  plutôt qu'inventé.)* Test `TestWrapValeur` ; vérifié dans le serveur réel.
- **Catalogue Logiciels / Magazines / Livres** basé sur DataWindow. Nouvelle option
  `fichier_colonne` sur un descriptif DataWindow (`internal/content`) : la colonne nommée
  porte le fichier téléchargeable ; la grille expose alors la touche **`X` (télécharger la
  ligne)** qui envoie ce fichier via XMODEM (helper `sendFileDownload` factorisé, réutilisé
  par l'applet `download`). La fiche détail existante (`V`), le filtre (`F`), le tri (`T`) et
  la pagination servent la consultation. Validation `fichier_colonne` (colonne existante) +
  légende `X=DL` affichée uniquement si un fichier est disponible.
  - **Contrainte assumée** : le téléchargement XMODEM vers l'Oric est borné (~30 Ko buffer
    terminal, garde 64 Ko). Les logiciels (petits `.tap`) sont téléchargeables ; les magazines
    et livres (PDF) sont **listés et consultables** mais leur colonne fichier reste vide.
  - **Filtre fixe par page** (`filtre_fixe` : colonne = valeur) : une page n'affiche que sa
    catégorie **sans que l'utilisateur saisisse un filtre**, tout en restant combinable (AND)
    avec le filtre `F` et le tri `T`. Appliqué en SQL (source SQLite) et en mémoire (source API).
    Le catalogue est donc **un seul source** (colonne `categorie`) présenté en 3 vues filtrées.
    Tests `TestListerFiltreFixe`, `TestValidateFiltreFixe`.
  - **Générateur** `scripts/gen-catalogue.py` : produit un Site catalogue (1 source `catalogue`
    + 3 grilles filtrées + 1 menu) depuis la bibliothèque **OricProgramsLib** (`data/catalog.json`
    pour les logiciels, `library/*/*.pdf` pour les magazines, `library/livres/*.pdf` pour les livres).
    Démo committée : `docs/examples/catalogue-demo.json` (8 items/catégorie, validée `content`,
    vérifiée dans le serveur réel : chaque vue ne montre que sa catégorie). Tests
    `TestDataWindowDownloadFromRow`, `TestValidateFichierColonne`.

### Changed (Sprint 11 slice 4 — hygiène, 16/07/2026)
- **S11.9 — Champ JSON `"type"` clarifié.** Documenté (struct `content.Page` + `docs/content.md`)
  comme un indice descriptif lisible, sans champ Go, jamais interprété (le genre de page est
  déduit des champs renseignés) et préservé au round-trip du studio — plutôt que retiré de
  quelques fixtures alors qu'il est utilisé partout.
- **S11.10 — Invariant de ré-émission d'attributs unifié.** Nouvelle méthode
  `oascii.Builder.ReemitAttrs` partagée par le mode sticky (`Newline`) et le repli du rendu
  (`internal/render`, `reemitState`) — fin de la double implémentation. Centrage `render.center`
  rendu rune-safe (largeur comptée en runes, pas en octets). Test `TestReemitAttrs` (couvre le
  cas multi-attributs auparavant non testé).
- **S11.11 — Nettoyages firmware Oric.** Commentaire trompeur corrigé dans `client/term.s`
  (l'octet `$3A` envoyé est bien « : », séparateur hôte/port) ; bloc mort retiré dans
  `client/hires.s` (`hires_fillbox` : `hy0` réassigné avant lecture + `lda hy1`/`sta hy1` no-op).
  Firmware réassemblé (`make client`, `term.tap` régénéré, artefact gitignoré).

### Changed (Sprint 11 slice 3 — robustesse & tests, 16/07/2026)
- **S11.6 — XMODEM durci et testé.** `internal/xmodem` distingue désormais un dépassement
  d'échéance (transitoire → ré-essai) d'une vraie erreur d'E/S (connexion fermée → remontée
  immédiate, plus de boucle jusqu'à `ErrTooManyNAK`) ; émet `CAN` au pair en cas d'abandon ;
  borne `Receive` à `maxReceiveBytes` (1 Mio, garde-fou mémoire, `ErrTooLarge`). Tests ajoutés
  couvrant la **branche somme de contrôle** (jamais exercée) : `TestSendChecksumMode`,
  `TestReadBlockChecksum`, plus `TestSendSurfacesIOError`, `TestReceiveRejectsOversize`.
- **S11.7 — Validation de contenu plus stricte au chargement.** Les motifs regex de colonne
  (`pattern`) sont compilés à la validation (`internal/content`) ; nouvelle
  `bbs.ValidateSiteApplets` détecte au démarrage un applet référencé mais non enregistré
  (appelée dans `main`) ; largeur de colonne par défaut dédupliquée via
  `content.DefaultColWidth` (validation ⇄ rendu). Tests `TestValidateColumnPattern`,
  `TestValidateSiteApplets`.
- **S11.8 — Couche HTTP de la Forge durcie.** Les endpoints mutants (`/api/validate`,
  `/api/save`, `/api/screen`, `/api/deploy`) exigent POST (405 + en-tête `Allow` sinon) ;
  les erreurs de lecture du corps ne sont plus ignorées (400) ; une sauvegarde invalide
  renvoie 400 (le détail reste dans le corps). Tests `TestMutatingEndpointsRequirePOST`,
  `TestHandleSaveInvalidReturns400`.

### Security (Sprint 11 slice 2 — 16/07/2026)
- **S11.3 — Déploiement studio à l'épreuve de l'injection shell.** `deployRemote`
  (`studio/internal/deploy`) interpolait `CONTENT_PATH`/`SERVICE` dans des commandes
  exécutées par le shell distant (`ssh … test -f … && cp …`, `systemctl reload …`) sans
  échappement. Ajout de `validateProfileFields` (jeu de caractères sûr `[A-Za-z0-9._@/-]`)
  appliqué à `Deploy` **et** `SaveProfile` : un champ de profil contenant un métacaractère
  (`;`, `$(…)`, backtick…) est refusé avant toute exécution. Test `TestDeployRejectsShellInjection`.
- **S11.4 — Rate-limiting anti brute-force sur l'authentification.** Nouveau composant
  `server/internal/throttle` (fenêtre glissante par clé, sûr en concurrence, horloge
  injectable). Câblé sur l'IP client : au plus **5 échecs d'auth par IP sur 5 minutes**, en
  complément du plafond de 3 essais par passage d'applet. Appliqué à `login` et à l'action
  `form` login. Tests unitaires du limiteur + intégration `TestLoginRateLimited`.
- **S11.5 — Écriture DataWindow réservée aux administrateurs.** Nouveau champ `User.Admin`
  (le 1er compte enregistré devient sysop ; flag éditable dans le JSON des comptes pour en
  promouvoir d'autres). L'écriture (CRUD) exige désormais `SessionState.IsAdmin()` ; la
  lecture reste ouverte à tous, et la légende de la grille n'affiche `N/E/D` qu'aux admins.
  Tests `TestFirstAccountIsAdmin`, `TestAdminFlagPersists`, `TestDataWindowGuestCannotCreate`
  (vérifié en échec sous l'ancien gate `LoggedIn`).

### Fixed (Sprint 11 slice 1 — bugs réels, 16/07/2026)
- **S11.1 — Présence propagée après login via une page `form`.** `applyFormAction`
  (`server/internal/bbs/form.go`) posait `State.User` sans appeler `setPresenceHandle` (à la
  différence de `login.go`) : un utilisateur authentifié par une page `form` restait affiché
  « connexion… » dans « qui est en ligne » et le chat. Ajout de `setPresenceHandle(ac.State,
  u.Handle)` dans les cas login **et** register. Test de régression `TestFormLoginSetsPresence`
  (vérifié : échoue sans le correctif, handle restant `connexion...`).
- **S11.2 — Refus propre d'un fichier trop volumineux au téléchargement.** L'en-tête de
  download (`downloadHeader`, `xfer.go`) code la taille réelle sur 16 bits ; au-delà de 65535
  octets elle était tronquée silencieusement (sauvegarde terminal corrompue). Ajout d'une garde
  `maxDownloadSize` (0xFFFF) refusant le fichier avec un message explicite avant tout transfert.
  Test `TestDownloadTooLarge`. *(Variante « en-tête élargi » côté firmware = S11.2b, non traitée.)*

### Added
- **Sprint 11 — Code quality & hardening** (`ROADMAP.md`) : décomposition de l'Epic I en
  12 tâches techniques (S11.1→S11.12) réparties en 4 slices (bugs réels → sécurité →
  robustesse/tests → hygiène), chacune reliée à sa story, avec `fichier:ligne` et test
  d'acceptation.
- **Epic I — Code quality & hardening** (`docs/agile/backlog.md`) : 11 user stories issues
  de l'analyse complète du code du 16/07/2026 (serveur / studio / `internal/` partagé /
  firmware Oric), classées bugs réels → sécurité → robustesse/tests → hygiène, chaque
  story citant le `fichier:ligne` concerné.
- **`CLAUDE.md`** — guide d'onboarding pour Claude Code : commandes (`make`), architecture des trois
  sous-projets (server / studio / client), couche OASCII, moteur BBS (Site JSON, applets, modèle d'entrée
  ReadKey/ReadLine), concepts transverses (DataWindow, HIRES, XMODEM, terminal), et conventions du projet.

### Fixed (HIRES — revue qualité, 28/06/2026)
- **`fillbox` ne dégénère plus en colonne d'1 pixel.** `fb_hline` (`client/hires.s`)
  échangeait `hx0`/`hx1` **en place** pour ordonner les bornes ; comme `hires_fillbox`
  réutilise `hx1` (x cible) à chaque ligne sans le recharger, dès la 2e ligne
  `hx0 == hx1` et le rectangle se réduisait à sa 1re ligne + un trait vertical.
  Déclenché quand le crayon était à droite de la cible (`curset 200,y` puis
  `fillbox 10,…`). Les bornes sont désormais copiées dans `fbx`/`fbxe` sans muter
  `hx0`/`hx1`.
- **`circle` ne dessine plus de pixels parasites sur le bord opposé.** `circ_points`
  calculait `cx±x` / `cy±y` en 8 bits ; un dépassement (`cx-x` négatif ou `cx+x` > 255)
  se repliait dans la zone visible et échappait au clamp de `setpixel_xy`, traçant un
  point fantôme (et, en mode couleur, un attribut d'encre) à l'opposé. Nouveaux helpers
  `cp_xp`/`cp_xm`/`cp_yp`/`cp_ym` qui clampent tout débordement hors champ.
- **Aperçu studio (`studio/web/app.js`)** : l'op `ink` ignore désormais une couleur
  absente ou hors `0–7` (au lieu de basculer silencieusement en noir/blanc) ; commentaire
  d'en-tête et hint mis à jour (l'aperçu est colorisé ; un tracé en encre 0 noir est
  invisible sur le fond noir, `paper` non rendu).
- Firmware réassemblé (`client/term.tap`), `go test ./...` vert.
- **Validé visuellement dans `oric1-emu`** (`docs/img/hires-regress-emu.png`) : un
  `fillbox` crayon-à-droite rend un rectangle plein (et non plus une ligne + un trait
  d'1 px) et un `circle` débordant à gauche est clippé net sans point fantôme au bord
  opposé.

![Validation régression HIRES](img/hires-regress-emu.png)

### Added (HIRES pages — ink colour, 27/06/2026)
- **HIRES drawing is now in colour.** The `ink` primitive sets the colour of the
  following shapes; the terminal renders it the Oric way — a **per-line ink serial
  attribute** placed at **column 0** of each drawn row. This carries the authentic
  hardware behaviour: the colour spans the whole row (two inks on one row *clash*,
  last wins) and the first cell (x 0–5) of a coloured row is spent on the attribute.
  With **no `ink` op** the rendering stays **monochrome** (white, column 0 free) — no
  regression to existing pages. `paper` is not rendered yet (black background).
- **Studio preview colourised** (per-pixel approximation of the per-line Oric attribute).
- **Validated in `oric1-emu`**: the demo now renders the cyan frame, red diagonals,
  yellow/red circle (visible clash) and white `ORIC` text
  (`docs/img/hires-demo-emu.png`). `go test ./...` green. Docs `docs/hires.md`.

### Added (HIRES pages — clean TEXT-mode return, 27/06/2026)
- **Leaving a HIRES page back to text now works.** New serial command **`1F FB`**
  (return to TEXT): the server emits it (tracked by a per-session flag in
  `engine.go`) before rendering a text page that follows a HIRES one. The terminal
  restores the text charset (`$9800` → `$B400`, overwritten while drawing), re-asserts
  the TEXT video attribute (`0x1A` at `$A000[0]`, latched by the ULA) and clears the
  screen. Without this a HIRES page was a dead-end (screen stuck in graphics mode).
- **Validated in `oric1-emu`** (`docs/img/hires-text-return-emu.png`: the text menu
  re-renders cleanly after a HIRES page + keypress) and by an integration test
  asserting `1F FB` precedes the text content on return. `go test ./...` green.

### Added (HIRES pages — Forge studio editor, 27/06/2026)
- **The Forge studio now edits HIRES pages** (« Édition » tab). A **« + page graphique
  (HIRES) »** button converts a page to graphics; a **primitive table** edits the `draw`
  list (op `curset`/`point`/`line`/`box`/`fillbox`/`circle`/`char`/`ink`/`paper` with the
  relevant X/Y/R/colour/char fields, reorder/remove); an **image import** reduces a
  picture to 240×200 and 1-bit-thresholds it into the `background` VRAM buffer.
- **Live 240×200 preview rasterized in JS** — a faithful mirror of the firmware
  (`client/hires.s`): setpixel/Bresenham/box/fillbox/midpoint-circle/char (via the
  shipped `ORIC_CHARSET`) + bitmap decode, painted on the ULA preview canvas (monochrome,
  white ink; per-attribute colour is rendered on the terminal, not yet in the preview).
- The page map labels HIRES pages **`graphique`** (`◨ hires`). Save/Validate go through
  the same `content.Parse` as the server. **Store round-trip test** (primitives + 8000-byte
  bitmap background preserved). `node --check` clean; forge serves the editor + JS. Docs
  `docs/hires.md`. *Completes the HIRES feature: server + terminal + studio.*

### Added (HIRES pages — terminal firmware + oric1-emu validation, 27/06/2026)
- **The Oric terminal now renders HIRES pages** (`client/hires.s`, concatenated by
  `client/build.sh`). On `1F FC`, `handle_rx` enters a HIRES-stream state feeding
  `hires_feed`, a state machine executing the opcodes:
  - **mode switch**: video attribute `0x1E` written to `$BB80` (latched by the ULA →
    persistent HIRES, verified against `oric1-emu` `video.c`), VRAM `$A000` cleared
    (8000 bytes to `$40`) and the 3 bottom text lines blanked;
  - **self-contained 6502 primitives** (no BASIC ROM dependency): `setpixel`
    (`$A000+y*40+x/6`, bit `5-x%6`), Bresenham line (x/y-major, 16-bit error),
    box/fillbox, midpoint circle, and `char` (6×8 glyph from the charset, backed up to
    `$9800` since clearing `$A000` overwrites the `$B400` text charset);
  - **bitmap blit**: RLE decoder writing decoded bytes to `$A000+offset`.
- **Validated visually in `oric1-emu`** (real Oric terminal → modem → local BBS):
  **both models** render — primitives (demo frame + circle + diagonals + `ORIC` text,
  `docs/img/hires-demo-emu.png`) and a bitmap background via blit with a rectangle on
  top (`docs/img/hires-bitmap-emu.png`). Debugging milestones: fixed `setpixel`'s
  `Y*40`, the Bresenham overflow, box corner bookkeeping, the charset overwrite, and an
  RLE-decoder bug where `lda hrun` clobbered the just-received count byte.
- **Known limits** (next increments): monochrome only (no HIRES `ink`/`paper` yet),
  large blits can saturate the serial FIFO (flow-control, cf. client-review #1), and a
  clean TEXT-mode return after a HIRES page is not wired yet. Docs `docs/hires.md`.

### Added (HIRES pages — server foundation, 27/06/2026)
- **New `hires` page type** for Oric high-resolution graphics (240×200), carrying
  **both** models the owner asked for: a **bitmap** background (`background`, full
  8000-byte VRAM, posted in one block) **and** **primitives** (`draw`: `ink`/`paper`/
  `curset`/`point`/`line`/`box`/`fillbox`/`circle`/`char`) drawn on top — combinable.
  Model in `internal/content` (`Hires`/`HiresOp`), validated by `Site.Validate()`
  (bitmap size, 240×200 bounds, colours 0-7, known ops).
- **Unified wire protocol** (`render.Hires`, single source server+studio): one HIRES
  **command stream** opened by the free serial sub-command **`1F FC`** (ignored by
  generic telnet clients), then 1-byte opcodes + fixed args until `HiEnd`
  (`internal/oascii/hires.go`). The bitmap is sent via `HiBlit` as an **RLE** stream
  (count/value pairs) — a blank screen ≈ 32 pairs instead of 8000 bytes. Coordinates
  fit one byte; the terminal keeps a drawing **pen**. Mixed mode (HIRES top + 3 text
  lines) reuses the « menu over background » pattern: a HIRES page with `entries`
  routes keys, otherwise one key returns.
- **Engine wiring**: `case page.Hires` emits the stream then navigates (menu) or waits.
- **Tested**: content validation (valid bitmap/primitives + 8 error cases), RLE
  round-trip + compression, `render.Hires` byte-stream (primitives + bitmap blit
  decodes back to source), and a TCP-driver integration test asserting the `1F FC …
  HiEnd` stream reaches the session. `go test ./...` green. Design `docs/adr/0005`,
  spec `docs/hires.md`. *Next slices: terminal firmware (6502) + studio editor.*

### Fixed (Oric terminal — manual entry / `input_line` regression, 27/06/2026)
- **`input_line` ate every typed character** (regression introduced with the
  backspace support): the normal-character path **fell through** into the backspace
  handler `il_back`, which stored + echoed the char then immediately decremented
  `INLEN` and erased it. Net effect: **manual host/port entry was completely broken**
  (the field stayed empty; only phonebook dialing worked, masking the bug). Fix: a
  missing `jmp il_skip` after the echo in `client/term.s`. `.tap` rebuilt; **validated
  visually in `oric1-emu`**: manual entry `127.0.0.1`/`6502` now reaches the protocol
  prompt, dials the **local** BBS and renders the grid.
- **New driver `scripts/test-emulateur-grille.sh`**: end-to-end capture of a DataWindow
  grid in `oric1-emu` (modem menu → manual entry → `modem:` dial → BBS session →
  grid → selection). Records the `--type-keys` timing trick the terminal needs:
  `input_line` does one `get_key` + `wait_release` per char and `--type-keys` *holds*
  a key until the next event, so each character — and the Enter especially — needs a
  `\p` pause to release. Captures archived: `docs/img/datawindow-grid-emu.png` (grid),
  plus `…-tri.png` (sort `T` → footer `tri Nom+`) and `…-fiche.png` (detail `V` card),
  driven end-to-end through the emulator. Docs `docs/client-review.md` updated (resolved
  #5c, deferred #12 partly addressed) and `docs/datawindow.md` (interaction captures).

### Added (DataWindow — full structured editing in the Forge studio, 27/06/2026)
- **The Forge studio now edits the whole DataWindow model visually**, no more JSON
  by hand (this closes the increment deferred on 27/06/2026).
  - **New « Données » tab** to manage `sources_donnees`: create/load/delete a source,
    pick its **type** (SQLite CRUD or **REST API** read-only), edit **typed columns**
    (key, type, label, primary key, auto-increment, required, max length, pattern,
    default value, auto-date), the SQLite **seed rows**, the API config (`url`,
    `racine`, `ttl_sec`), plus default sort and rows-per-page. Renaming a source or a
    column carries references over (grid pages follow) and **preserves column order**.
  - **Grid descriptor editor** in the « Édition » tab: a **« + grille de données »**
    button converts a page to a grid; the editor sets the **source**, **displayed
    columns** (add/remove, reorder via ↑/↓, per-column **width** with a live **/40
    budget** counter), header/rows/selection **colours**, rows-per-screen and the
    **editable** flag (N/E/D). Page map already labels these pages `grille`.
  - Save/Validate go through the same `content.Parse` as the server (over-budget grid,
    unknown column, API source without URL… are refused before writing).
- **Tested**: a richer studio round-trip test (`studio/internal/store`) covering an
  API source, all `ColonneDef` fields (pattern/default/auto-date/max-length), seed
  rows and the grid colours/`lignes_max`/`editable` — none dropped. `node --check`
  clean on `app.js`; forge smoke-test serves the new tab and JS. `go test ./...` green.
  Docs `docs/datawindow.md` (« Édition dans le studio Forge »).

### Added (DataWindow — studio awareness + round-trip guard, 27/06/2026)
- **The Forge studio now recognizes DataWindow pages**: the page map labels them
  as `grille` and shows the bound source (`▦ <source>`). The studio already
  **round-trips** `sources_donnees` and per-page `datawindow` losslessly (it edits
  the full JSON object), and `content.Parse` validates them — covered now by a
  **regression test** (`studio/internal/store`: save → load → fields preserved).
  Full structured source/data editing in the studio remains a later increment.

### Added (DataWindow — REST API sources, 27/06/2026)
- **A DataWindow source can be backed by a REST endpoint** (`type_source: "api"`)
  instead of SQLite — **read-only** live data (weather, news, …), enabling the
  teased *Meteo/Actualites* services. The endpoint returns a JSON array (or an
  object whose `racine` key holds it); fields map to columns by name. Filter/sort/
  pagination are applied **server-side** on the fetched rows, with a **TTL cache**
  (`ttl_sec`, default 60 s). Writes are refused on API sources (`errSourceLectureSeule`).
  New `server/internal/datawindow/api.go`; `Lister`/`Consulter` dispatch on
  `EstAPI()`; `InitialiserSource` is a no-op for API sources.
- **Tested** with an `httptest` server: list/filter/numeric-sort/pagination, the
  `racine` key, the TTL cache (1 HTTP hit for N reads, refetch after TTL), detail
  lookup, read-only refusal, HTTP-error propagation; plus content validation
  (API source needs `api.url`). Docs `docs/datawindow.md`. `go test ./...` green.

### Added (DataWindow — public « Annuaire BBS » grid, 27/06/2026)
- **First real DataWindow in the production content** (`content/site.json`): a
  **read-only « Annuaire BBS Oric »** grid (source `annuaire`, seeded with the Oric
  BBS directory) reachable from the main menu (entry `7`). `editable:false` → the
  grid shows no `N/E/D` (safe on a public server); browse/sort/filter only.
- **Deploy wiring**: the systemd unit passes `-data /var/lib/bbsoric/dwdata` so the
  engine is available in production (idle until the live content declares a source).
  Driver smoke-test confirms the rendered directory (6 entries, read-only legend).

### Added (DataWindow — interactive column sort, 27/06/2026)
- **Interactive sort** in the grid: key `T` cycles the sort column (default → col 1
  ASC → DESC → col 2 ASC → …), using the engine's `tri` parameter; the current sort
  is shown in the footer (`tri <col>+/-`). A sort triggers a full redraw (it reorders
  most rows). Legend updated to `F/T`. Integration test covers it. Docs updated.

### Added (DataWindow — typed data grids with SQLite + CRUD, 27/06/2026)
- **New « DataWindow » subsystem** (notion ported from the telenet server): typed
  *data sources* (SQLite tables) presented to the Oric user as a **paginated grid**
  with full **CRUD**, validation, sort and LIKE filter.
  - **Engine** `server/internal/datawindow` (port of telenet's `datawindow.go`):
    `Lister`/`Consulter`/`Creer`/`Modifier`/`Supprimer`/`Valider`, per-DB pool+mutex,
    seed import + column auto-migration. Backend **`modernc.org/sqlite`** (pure Go,
    no CGO — the only new dependency). `cellString` normalizes modernc's `[]byte` TEXT.
  - **Model** in `internal/content` (`SourceDonnees`/`ColonneDef` + page `DataWindow`
    descriptor); `Site.Validate()` checks source/column names (SQL-injection whitelist),
    displayed columns, widths and the **40-column budget** at load.
  - **Grid applet** `datawindow` rendered on the **`oascii.Screen` differential buffer**
    (moving the selection re-emits only 2 rows); selection/header in per-char **inverse**
    (bit 7). Keys: `+/-` select, `S/R` pages, `V` detail, `N/E/D` CRUD (if editable +
    logged in), `F/C` filter, `Q` quit. New `case page.DataWindow` in the engine.
  - **Wiring**: `-data <dir>` flag (`bbsd`) builds the engine and initializes sources;
    threaded through `WelcomeHandler`/`SessionState`/`AppContext` (now also carries `Site`).
- **Tested**: engine unit tests (init/seed idempotent, pagination/filter/sort, CRUD
  round-trip, validation, **SQL-injection guards**, `cellString`), content validation
  tests, and TCP-driver integration tests (grid display, filter, create → total grows).
  Driver smoke-test confirms the rendered grid, inverse selection move and filter.
  **Validated visually in `oric1-emu`** (real Oric terminal → modem → local BBS): the
  grid renders in 40 columns without wrapping, header coloured, the selected row in
  per-char inverse video (`docs/img/datawindow-grid.png`).
  Demo content `docs/examples/datawindow-demo.json`. Docs: `docs/datawindow.md`,
  `docs/adr/0004-datawindow-sqlite.md`. `go test ./...` + `go vet` + `go mod tidy` green.

### Added (Sprint 8 — S1: user-editable filename at reception, 27/06/2026)
- **The received file's name can be edited before saving.** After `xmodem_recv`,
  the terminal prompts `NOM (RET=DEFAUT)` (`client/term.s`, `edit_dlname`): RETURN
  alone keeps the server-proposed name, otherwise the typed `NAME.EXT` is parsed
  (`user_to_sedoric`) into the 12-byte Sedoric format (uppercased, `.`-split,
  9+3 padded) and **both Sedoric and LOCI** save under it. Reuses the proven
  `input_line` keyboard routine.
- **Runtime-validated** (`scripts/test-loci-emu.sh` case 3): the *real*
  `user_to_sedoric` extracted from `term.s` is assembled with `loci.s`;
  `editbuf="myfile.txt"` → the file is saved as **`MYFILE.TXT`** in the LOCI
  sandbox. Terminal `.tap` rebuilt clean (`$1000`→`$22EB`).

## [v0.1.3-alpha] — 2026-06-27
Publiée : https://github.com/benedictemarty/bbsoric/releases/tag/v0.1.3-alpha
(assets `term.tap`, `term-boot.dsk`). Téléchargement sous le vrai nom et à la
taille exacte (header v3), sauvegarde LOCI SD en fallback, download binaire
débloqué (`ATNET0`). Les entrées ci-dessous étaient sous `[Unreleased]`.

### Added (Download — exact file size + LOCI robustness/tests, 26/06/2026)
- **Files are now saved at their exact size** (no more XMODEM 128-byte padding).
  New **download header v3**: after the 12-byte name, the server appends the
  **real byte size** (lo, hi) (`server/internal/bbs/xfer.go`, `downloadHeader`,
  unit-tested `TestDownloadHeader`). The terminal reads it (`client/term.s`,
  `handle_rx` states 6/7 → `dlsize`) and clamps `XSIZE` to it before saving, so
  **both Sedoric and LOCI** write the precise length. `loci_save` now writes a
  **partial final block** (`nb = min(128, rem)`) instead of assuming a 128
  multiple (`client/loci.s`). Server + terminal evolve together (header grew).
- **LOCI robustness**: `loci_save` closes the file descriptor on a write error
  (`ls_wfail`) instead of leaking it — the LOCI exposes only 16 fds.
- **Versioned runtime test** `scripts/test-loci-emu.sh`: assembles a standalone
  6502 harness and runs it in `oric1-emu`, asserting the saved file byte-for-byte
  on **both** `--loci-flash` (host passthrough) and `--loci-sdimg` (real FAT16
  write path). Exercises a 200-byte file (full block + 72-byte partial block).
- Terminal `.tap` rebuilt clean (`$1000`→`$225A`); `go test ./...` + `go vet` green.

### Added (Download — LOCI SD save fallback, 26/06/2026)
- **The received file is now saved to the LOCI SD card when Sedoric is not
  resident.** New module `client/loci.s` (concatenated by `client/build.sh` after
  `sedoric.s`) writes the `$4000` buffer (`XSIZE` bytes) via the MIA API at `$03A0`
  (`OPEN`/`WRITE_XSTACK`/`CLOSE`), pushing the path and 128-byte blocks onto the
  XSTACK in reverse. LOCI presence is detected through the fixed signature opcodes
  at `$03B3/$03B5/$03B7` (`A9/A2/60`), independent of the ROM init (the terminal
  boots from tape). The filename is rebuilt from the 12-byte Sedoric 8.3 `dlname`
  into `NAME.EXT`.
- **Save dispatch** (`save_received`): `sed_save` now returns `A=1` if it persisted
  to Sedoric, `A=0` otherwise (`client/sedoric.s`); on `A=0`, `save_received` falls
  back to `loci_save`. `term.s` (`handle_rx` state 5) calls `save_received` instead
  of `sed_save`. User feedback: `SAUVE SUR CARTE SD` / `ECHEC SAUVEGARDE LOCI`.
- **Conformance fix**: `loci_save` now pushes a **NUL terminator** before the
  path (read last → terminates the z-string the LOCI `OPEN` firmware pops). The
  emulator tolerated its absence (pre-zeroed XSTACK + boundary stop), but the real
  LOCI firmware would read trailing garbage. Out-of-range forward branches to
  `ls_fail` reworked as inverse-branch + `jmp`. Terminal `.tap` rebuilt clean
  (`client/term.bin`/`term.tap`, `$1000`→`$21FD`).
- **Validated at runtime in `oric1-emu --loci-flash`** with a standalone 6502
  harness (`loci_save` on a known `$4000` buffer + `dlname`): a 256-byte file
  `TEST.BIN` is written to the SD sandbox **byte-for-byte identical** to the
  source (`0..255`), `loci_save` returns `A=1`. Confirms the MIA `OPEN`/
  `WRITE_XSTACK`/`CLOSE` opcodes (`$14`/`$18`/`$15`), flags (`$32`), the reverse
  XSTACK push convention, the 128-byte block loop, and `NAME.EXT` path building.
  Audited against the emulator source (`src/io/loci_fs.c`, `include/io/loci.h`):
  opcodes, flag bits and the write-count convention all match.
  Remaining tracked stage: user-editable name at reception, and **tape** target.

### Added (Download — real filename, save under it, 26/06/2026)
- **The downloaded file is now saved under its real name** instead of the fixed
  `BBSFILE.BIN`. New **download header v2** (after `1F FE`): the 2 block-count
  bytes are followed by the **12-byte Sedoric 8.3 name** (`server/internal/bbs/xfer.go`,
  `sedoricName`, unit-tested). The terminal (`client/term.s`, `handle_rx` state 5)
  reads the 12 bytes into `dlname` and `sed_save` (`client/sedoric.s`) writes the
  Sedoric file under that name. Server + terminal must match (the header grew).
- **Validated end-to-end** in `oric1-emu` (`--loci --serial picowifi`, modem
  `telnet=1`): the terminal reads `nom=ASTERORICTAP`, the binary transfer completes
  (gauge `100%`, "FICHIER RECU EN 4000"). Next stages (tracked): user-editable name
  at reception, and save targets **LOCI SD** (MIA `OPEN/WRITE/CLOSE`) and **tape**.

### Fixed (XMODEM download stuck at 0% — diagnosis, 26/06/2026)
- **Root cause of "download frozen at `0%`" identified and proven**: the
  **picowifi modem in TELNET mode (`telnet=1`)** mangles the binary XMODEM stream.
  A telnet modem reinterprets `0xFF` (IAC) and bare CR; an XMODEM block routinely
  carries `0xFF` (an Oric `.TAP` header has `0xFF` in its very first block) → the
  block checksum fails → endless NAK → the gauge stays at `0%`. The handshake
  (`1F FE` + block count) carries no `0xFF`, which is why the bar is **drawn** but
  never **advances** — exactly the reported screen.
- **Neither the server nor the 6502 receiver is at fault.** Verified with a
  faithful end-to-end bench (real `oascii` + `xmodem` packages driving the real
  `term.tap` in `oric1-emu`, `--loci --serial picowifi`): same 3-block file with
  `0xFF`/`0x0D` in block 1 → `telnet=1` = stuck at `0%` (10× NAK), `telnet=0` =
  full transfer (`SOH→ACK×3→EOT→ACK`, "xmodem.Send OK"). Serial trace shows the
  emulated ACIA `OVERRUN` only when `--serial-buffer` is omitted (separate
  emulator-config footgun).
- **Fix (terminal, in-project): the terminal now issues `ATNET0` at modem init**
  (`client/term.s`, `mm_init`) to force the WiFi modem into **raw mode** before any
  dialing, so binary XMODEM blocks pass through intact. No change to the emulator
  or its config. Harmless (or `ERROR`, ignored) on a modem without `ATNET`; the
  command is echoed like the existing `ATD`. **Validated end-to-end against the
  default `telnet=1`**: with the fix, the same block-1 (`0xFF`/`0x0D`) transfer now
  ACKs on the first try → gauge `[####################] 100%` → "FICHIER RECU EN
  4000" (`--loci --serial picowifi`, modem left at `telnet=1`).
- **Real Pico W hardware**: `ATNET0` is the standard "no telnet" command of WiFi
  modems (Zimodem/WiFi232/PicoWiFiModemUSB); the same fix applies. Separately, if
  the emulator drops bytes during the burst, keep `--serial-buffer 512` (one block
  fits). Documented in `docs/hardware-connection.md` §6.

### Added (Phonebook — IDreamIn8Bits entry, 26/06/2026)
- **New directory entry 6 `IDreamIn8Bits`** in the Oric terminal phonebook
  (`client/term.s`): telnet `bbs.idreamtin8bits.com:6500` (ASCII/ANSI mode),
  protocol 0 (ATD). `NUM_ENTRIES` 5 → 6, prompt updated to `Choix (1-6, M)`.
  The phonebook dials telnet/TLS `host:port` (not HTTP), so the connection
  address was used in place of the `https://www.idreamtin8bits.com/bbs` web URL.
  Build green (`client/build.sh`, `term.tap` 4204 o).

### Changed (Documentation — translated to English, 26/06/2026)
- **Entire project documentation translated from French to English**, in place.
  26 Markdown files (`README`, `CHANGELOG`, `ROADMAP`, `docs/`, ADRs, agile backlog,
  client/studio `README`, deploy notes, terminal-team message). Code blocks, shell
  commands, file paths, memory addresses, version numbers and program-emitted strings
  (Oric screen output, Caddyfile values, Prometheus `# HELP`, `oric1-emu` output) were
  preserved verbatim.
- **French-named files renamed** with internal links updated across the repo and the
  source reference in `server/internal/user/user.go`:
  - `docs/test-emulateurs.md` → `docs/emulator-testing.md`
  - `docs/connexion-materielle.md` → `docs/hardware-connection.md`
  - `docs/etat-de-l-art.md` → `docs/state-of-the-art.md`
  - `docs/transfert.md` → `docs/transfer.md`
  - `docs/revue-client.md` → `docs/client-review.md`
  - `docs/guide-utilisateur.md` → `docs/user-guide.md`
  - `docs/adr/0001-login-composant-page.md` → `docs/adr/0001-login-component-page.md`
  - `docs/adr/0002-modele-de-saisie.md` → `docs/adr/0002-input-model.md`
  - `MESSAGE-equipe-terminal-LOCI.md` → `MESSAGE-terminal-team-LOCI.md`

### Added (XMODEM transfer — progress gauge, 26/06/2026)
- **Progress bar `[####------]  NN%`** during XMODEM transfers,
  on the terminal side (since the channel is raw binary, only the terminal sees the blocks).
  - **Protocol**: the server sends the **total number of blocks** (2 bytes, low/high)
    right after `1F FE` (`server/internal/bbs/xfer.go`). An older terminal
    ignores these 2 bytes (non-SOH); a recent terminal **requires** this total
    (otherwise no gauge). Download/upload: server tests green.
  - **Terminal** (`client/xmodem.s`): `handle_rx` reads the total (states 3/4); the
    bar (BARLEN=20 segments) fills via **Bresenham counting** (no
    16-bit mult/div), percentage = segments×5, displayed on a fixed line (row 25).
    Upload computes the total from `XSIZE`. Gauge variables aliased onto zero-page
    cells **inactive during a transfer** (input/plot/dial); `PLOTST`
    reset after the transfer (the `XACC` alias overwrites it).
  - **Validated** in the emulator (local download): bar at **40%** mid-transfer,
    **100% + "FICHIER RECU EN 4000"** at the end.
  - **Deployed to prod** (`make deploy`) and **verified**: `pavi.3617.fr:6502` indeed sends
    `1F FE AF 00` = **175 blocks** for Astéroric (22396 b / 128). Live protocol.
  - **Release [`v0.1.2-alpha`](https://github.com/benedictemarty/bbsoric/releases/tag/v0.1.2-alpha)**
    with up-to-date `term.tap`/`term-boot.dsk` (v0.1.1 marked obsolete).

### Deployed (Production — server backspace, 26/06/2026)
- **Prod `pavi.3617.fr` updated** (`make deploy`) with backspace handling
  (`Session.ReadLine` handles `$08`/`$7F`). Verified end-to-end: welcome rendered,
  login form processing an input corrected via backspace, service `active`.

### Fixed (Oric terminal — engineering review, 26/06/2026)
Full review of the 6502 client (`docs/client-review.md`). Fixes delivered:
- **LOCI — wrong ACIA base** (`client/term.s`): the "2 = LOCI" option targeted
  **`$03A0`** (the LOCI **MIA** space), not the modem → MIA/ACIA collision, PSG
  disrupted, **keyboard frozen on the directory**. Fixed to **`$0380`** (ACIA of the
  LOCI WiFi modem, cf. firmware `PicoWiFiModemUSB`). **Validated** in the emulator
  `--loci --serial picowifi`: `2`→`1`→`CONNECT pavi.3617.fr` (banner rendered).
  Detail: `phosphoric-findings.md` F1. Docs aligned (`$03A0`→`$0380`).
- **Out-of-bounds plot** (`set_cursor_xy`): clamp `row<28`/`col<40` — removes a
  write outside VRAM driven by untrusted network input (third-party BBS).
- **Unbounded XMODEM reception**: refusal beyond `$B800` (`CAN` + "FICHIER
  TROP GROS") — removes a buffer overflow from the network.
- **Uppercase (SHIFT)**: `scan_shift` + `key_scan` (`a-z`→`A-Z`) — identification
  with a mixed-case password becomes possible. Validated (trace: TX `Y`/`Z`).
- **Backspace**: **DEL** key (col5/row5) → `$08`; `putbyte`/`input_line`
  (client) and `Session.ReadLine` (server, `$08`/`$7F`) erase the last
  character. Test `TestReadLineBackspace` (4 cases).
- `sei` comment + zero-page map documented (`term.s`).
- **Documented deferrals** (with justification): RX flow control (#1), modem/DCD
  codes (#6), telnet IAC (#7), ACIA overrun (#8), Sedoric name (#9), client tests
  (#12). See `docs/client-review.md`.

### Documented (emulator findings, 26/06/2026)
- **`phosphoric-findings.md`** (new): log of defects in the Phosphoric emulator
  spotted from bbsoric. F1 = `--loci` + `--acia-addr 03A0` freezes the
  keyboard (double mapping `$03A0` MIA/ACIA, the MIA masks the ACIA and breaks keyboard
  scanning via the PSG). **The picowifi IS the LOCI's modem** → correct faithful model:
  `--loci --serial picowifi` (without `--acia-addr`; default ACIA `$0380`), the
  terminal having to address **`$0380`** and not `$03A0`. → **to be fixed on the terminal side**
  (`client/term.s`): the menu option "`2` = `$03A0`" should target `$0380`. Guardrail
  Phosphoric ≥ 1.27.3 (message pointing to `$0380`). Troubleshooting added to
  `docs/hardware-connection.md` and to the `run-bbsoric` skill.

### Distributed (GitHub Release — Oric terminal alpha, 26/06/2026)
- **Release `v0.1.0-alpha`** (prerelease) on the public repository:
  <https://github.com/benedictemarty/bbsoric/releases/tag/v0.1.0-alpha>.
  Assets: **`term.tap`** (autorun cassette, 3,668 b) and **`term-boot.dsk`**
  (bootable Sedoric floppy with `TERM.COM`, 1 MB) — both rebuilt
  from the current `term.s` (`make client` + `client/build-disk.sh`). Notes including
  the emulator launch commands, including the **LOCI pitfall** (do not combine
  `--loci` with `--acia-addr 03A0`: double mapping `$03A0` that freezes keyboard
  scanning; use `--serial picowifi --acia-addr 03A0` without `--loci`).

### Deployed (Production — online chat, 25/06/2026)
- **Prod `pavi.3617.fr` updated** via `make deploy` (binary with presence +
  `who`/`chat` applets, `bbsoric` service active, listening on 6502 OK) then push of the
  up-to-date `content/site.json` (remote backup `site.json.bak-20260625-225909`,
  hot-reloaded). **Verified end-to-end on `pavi.3617.fr:6502`**: guest →
  main menu → **Community** → *Who is online* / *Chat* (room open).
  Real-time chat is now usable online.

### Upgraded (Studio "Forge" — who/chat applets)
- **`studio/web/app.js`**: `KNOWN_APPLETS` extended with **`who`** and **`chat`**
  (the "▶ applet" dropdown now offers them without typos);
  added a **descriptive tooltip** per applet (`APPLET_DESC`). The studio
  loads/edits/previews the **Community** menu (screen rendered via
  `internal/render`, identical to the server). Verified: `/api/site` loads the page,
  `/api/screen?page=communaute` renders 192 OASCII bytes (HTTP 200), studio tests
  `-race` green. Doc `studio/README.md` updated (list of known applets).

### Added (Sprint 7 — Who is online + chat between callers)
- **Real-time communication between sessions** (first building block of
  state-of-the-art parity, cf. `docs/state-of-the-art.md` §6):
  - **`server/internal/presence`**: in-memory registry of connected users
    ("who is online") + chat relay **pub/sub with non-blocking broadcast**
    (a slow subscriber never freezes the sender) with **replay** of recent
    messages. Tests: presence, sort by arrival, bounded backlog, non-blocking when buffer
    full, unsubscribe.
  - **`who` applet**: list of callers (handle + presence duration, marker
    "(you)").
  - **`chat` applet**: real-time room. **A single goroutine reads the session**
    (byte-by-byte read with a short deadline, draining messages between
    two keystrokes) — no byte stealing from the engine, local echo, `/q` to quit,
    system messages on arrival/departure, `HH:MM` timestamp.
  - Presence handle set at identification (`Invite-N` for guests,
    account handle otherwise); `SessionState` extended (`Presence`, `MemberID`,
    `Handle`); `WelcomeHandler.Presence` injected from `cmd/bbsd`.
  - **Content**: **Community** menu (key 6 of the main menu) → *Who is
    online* / *Chat*.
- **Tests**: `presence` package + `who`/`chat` integration (two TCP clients,
  message relay verified). Full suite green, **`go test -race` clean**.
  Also validated live (two guests, message relayed with handle + timestamp).

### Documentation (State of the art — functional parity / gaps, 25/06/2026)
- **`docs/state-of-the-art.md` §6**: **functional** comparison of the server to
  the state of the art (ref. petscii-bbs). Lists what exists, then the main gap —
  the **communication spaces between callers** (the "Guestbook" is static,
  not writable) — and prioritizes 6 features: one-liner wall (#2), base/forum messages
  (#1), who-is-online + chat (#3), private messaging (#4),
  RSS→OASCII news (#5), door game (#6).
- **`ROADMAP.md`**: new **Sprint 7 — Communication between callers** picking up
  this backlog (each feature = a `bbs.Register` applet + persisted store).

### Communication (Alpha announcement published — Defence Force, 25/06/2026)
- **Public announcement of the alpha version** (server + Oric terminal + studio
  "Forge") on the **Defence Force forum**:
  <https://forum.defence-force.org/viewtopic.php?t=2897>. Demo video:
  <https://youtu.be/YRFBYkpsKMc>. Source text: `~/bbsoric-announce-defence-force.md`
  / `.bbcode.txt`. Full record: **`docs/communication.md`**.
- **GitHub repository made public**: <https://github.com/benedictemarty/bbsoric>
  (history rewritten beforehand via `git filter-repo` to purge internal
  IPs; placeholders in `deploy/caddy-tls.md`).
- The announcement launches a **call for testing on real hardware** (terminal rendering,
  serial XMODEM timing, Sedoric write on a physical drive) — feedback to be recorded in
  `docs/communication.md`.

### Fixed (XMODEM terminal — fast startup, no more freezing on network jitter)
- **`client/xmodem.s`**: block reception used `xr_rx` **blocking
  (without timeout)** → if a byte of a block was late (TCP segmentation/jitter to
  prod), the terminal **froze indefinitely** instead of re-NAKing, causing a
  download startup of ~43 s. Replaced with **`xr_rx_t`** (timeout ~1.3 s,
  preserves Y); on a missing byte, `bcc xr_start` → **fast re-NAK** (the
  server resends the block). Measured on prod: startup **~43 s → ~2 s**,
  complete file received (174 ACK). The transfer throughput remains bounded by the
  network (XMODEM stop-and-wait, 1 round trip per block).

### Validated (XMODEM download client↔prod — end-to-end + startup diagnosis)
- **End-to-end download PROVEN**: an emulated Oric terminal connected to **prod**
  (`pavi.3617.fr` via modem backend) downloads `ASTERORIC.TAP` (22 KB) up to
  "FICHIER RECU EN 4000 / Download complete" (reception in RAM `$4000`).
  Made possible by Phosphoric's `--realtime` pacing (otherwise non-deterministic
  timing). Video: `~/bbsoric-client-prod-demo.mp4`.
- **Diagnosis (serial trace)**: the `1F FE` (RecvCmd) is indeed received and
  `handle_rx` switches to `xmodem_recv`, but **startup is slow** — the
  receiver re-emits the NAK 2-3 times with a **long timeout (~32 s)** before
  synchronizing and ACKing the blocks. Startup race (order RecvCmd→1st block
  vs receiver's NAK). **Optimization lead**: shorten the re-NAK timeout of
  `xmodem_recv` (`client/xmodem.s`) and/or guarantee the flush of
  `RecvCmd` before `waitStart` on the server side. The transfer itself is intact.

### Deployed (Production — full alignment, 25/06/2026)
- **Prod `pavi.3617.fr` (LXC pavi3617) re-aligned on the repo** via `make deploy`
  (up-to-date binary + service with `-files`/`-users`/`-metrics-addr` + backup/monitoring
  timers) then push of the current `content/site.json`
  (prior backup `site.json.bak-20260625-111109`).
- **Verified end-to-end** on `pavi.3617.fr:6502`: welcome → guest → main
  menu → **Files** → **Download** applet operational (`-files`
  active, empty library). Prod now exposes the same functional level
  as the repo (server = studio = client).

### Added (Tooling — "run-bbsoric" skill)
- **`.claude/skills/run-bbsoric/`**: launch/drive skill. `SKILL.md`
  (build, run, test, studio, terminal) + **`driver.py`**: harness that drives the
  BBS server over a TCP socket (sending keys, reading/rendering OASCII, smoke flow banner
  + menu navigation, captures `/tmp/bbs-*.txt`). Verified end-to-end
  (`make build` → `./bbsd` → driver `exit 0`, 4 screens validated).

### ✅ Sedoric — floppy save VALIDATED end-to-end on SEDORIC V3.0
- **Sedoric save proven from machine language**: a file
  (`TESTML  BIN`) is **written and persisted** in `sedoric3.dsk` (catalog entry
  + write-back, md5 changed) by the ML sequence, tested in the emulator.
- **V3.0 recipe** (disassembled SEDORIC 3.0 manual, APPENDIX 15) — `JSR $04F2`
  (ROM→overlay switch) → set BUFNOM/VSALO0/FTYPE/DESALO/FISALO/LGSALO/EXSALO
  → `JSR $DE9C` (XSAVEB) → `JSR $04F2`. The overlay switch **changes per
  version**: `$04F2` in V3.0, `$0472` in 1.x/2.x. Confirmed first by the example
  "HELLO ANDRE" of APPENDIX 15, then by XSAVEB.
- **Public vectors confirmed identical V1.0/V3.0** (CPU-view dump during SAVE):
  `$FF7C = JMP $DE9C` (XSAVEB), `$FF76 = JMP $DE28` (XDEFSA). `$DE9C` starts with
  `SEI $78` (used to detect "Sedoric resident").
- **`client/sedoric.s` finalized**: `OVL_TOGGLE = $04F2`, `XSAVEB = $DE9C`,
  variables at the documented addresses (`$C04D`/`$C051`/`$C052`/`$C054`…), detection
  `$78`. Assembles (`build.sh` green). Two PDFs ("Sedoric à nu" V1.0 + disassembled
  V3.0 manual) provided by the user were used.
- **Sedoric presence guard (safe without a disk)**: `sed_save` first checks,
  in always-mapped RAM page 4, the jump table installed by Sedoric
  (`$04F2`/`$04F5` = `4C xx 04`) **before** any `JSR $04F2`. Validated: under Sedoric
  the guard passes and the file is saved (`TESTG4 BIN`); without a disk `$04F2=$55`
  → guard refuses, no crash. The same terminal is therefore safe on cassette and
  under Sedoric.
- **Integration already wired**: `term.s` (`handle_rx`) calls `sed_save` after a
  download, `XSIZE` set by the XMODEM receiver.
- **✅ Bootable terminal floppy**: `client/build-disk.sh` (reproducible)
  produces `term-boot.dsk` = Sedoric master floppy + **TERM.COM** (terminal
  injected into RAM via fast-load tape then Sedoric `SAVE`). The terminal **runs**
  from the floppy (`LOAD"TERM":CALL#1000` → modem menu displayed, ~2.6 M
  instructions executed). The initial `BREAK` came from the `,J` option (resolved:
  `LOAD`+`CALL`). The ACIA `$03A0` (LOCI) is a runtime choice (menu) to coexist
  with the Microdisc — no build variant. Hands-free auto-start =
  refinement (replace the boot program of the master). See `docs/sedoric-api.md`.
- *Tooling detail*: xa65 splits comments on ":" (comments without
  colons); `--type-keys` sometimes loses the 1st character of a line.

### Added (Content — Files submenu: download/upload accessible)
- **`content/site.json`**: **"Files"** entry (key `5`) in the main menu
  → **`fichiers`** page with **Download** (applet `download`) and **Upload**
  (applet `upload`), `next: fichiers` (back to the submenu), plus `Back`.
  The XMODEM applets (already coded/tested) are finally **reachable from the UI**
  (they were registered but wired nowhere). JSON validated, targets
  consistent, `content`/`bbs` tests green.

### Added (Infrastructure — state backup & restore)
- **`scripts/backup.sh`**: timestamped `tar.gz` archive of the persistent state
  (accounts `users.json`, library `files/`, content `site.json`) into
  `/var/backups/bbsoric/`, with **rotation** (14 by default) and a **manifest**.
  **"Hot" backup** (atomic server writes → no shutdown required).
- **`scripts/restore.sh`**: restore of an archive (`<file>`, `latest`
  or `--list`) — stop service → set aside `*.pre-restore` → restore
  → restart (systemd re-takes ownership of the `StateDirectory` under `DynamicUser`).
- **`deploy/bbsoric-backup.{service,timer}`**: **daily** backup
  (03:30, `Persistent=true`), hardened (`ReadWritePaths` to the backups folder only).
- **`deploy/vps-deploy.sh`**: installs backup/restore scripts + enables the timer.
- **`scripts/test-backup.sh`**: end-to-end test (13 cases) — backup,
  archive content, restore after corruption, `latest`, rotation. **Green.**
- **`docs/backup.md`**: complete procedure (target, mechanism, restore,
  `DynamicUser` note, off-site).

### Investigated (Sedoric — full reverse of the SAVE dispatch)
- **Reverse map established** (save-state at the prompt + CPU trace + watchpoint
  `memory_set_trace`): command buffer **`$0035`**, **self-modifying scanner
  `$00E2`–`$00ED`** (operand of `LDA $00E8` advanced via `$E9/$EA`), keyword
  table **`$CA6F`** (match via `$DE/$DF`, separator `$22`), compare helper
  `$D5B5`, save cluster `$D33x`–`$D39x`, FDC primitive `$D075`, page 4
  trampolines (`$04EF`→`$C4A0`).
- **Decisive conclusion**: `SAVE` is **dispatched by the BASIC ROM**
  (`$F6xx`–`$F8xx`) then the Sedoric scanner — `$C4A0` is only executed once
  while idle, not on the SAVE path. The dispatch depends on many zero-page
  variables → **no trivially isolable ML entry point**; calling `SAVE` from standalone
  code is not a simple `JSR`.
- **Chosen path**: a **documented** mechanism to execute a Sedoric command
  from ML (to be obtained via "Sedoric à nu"/manual) — the only portable path for
  real hardware; alternative: keyboard injection (type-ahead).
- **Deployment**: `tap2sedoric` (oric1-emu) is a **stub** → no direct `.dsk`;
  realistic path = **`CLOAD` of the terminal under resident Sedoric**.
- **`client/sedoric.s`**: code by vectors `$FF7x` marked **superseded**
  (safe no-op guard kept). **`docs/sedoric-api.md`**: map + recommended
  approaches + deployment.

### Investigated (Microdisc/Sedoric storage — floppy write PROVEN)
- **Root cause of the "blockage" identified**: it was neither the addresses of
  the Sedoric API nor the ROMDIS mapping, but the emulator flag **`--disk-writeback`**
  (opt-in write-back, disabled by default). Without it, `SAVE` writes the image
  **in memory** but nothing is persisted to the host `.dsk`.
- **Write chain validated end-to-end** in `oric1-emu`: boot **Sedoric V3.0**
  resident (`-r basic11b.rom --disk-rom microdis.rom -d sedoric3.dsk`), binary `SAVE`
  from the prompt → real file written (catalog entry `TEST     BIN`,
  data + bitmap), persisted with `--disk-writeback` (`.dsk` md5 changed).
  FDC sector-write primitive at `$D075` (Type II cmds `$A8`/`$AC`).
- **`microdis.rom` = `Oric DOS V0.6`**: page `$FF` empty → the PDF vectors
  (`$FF73`…) are not there; the Sedoric API is installed in RAM overlay at boot.
- **`docs/sedoric-api.md`**: "Floppy write VALIDATED" section (root
  cause, reproducible recipe, consequences). **`client/sedoric.s`**: status
  fixed (the call via PDF vectors still needs realigning via the `SAVE` trace).
- *Remaining (G1, path B)*: trace the **machine call entry point** of the `SAVE` to
  reproduce it from the terminal, and run the terminal **under resident
  Sedoric** (`.dsk` bootable).

### In progress (Microdisc/Sedoric storage — path B, exploration)
- **`docs/sedoric-api.md`**: Sedoric API extracted from the disassembly (vectors
  `$FF73`/`$FF76`/`$FF79`/`$FF7C`, variables `BUFNOM`/`DESALO`/`FISALO`) +
  SAVE/LOAD sequences.
- **`client/sedoric.s`**: `sed_save` (saves `$4000` to a file via the API),
  **safe detection** (does not crash without Sedoric); `handle_rx` calls
  `sed_save` after a download. Assembled.
- **Discovery (emulator tests)**: the **Microdisc ROM mapping** masks the
  page `$FF` vectors, and the PDF addresses do not match `sedoric3.dsk`
  → the call is not operational as is. Realigning the addresses to the target
  version + ROMDIS handling needed (a reverse sub-project, real-hardware
  validation recommended). Sedoric boots fine in the emulator. Backlog **G1**.

### Added (Oric terminal — XMODEM file send, upload)
- **`client/xmodem.s`**: **6502 XMODEM** transmitter in **CRC-16** (`xmodem_send` +
  `crc_update`) — sends `XSIZE` bytes from the RAM buffer (`$4000`), retransmission
  on NAK/timeout, EOT. The CRC avoids the switch delay on the receiver side (the server
  starts in CRC).
- **`client/term.s`**: `handle_rx` detects **`1F FD`** (`oascii.SendCmd`, emitted by
  the `upload` applet) and starts `xmodem_send`.
- **Validated in the emulator**: an Oric uploads 256 bytes, received intact and
  stored on the server side — `docs/img/xmodem-upload.png` ("FICHIER ENVOYE" /
  "Recu : f (256 octets)"). **Bidirectional** Oric ↔ server transfer complete.
- *Remaining*: **storage** on mass memory (SD card via LOCI / Microdisc /
  cassette) — today the buffer is in RAM `$4000` (backlog G1).

### Added (Oric terminal — XMODEM file receive, download)
- **`client/xmodem.s`**: **6502 XMODEM** receiver (checksum mode), receives
  a file in RAM (`$4000`), ACK/NAK, EOT. `xr_rx` preserves Y (which `ser_rx`
  overwrites) — loop bug fixed.
- **`client/term.s`**: `handle_rx` detects the **`1F FE`** sequence sent by the
  server and switches to receive mode (`xmodem_recv`); `build.sh` integrates `xmodem.s`.
- **Signaling**: `oascii.RecvCmd()` (`1F FE`) / `SendCmd()` (`1F FD`);
  the `download` applet emits `RecvCmd` before the XMODEM send.
- **Validated in the emulator**: an Oric downloads a file (128 b) from the server,
  received intact in RAM — `docs/img/xmodem-download.png` ("FICHIER RECU EN 4000").
- *Remaining*: 6502 upload (transmitter), SD card (LOCI)/Microdisc/cassette storage
  (today reception in RAM only) — backlog G1.

### Added (File transfer — XMODEM download/upload, server side)
- **`internal/xmodem`**: XMODEM protocol (128 b blocks, checksum **and**
  CRC-16, retransmission, trimming of `SUB` padding). Round-trip tests (checksum +
  CRC) via `net.Pipe`.
- **`server/internal/files`**: on-disk file library (list,
  read, atomic write), validated names (anti path-traversal), max size.
- **`server.Session.Raw()`**: raw byte channel for binary transfers
  (bypasses telnet/line filtering) + `ClearDeadline()`.
- **`download`/`upload` applets** (`server/internal/bbs/xfer.go`): download lists
  the library and **sends** a file via XMODEM; upload **receives** and
  saves. Injected via `AppContext.Files` / `WelcomeHandler.Files`. End-to-end
  tests (`TestDownloadApplet`, `TestUploadApplet`).
- **`bbsd`**: flags `-files <dir>` and `-max-upload <bytes>`; `bbsoric.service`
  uses `/var/lib/bbsoric/files`. Studio: `download`/`upload` in the applet
  selector. Doc: `docs/transfer.md`.
- *Remaining on the Oric side*: transfer mode + 6502 XMODEM + SD/floppy storage
  in `client/term.s` (backlog G1).

### Added (Rendering — automatic wrapping of lines > 40 columns)
- **`internal/render`**: a text line exceeding **40 columns** is
  now **wrapped** onto the next line (break at spaces; hard hyphenation
  for a word longer than a line) instead of being truncated by the terminal.
  On the line break, the **current attributes (ink/paper/…) are re-emitted**
  to keep the same rendering (the ULA resets them at each line start).
  Concerns only "logical" pages (`writeLine`/`Screen`); the "raw screen"
  (`RawScreen`) is still emitted as is. Test `TestWrapWidthAndColor`.

### Added (Applets — retry + failure page)
- **In-place retry**: the `form` applet re-asks the fields on failure
  until success or exhaustion of attempts (`Form.Retries`, default 3).
  Cancellation (1st field empty) remains a deliberate return, not a failure.
- **Configurable failure page**: new `Outcome.Failed`; on definitive failure,
  the engine routes to **`Form.Fail`** (form page) or **`Entry.Fail`** (entry
  ▶ applet) if defined, otherwise goes back / stays on the menu. The `login`/`register`
  applets also signal `Failed` after "Too many attempts".
- **Validation**: `Form.Fail` / `Entry.Fail` must reference an existing page.
- **Studio**: `formEditor` exposes "On failure" (page) + "Attempts";
  the ▶ applet entry has a "page on failure" selector (next to success).
- Tests `TestFormFailToPage`, `TestFormRetryThenSuccess`.
- **Content** (`content/site.json`): the `login`/`register` pages route to a
  dedicated **`echec`** page (`fail: echec`) after attempts are exhausted.

### Changed (Studio — form editable from the Screen tab)
- **"Screen" tab**: the block under the grid now also handles the
  **input form** (applet `form`), not just menu entries.
  A `form` page (e.g. `login`) loaded into the screen editor shows its
  `formEditor` (action, fields, **X/Y positions**, next); a menu page keeps
  its entries editor + a "+ form" button. So one can compose a
  **full-screen login** from a single place: decor in the grid + positioned
  fields. `formEditor` made reusable (refresh callback).

### Changed (Studio — applet insertion via dropdown)
- **Entries editor** (Edit *and* Screen tabs): for a
  "▶ applet" entry, the name is now chosen from a **dropdown**
  (`login`/`register`/`guest`, + the current value if custom) instead
  of a free text field — no more typos. `appletSelect` /
  `KNOWN_APPLETS`.

### Changed (Studio — composer removed, Screen navigation more visible)
- **"Edit" tab**: removal of the **line composer** (canvas + palette
  `glyph-palette` + `comp-*` buttons), redundant with the cell-by-cell screen
  editor. Associated code/HTML/CSS removed (`comp`, `drawComp`, `compAdd`,
  `compInsert`, `renderPalette`).
- **"Screen" tab**: the **menu navigation** editor is now
  **discoverable** — shown as soon as the tab opens (with a prompt message
  when no page is loaded), instead of appearing only after a page is
  loaded.

### Added ("Smart" screen buffer — differential rendering)
- **`internal/oascii.Screen`**: a 40×28 buffer that maintains both the composed state AND
  the state displayed by the terminal. `Render()` emits ONLY the changed cells,
  grouped into segments (positioned by plot X,Y), without crossing line
  ends. Exact on Oric (the screen IS the VRAM: each cell is independent,
  the ULA recomposes the line at scan time). Saves the 9600 baud serial link
  for dynamic screens (games, refreshed values) — re-emitting everything costs
  ~1.2 s, a diff of a few cells is nearly instant.
- API: `NewScreen`, `Put`/`PutText`/`Clear`/`At`/`Buffer`, `Render` (diff +
  memorization), `Reset` (forces a full re-emission). The diff even skips the
  common cells at the head of a change ("000"→"042" emits only "42").
  Tests `TestScreen*`.

### Added (Example — full-screen login page + emulator capture)
- **`docs/examples/example-login.json`**: a **full-screen** login page combining a
  40×28 **`raw` decor** (frame, colored titles, "Pseudo"/"Mot de passe" labels)
  and a **`form`** whose fields are **positioned** (`at:[20,11]`, `[20,14]`) via
  plot X,Y. The `form` applet displays a full-screen raw decor from (0,0)
  (`server/internal/bbs/form.go`).
- **Emulator capture** `docs/img/example-login-plein-ecran.png`: real rendering on
  oric1-emu (ULA) — decor + login field prompt placed at its coordinates.

### Added (Cursor positioning — plot X,Y)
- **Oric terminal** (`client/term.s`): state machine on the RX stream — the
  **`1F col row`** sequence repositions the VRAM write cursor
  (`handle_rx`/`set_cursor_xy`, `SCRPTR = $BB80 + row*40 + col`). Assembled (xa).
- **`internal/oascii`**: constant `PlotByte` (0x1F), `Plot(col, row)` and
  `Builder.At(col, row)`; test `TestPlot`.
- **Positioned fields**: `content.Field.At [col,row]` (validated: length 2 and
  within the 40×28 screen). The `form` applet emits the positioning sequence before
  the field prompt; otherwise sequential display. Test `TestFormFieldPlot`.
- **Studio**: **X / Y** columns per field in the form editor (empty =
  sequential prompt). Doc: `docs/oascii.md` (positioning section).

### Changed (Content — login AND registration default to "form" pages)
- `content/site.json`: the welcome no longer launches the `login`/`register` applets
  directly; entries 1 and 2 target **dedicated `form`-type pages** —
  `login` (action `login`, handle/password) and `register` (action `register`,
  handle/password/confirm), `next: main`. Demonstrates the declarative model on
  production content. Validated end-to-end (account creation → account persisted
  with PBKDF2 hash; login → personalized welcome).

### Added (Declarative input pages — "form" applet)
- **Model** (`internal/content`): type **`Form`** (`action`, `fields`, `next`) +
  **`Field`** (`key`, `label`, `secret`) on the page. `Validate` checks the action
  (`login`/`register`), the required fields (`login`+`password`, plus `confirm` for
  registration) and the existence of `next`.
- **Engine** (`server/internal/bbs`): generic **`form`** applet (`form.go`) —
  displays the decor (composed raw buffer OR title banner), captures the declared
  fields, then executes the action on the server side (authentication / account
  creation, PBKDF2 hashing unchanged). `runFormPage` routes to `Form.Next`. A single
  declarative applet replaces writing Go per input screen; the historical `login`/
  `register`/`guest` remain (compat). Tests `TestFormPageLogin` /
  `TestFormPageRegister`.
- **Studio**: **form** editor in the "Edit" tab (`formEditor`) —
  action, list of fields (key/label/secret), `next`; auto-add of the
  `confirm` field in registration mode. A form page does not show a menu
  editor (the form drives the page).

### Changed (Studio — raw navigation: "label" column hidden)
- **"Screen" tab, Navigation block**: the **"Label"** column is removed.
  On a "menu over a background screen" (raw page), the label is **drawn in the
  decor** and `e.label` is ignored at render time (`RawScreen`) — only the
  key → target/applet mapping matters. `entriesEditor` receives a `hideLabel` option (the column
  remains shown in the "Edit" tab for normal menus).

### Changed (Studio — glyph drop: auto alternate charset)
- **"Screen" tab**: clicking a BBS glyph now **drops** it directly
  at the cursor (instead of only loading the brush) and **sets the alternate
  charset attribute (0x09) if it is not already active** at that position — a
  glyph is only rendered in the BBS font if alt is active. `altActiveAt` computes
  the state by serialization from the line start; `dropGlyph` only adds the
  attribute cell if necessary (no duplicate if alt is already set).

### Changed (Studio — glyph palette to the right of the screen)
- **"Screen" tab**: the BBS glyph palette moves **below** the canvas to
  the **right** of it (flex container `.screen-edit`: canvas + `.palette-side`).
  No more vertical scroll to reach the glyphs; responsive layout (the
  palette moves back below on a narrow screen).

### Added (Menu over a background screen — combined raw + entries)
- **Engine** (`server/internal/bbs/engine.go`): a "raw screen" (`raw`) page
  now **combines** with `entries`. The 40×28 buffer serves as the **background
  screen** (cell-by-cell composed decor, menu labels drawn into it) and
  the `entries` provide the **navigation** (key → page, or ▶ applet) — without
  an added "Your choice" prompt. Presentation (Screen) and logic (Entries)
  separated. The `switch` handles entries first, the rendering follows `raw`.
  Test `TestRawScreenMenu` (raw background + navigation, without prompt).
- **Model** (`internal/content`): documentation of the `Raw`+`Entries` mix;
  `Validate` already accepted it.
- **Studio, "Screen" tab**: **navigation** editor under the grid
  (`renderScreenNav` + `#screen-nav`) — compose the decor above and wire
  the keys just below. `entriesEditor` made reusable (refresh callback).
  Saving preserves the entries ("raw + menu" status).

### Fixed (Studio — screen editor: empty page selector)
- **"Screen" tab**: the page selector remained **empty** when the site
  contained no "raw screen" (`raw`) page, even though `screenLoad` already knows how
  to load **any** page (server render → editable buffer).
  `refreshScreenPages` now lists **all** pages (suffix "(screen)"
  for those already in raw mode). Loading a normal page then saving it
  converts it to a raw screen.

### Fixed (Studio — screen editor: setting color attributes)
- **"Screen" tab**: clicking an **ink/paper** swatch (or a text attribute
  button alt/cli/norm) only changed the brush **without setting anything** —
  the color seemed to "not apply" and the click **stole the
  focus** from the canvas (keystrokes inoperative afterwards). Now the click
  **sets the attribute cell** at the cursor position (an attribute OCCUPIES a
  cell on Oric: the colored "space"), **advances the cursor**, and **returns the
  focus** to the canvas to keep typing (`pickAttr`/`putByteAdvance` in
  `studio/web/app.js`). The brush stays set to the chosen value (click-painting
  still possible).

### Added (Sprint 5 — Docker containerization)
- **`Dockerfile`** multi-stage: build `golang:1.26-alpine` (static binary
  `CGO_ENABLED=0`, `-trimpath -ldflags='-s -w'`) → runtime `alpine:3.20`
  **non-root** (uid 10001), default `site.json` embedded. Image **~18 MB**,
  **`HEALTHCHECK`** on `/healthz`. No external dependency (stdlib only).
- **`docker-compose.yml`**: `bbsoric` service (port 6502, `restart:
  unless-stopped`, volume `bbsoric-state` for accounts, optional `site.json`
  mount). **`.dockerignore`** (minimal build context).
- **Makefile**: targets `docker-build`, `docker-up`, `docker-down`.
- **Doc**: `docs/docker.md` (image, startup, config, TLS, security).
- Validated: `docker build` OK, container started, BBS responds on 6502 (ASCII-art
  banner), healthcheck `ok`. Sprint 5 **finished** (prod stays on systemd).

### Added (Sprint 5 — Monitoring/alerting + user doc)
- **HTTP supervision endpoint** (`server/internal/server/metrics.go`):
  `GET /healthz` (liveness probe "ok") and `GET /metrics` (Prometheus
  text format). Enabled by the **`-metrics-addr`** flag (empty = disabled;
  **local-only** in prod, e.g. `127.0.0.1:6510` — never exposed to the Internet).
  Graceful shutdown on SIGINT/SIGTERM.
- **Metrics** (`server/internal/server/server.go`): atomic counters and
  `Server.Stats()` — `bbsoric_uptime_seconds`, `bbsoric_connections_total`,
  `bbsoric_connections_active`, `bbsoric_connections_rejected_total`. Tests
  `TestHealthz` / `TestMetricsReflectsCounters`.
- **Probe + alerting**: `scripts/monitor.sh` (tests `/healthz` then the
  telnet port via `/dev/tcp`, alerts by email if down) triggered by
  `deploy/bbsoric-monitor.timer` → `deploy/bbsoric-monitor.service` (oneshot,
  every 5 min). `bbsoric.service` adds `-metrics-addr 127.0.0.1:6510`;
  `vps-deploy.sh` installs and enables the supervision automatically.
- **Docs**: `docs/monitoring.md` (supervision layers, endpoint, probe,
  watchdog/Prometheus leads) and `docs/user-guide.md` (general-public connection
  from a real Oric and from a PC, navigation, accounts, troubleshooting).

### Added (Sprint 4 — Real hardware connection)
- **`docs/hardware-connection.md`**: complete guide to reach the BBS from a
  **real Oric**. Hardware chain Oric→ACIA→WiFi modem→TCP; addressing **ACIA
  `$031C`** (standard) and **LOCI `$03A0-$03BF`** with a table of the 6551
  registers; WiFi modem (Hayes / picowifi v0.2.0 firmware), settings **9600 8N1**; AT
  commands emitted by the client (`ATD`, `ATDT#` TLS, `AT$CA`/`AT$CV1`, `ATGET`);
  step-by-step procedure (`CLOAD"TERM"` → menu → dialing), troubleshooting table.
- **Hardware acceptance procedure** (checklist **T1–T9**, §7 of the same doc):
  loading, ACIA backend, directory/CONNECT, color banner, keyboard
  navigation, manual entry, TLS, disconnection, stability. *Physical test pending
  hardware* — the pipeline is validated in the emulator.
- **"ORIC" ASCII-art welcome screen** (`server/internal/bbs/welcome.go`):
  banner enriched with a 5-line art assembled from 5×5 glyphs (`buildOricArt`),
  centered and **OASCII-compliant** (width ≤ 40 columns, 1 attribute byte/line),
  colors yellow (art) / cyan (subtitle `B B S   O R I C`). App version
  bumped to "Sprint 4". Server tests green.

### Added (Full 40×28 screen editor + "raw screen" page)
- **"Raw screen" page**: `raw` field + **`screen`** buffer (40×28 bytes, base64) in
  `internal/content` — rendered **as is** without a title bar or prompt
  (`internal/render.RawScreen`/`screenRows`, no final line break). Falls back to `lines`
  if no buffer. Engine: one key to exit.
- **Studio, "Screen" tab**: **character-by-character** editor on the 40×28 grid,
  **faithful to the ULA** — it works on the **byte screen buffer** where **attributes are
  cells** that you set explicitly (ink/paper/text), exactly like the Oric (an
  ink change occupies a cell and applies until the next attribute). No more
  "per-cell" coloring inconsistent with serialization; inverse remains per
  character (bit 7). Brush = byte to set (character + inverse, or attribute via ink/paper
  swatches + alt/cli/norm buttons); click = paint, keyboard = type (arrows/cursor,
  ⌫/Delete); BBS palette for the character. Create/Load/Save (buffer ↔ base64).
- `/api/screen` renders `raw` pages via `RawScreen`; shared ULA preview rendering
  (`renderScreenBuf`, reused by the page preview). `render` tests; validated server + studio.
- Edit tab: a **composer** assembles a line character by character mixing
  **normal text** (the "+ text" field) and **BBS glyphs** (click in the palette), with
  a **live ULA preview**. "Insert as line" adds the line to the current page,
  **grouped into segments** by mode (alt vs normal). Replaces glyph-by-glyph insertion
  in the focused field.

### Added ("BBS Oric" font — redefined alternate charset, BBS art)
- Since the Oric has no BBS-oriented glyphs (unlike PETSCII/ATASCII), we
  **redefine its alternate charset**: a new **6×8 BBS font** (single and double
  rules/frames, blocks ▌▐▀▄█, shades ░▒▓, symbols ►◄▲▼★•✓…), 35 glyphs.
- `tools/genfont`: generator (glyphs described in ASCII-art, **single source**) producing
  `studio/web/altcharset.js` (simulator) and `client/altcharset.s` (data for `$B800`).
  Target `make genfont`.
- Studio: the ULA simulator renders `altCharset` cells with the BBS font (the rules
  connect), and a **palette** (Edit tab) inserts glyphs into the current field.
- Access via `altCharset: true` (line or segment). Rendering validated (frame `┌───┐`).
- **Oric terminal** (`client/term.s`): `load_altcharset` copies the font into `$B800` at
  startup; `client/build.sh` concatenates `term.s` + `altcharset.s`. **Validated in the
  emulator**: an Oric displays a frame drawn in the BBS font (real ULA rendering).

### Added (Studio — faithful "ULA simulator" preview + shared rendering)
- **`internal/render`**: a **shared** package producing the OASCII stream of a page screen
  (`Screen`) — **single source** reused by the server (`server/internal/bbs`) and the
  studio; removes server/preview duplication.
- Studio: the approximate HTML preview is replaced by a JS/canvas **ULA simulator**
  reproducing `oric1-emu/src/video/video.c` (ink/paper attributes, double height,
  blink, inverse, approximated alt charset) — **without ROM or emulator at runtime**:
  the **standard Oric font** is extracted once from the ROM (offset `0x3C78`, 96×8) and
  embedded (`studio/web/charset.js`). Endpoint `GET/POST /api/screen` → OASCII bytes;
  client rendering (`240×224`, pixelated scaling).
- **Fixed**: **inverse** is now **per character (bit 7)**, compliant with the ULA
  (`InverseText`), and not an erroneous serial attribute (byte 29 actually set the video
  mode). `oascii.InverseAttr`/`Builder.Inverse` removed.
- Tests: `render` (menu/content/segments/inverse bit 7), `oascii` (InverseText). Font
  rendering validated (ASCII-art "BIENVENUE"). Engine refactored on `render.Screen` (suite green).

### Added (Content — full Oric style + multicolor per segment)
- `internal/content`: a **`Style`** (ink, paper, **blink**, **double height**,
  **alternate charset/semi-graphics**, **inverse**) carried by a line **and** by each
  **`Span`**; a `Line` can be plain text or a sequence of styled `segments` →
  **several colors/attributes on a single line**.
- `internal/oascii`: `InverseAttr`/`Builder.Inverse` (inverse video); `Builder.Attrs`
  (blink/double height/alt charset in one byte).
- `server/internal/bbs`: rendering by **style delta** (`writeLine`/`emitStyle`) — emits only
  the attribute changes along the line (saving screen cells).
- Studio: line editor **by card** with ink/paper controls + C/H/A/I toggles and
  **splitting into segments**; the preview renders paper, blink, double height, inverse
  (swap ink/paper) and approximates semi-graphics.
- Docs `content.md`. Tests: oascii (Inverse), preview (segments/inverse/alt), engine
  (multicolor). Teletext bytes verified (hexdump).

### Fixed (Login: "Log in" returned to the menu via nc/line clients)
- A line-mode client (nc…) sends "1\r\n": the menu read `1` (single key) but
  the residual `\r\n` was read as an **empty line** by the login applet's first `ReadLine`
  → immediate cancellation → return to the menu. `ReadKey` **now drains the CR/LF/NUL
  already buffered** behind the key (without blocking). No effect on an Oric terminal in
  character mode (no residue). Dedicated `ReadKey` test; registration/login flow re-checked via `nc`.

### Added (Studio — deployment profile editing)
- The Configuration tab allows **editing profiles** (LOCAL, HOST, USER, PORT,
  CONTENT_PATH, SERVICE, RELOAD) and **saving** them into
  `deploy/profiles/<site>/<env>.conf` — no need to edit the `.conf` by hand.
- `studio/internal/deploy`: `Profile.Marshal` (`.conf` format) + `SaveProfile` (atomic
  write, anti-traversal) + JSON tags. `studio/cmd/forge`: `GET`/`POST /api/profile`.
- UI: profile selector → form (fields hidden depending on LOCAL) → "Save the
  profile"; Deploy block (Simulate/Deploy) below.
- Tests: `SaveProfile` round-trip + traversal refusal (site/env). Verified via `curl`.

### Changed (Content — merge menu/page into a single page type)
- Removal of the `type` field: a **page** has a title and, optionally, **text**
  (`lines`) **and/or** **choices** (`entries`). With `entries` → interactive screen (the text
  shows above the choices); without `entries` → content screen. Allows **text + choices
  on the same screen** (impossible before).
- `internal/content`: `Page` without `Type`; simplified validation. `server/internal/bbs` and
  `studio/internal/preview`: rendering/navigation based on the presence of `entries`/`applet`.
- `content/site.json`: `type` fields removed (the `type` remains ignored if it lingers in old
  JSON — read compat).
- Studio: no more type selector or `+ menu`/`+ page`/`+ applet` buttons; a single
  **"+ page"**, the form edits text **and** choices. The graph derives the label
  (menu/page/applet) from the structure.
- Tests: menu page with intro text, validation, parsing. Verified via `nc`.

### Added (Content — applet entries: a menu can offer several applets)
- `internal/content`: a menu `Entry` can now carry `applet` (+ `next`) **instead
  of** `target` — a (menu) page can therefore **contain several applets**, presented
  as choices. Validation adapted (target **or** applet required).
- `server/internal/bbs/engine.go`: an applet entry launches the applet via the registry then,
  on success, navigates to `next` (otherwise stays on the menu). `runApplet` factored out.
- `content/site.json`: the auth gate uses **applet entries** (`login`/`register`/
  `guest` directly on the `accueil` menu); separate applet pages removed.
- Studio: the entries editor offers the type **→ page** or **▶ applet** (name + `next`);
  the navigation graph links applet entries to their `next` and shows `▶applet`.
  The studio no longer creates an applet-type *page* ("+ applet" button and type option
  removed) — applets are launched via a menu entry. The engine keeps compat for
  hand-written applet pages.
- Tests: applet entry (launch + `next` navigation), validation. Validated via `nc`.

### Changed (Studio Forge — PER-SITE profiles + indented saving)
- Deployment profiles are now **specific to each site**:
  `deploy/profiles/<site>/<env>.conf` (each site has its `dev`/`int`/`prod` trio), instead of
  a global set. API: `GET /api/profiles?site=`, `POST /api/deploy?site=&profile=&dryRun=`.
  Examples moved under `deploy/profiles/site/`; `.gitignore` covers `deploy/profiles/**/*.conf`.
- Saving **re-indents** the JSON (`json.Indent`, 2 spaces): readable files,
  stable git diffs, all keys preserved (including `_comment`).
- Tests: `LoadSiteProfiles` (per site, missing directory tolerated, traversal refusal), `SiteKey`.

### Added (Studio Forge — increment 2: profiles & dev/int/prod deployment)
- **ADR-0003** (`docs/adr/0003-studio-forge.md`): Go web studio, shared `internal/`,
  profile-based deployment, studio = source of truth (overwrites + backs up).
- `studio/internal/deploy`: `KEY=VALUE` profiles (`deploy/profiles/<name>.conf`, the `.example`
  serves as default, the real gitignored `.conf` takes precedence). Deployment: **validate →
  timestamped backup → overwrite → reload**; **dry-run** (action log). `dev` = local (copy,
  hot-reload); `int`/`prod` = ssh/scp (dependency-free).
- `studio/cmd/forge`: API `GET /api/profiles`, `POST /api/deploy?profile=&dryRun=`.
- UI: profile selector, **Simulate** / **Deploy** buttons (confirmation), log.
- `deploy/profiles/{dev,int,prod}.conf.example`; `.gitignore` covers the real `.conf`.
- Tests: profile parsing/priority, local deployment (backup+overwrite), refusal of
  invalid content, no-effect dry-run. **Validated end-to-end**: forge → deploy `dev` →
  bbsd hot-reloads (verified via `nc`).

### Added (Studio Forge — increment 1: web editor + OASCII preview)
- New **`studio/`** sub-project: a local **Go** web app (stdlib, embedded assets) to
  edit the `site*.json` (pages `menu`/`page`/`applet`, auth gate).
- `studio/internal/store`: lists/loads/saves the sites; **validates via `internal/content`
  (same validation as the server)** before atomic write; refuses path traversal.
- `studio/internal/preview`: renders a page as **40-column colored HTML**, faithful to the engine
  (reuses the `internal/oascii` palette + `content.Ink`).
- `studio/cmd/forge`: `net/http` server (bind **127.0.0.1**, no auth); API
  `GET /api/sites|site`, `POST /api/validate|save|preview`.
- `studio/web`: vanilla JS editor (site/page selection, forms by type, live preview,
  Validate/Save).
- Make target `make studio`. Tests: store (validate-before-write, anti-traversal), preview
  (menu/page/applet rendering, HTML escaping), HTTP handlers (`httptest`). `curl` smoke test OK.

### Changed (Restructuring into 3 sub-projects: server / client / studio)
- The repository is organized into **`server/`** (Go server: `server/cmd/bbsd` + `server/internal/`
  bbs/server/user), **`client/`** (Oric terminal, formerly `oric-client/`) and **`studio/`** (upcoming).
- The **shared** `content` and `oascii` packages stay in the **root `internal/`** so they
  are reusable by the server **and** the studio (Go visibility rule) — zero
  validation/rendering duplication.
- Import paths of the moved packages rewritten; `Makefile` (`make client`, `make studio`),
  `scripts/test-emulateur.sh`, `deploy/vps-deploy.sh`, `.gitignore` and `docs/architecture.md`
  updated. Pure move: **test suite unchanged and green**, client `.tap` identical.

### Added (Login — increment 3: auth applets + wiring + deployment)
- `internal/bbs/login.go`: applets **`login`**, **`register`**, **`guest`** (registered
  via `init`). Handle + password entered line by line (RETURN), **password visible
  on screen** (warned; TLS covers transport), 3 attempts, cancellation by empty field.
  Personalized welcome "Bonjour {handle} — Appel n°{N}" (BBS style), guest access in
  read-only. End-to-end tests (login OK, wrong password, guest, persisted
  registration).
- `content/site.json`: **auth gate at CONNECT** — start page `accueil`
  (Log in / Create an account / Guest) leading to the applets, `next` to `main`.
- `cmd/bbsd`: flag **`-users <file.json>`** (persisted accounts; empty = memory).
- Deployment: systemd unit `-users /var/lib/bbsoric/users.json` + **`StateDirectory=bbsoric`**
  (RW directory owned by the DynamicUser, allowed despite `ProtectSystem=strict`).
- Validated end-to-end via `nc`: registration → hash persisted in `users.json` →
  reconnection and login (case-insensitive handle) → call counter incremented.
- **Oric terminal**: verified that `oric-client/term.s` **already** emits each keystroke
  immediately (`main` loop: `key_scan` → `ser_tx`, no line buffer) — single-key
  input works **without modifying the terminal** (ADR-0002 fixed). `.tap`
  reassembled identically (non-regression); the emulator confirms the
  keyboard→dialing→CONNECT→reception pipeline.

### Added (Login — increment 2: applet engine + single-key input)
- **ADR-0002** (`docs/adr/0002-input-model.md`): Oric terminal in **character mode**,
  `ReadKey` (menus, "press a key") + `ReadLine` (text fields).
- **ADR-0001 revised**: the login becomes an **applet** launched by an `applet`-type page
  (gate at CONNECT via the JSON start page), instead of special per-function targets.
- `server.ReadKey()`: reads a single key (filters IAC, ignores residual CR/LF/NUL). Tests.
- `internal/content`: new **`applet` page type** (`applet` + `next` fields) +
  validation. The page stays JSON; it references a Go applet by name.
- `internal/bbs`: **applet registry** (`Register`/`Applet`/`AppContext`/`Outcome`/
  `SessionState`). The engine (`engine.go`) now navigates **menus and pages by single
  key** (ReadKey) and **dispatches applet pages** (success → `next` page, unknown applet
  handled cleanly). Tests: dispatch + `next` navigation, non-existent applet non-blocking,
  single-key menu navigation validated (+ `nc` demo).

### Added (Login — increment 1: user model + hashed store)
- **ADR-0001** (`docs/adr/0001-login-component-page.md`): the login will be an **isolated
  interactive component** called by a **page via a special target** (`__login__`,
  `__register__`, `__guest__`, `__logout__`), extending
  `__quit__`/`__back__`/`__home__`. The page stays pure JSON. Hashed persistence, cleartext
  password on screen assumed (TLS covers transport), no-echo deferred.
- `internal/user`: model `User` (`Handle`, `PassHash`, `Created`, `LastLogin`, `Calls`)
  and a JSON-file `Store` with a **lock** (concurrent access) and **atomic write**
  (temp file + `rename`). API: `Register`, `Authenticate`, `Get`, `Count`.
- Passwords **never in cleartext**: **PBKDF2-HMAC-SHA256** hashing (`crypto/pbkdf2`,
  **stdlib** Go 1.24+, **no dependency added**), random salt per account, auto-descriptive
  encoded format `pbkdf2$sha256$<iter>$<salt>$<hash>`, constant-time comparison.
- Handle validation (2–16 ASCII alphanumeric characters + `-`/`_`) and password (≥ 4).
- Tests: hash/verify, random salt, rejection of malformed hashes, validation, duplicates
  (case-insensitive), call increment + `LastLogin`, persistence after
  reopening, missing file, **concurrent access** (suite green with `-race`).

### Added (Dynamic content — hot-reloaded JSON page flow)
- `internal/content`: model `Site`/`Page`/`Entry`/`Line` + JSON parsing/validation +
  a `Store` that **hot-reloads the file** (mtime watch; on error,
  the old version is kept). Navigation targets `__quit__`/`__back__`/`__home__`,
  ink colors by name. Default embedded content if no file.
- `internal/bbs/engine.go`: generic engine driven by the `Site` (menu/page rendering +
  stack-based navigation) — replaces the hard-coded menu.
- `cmd/bbsd`: flag `-content <file.json>`.
- `content/site.json`: editable page flow (menus, pages, `Services` submenu).
- `docs/content.md`: JSON format documented.
- Deployment: systemd unit `-content /etc/bbsoric/site.json`; the script **seeds** the JSON
  at initialization only (hot edits on the server are never overwritten).
- Tests: parsing/validation, hot reload, retention on invalid file,
  validity of `content/site.json`. Reload validated end-to-end (adding a menu entry
  visible without a restart).

### Validated (TLS verified end-to-end — AT$CV1)
- Emulator test (`--serial picowifi`) of directory entry 5 **with certificate
  verification**: the terminal loads the **ISRG Root X1** root CA (`AT$CA=` → "CA stored:
  1939 bytes"), enables `AT$CV1`, then dials `ATDT#pavi.3617.fr:6992`. Result:
  **`TLS session up (TLSv1.3, verified)`** → `CONNECT` → BBS banner served through the
  verified TLS tunnel (`docs/img/tls-verified-atcv1.png`).
- Confirms that the Let's Encrypt cert served by Caddy is trusted and that the chain
  (leaf → YE1 → Root YE → ISRG Root X2 → ISRG Root X1) validates on the Pico W side.
- Upload detail: the picowifi segments the `AT$CA=` capture on **LF** (`\n`), the `\r` is
  ignored — the PEM must therefore be sent in lines terminated by `\n`.

### Changed (Production — TLS termination by Caddy + Let's Encrypt)
- TLS of `pavi.3617.fr:6992` is now **terminated by Caddy** (CT 130,
  `caddy-l4`/layer4 module) with a **real Let's Encrypt cert** (`subject=CN=pavi.3617.fr`), instead
  of bbsd's self-signed cert. Caddy decrypts and proxies the cleartext telnet to `bbsd`
  (`.2:6502`). MikroTik NAT `:6992` redirected to Caddy (`.130`). Chain and config
  versioned in `deploy/caddy-tls.md`.
- The Pico W can now **verify the certificate** (`AT$CV1` + Let's Encrypt CA).
- `bbsd -tls-addr 6992` **removed** from the systemd unit: bbsd now only serves cleartext telnet
  (`.2:6502`) to Caddy. CT 130 websites (meteolib/3617/lamatronne…) verified without regression.

### Added (Production — TLS listener 6992, public access open)
- **Deployed**: `bbsd -tls-addr 0.0.0.0:6992` on the LXC pavi3617 (in addition to telnet 6502).
  Both ports listen (same process). systemd unit updated.
- **NAT forward open** on the MikroTik router (dst-nat `:6992` → `192.168.1.2:6992`,
  modeled on the 6502 rule). **`pavi.3617.fr:6992` is reachable publicly over TLS**:
  tested (`openssl s_client` → banner + navigation `1` → Information screen).
- Directory entry 5 (`BBS Oric TLS`) is now operational from a real
  Oric on the Internet (`ATDT#pavi.3617.fr:6992`).

### Added (Oric terminal — real TLS dial + standalone build)
- **TLS dialing**: protocol 2 (TLS) now dials **`ATDT#host:port`** (the `#`
  opens a TLS-terminated call on the Pico W side, firmware v0.2.0) instead of a plain `ATD`.
  **Validated end-to-end** in the emulator (OpenSSL build, `--serial picowifi`): TX
  `ATDT#127.0.0.1:6510` → `TLS session up (TLSv1.3)` → BBS banner rendered through a
  test TLS proxy (`docs/img/tls-dial.png`).
- `oric-client/bin2tap.py`: standalone `.tap` generator (Python) — the build no longer depends
  on the external `bin2tap` of the emulator (which may be cleaned up). `build.sh` uses it.

### Added (Oric terminal — multi-modem + manual host/port/protocol entry)
- **Serial I/O abstraction** via `ACIAPTR` (ZP pointer to the ACIA base) +
  primitives `ser_tx`/`ser_rx_ready`/`ser_rx`. A single `.tap` handles 2 backends, selected
  by a **modem menu** at startup:
  - **1 = direct ACIA 6551** (`$031C`)
  - **2 = LOCI / Pico W** (`$03A0`) — same 6551 interface, different base.
  (DTL2000 excluded: V23/Minitel, no AT or modern TCP. Both backends validated
  end-to-end → `CONNECT to pavi.3617.fr:6502`.)
- **Manual entry** (directory option `M`): **host**, **port**, **protocol**
  fields (1=telnet/raw working, 2=TLS), with echo. The terminal dials `ATD host:port`.
  Routine `input_line` (line input with echo + debounce `wait_release`).
- **TLS**: handled by the **modem** (Pico W) — the Oric does no crypto; on the Oric side the
  protocol picks the command. *(The TLS dial `ATDT#` has since been implemented and validated via
  the `--serial picowifi` backend — see the "real TLS dial" entry.)*
- Captures: `docs/img/modem-menu.png`, `docs/img/manual-entry.png`.
- Test note: `--type-keys` holds a key pressed until an identical key
  or the end of the string, which makes multi-screen navigation hard to automate
  (a tool artifact, not the terminal's); each step validated separately.

### Added (Oric terminal — directory + standalone AT dialing)
- `oric-client/term.s`: **directory (phonebook)** at startup + **standalone Hayes
  dialing**. The terminal displays a list of BBSes (BBS Oric prod, ParticlesBBS, Altair,
  Heatwave), the user chooses (1-4), and the terminal itself dials `ATD<host:port>`
  to the modem, then switches to terminal mode — no more need to configure the modem.
- Routines added: `print_string`, `send_string`, `get_key`, `reset_cursor`; directory
  data + address table of dialing targets.
- Validated on the emulator (`--serial modem`): selection → `ATD` dialed → `CONNECT` to
  `pavi.3617.fr:6502` (`docs/img/phonebook-dial.png`).

### Added (Production deployment)
- `deploy/`: deployment mechanism taken from the telenet project — `deploy.conf` (target LXC
  pavi3617 via VPN mustang), `bbsoric.service` (hardened systemd unit, port 6502), `vps-deploy.sh`
  (compiles linux/amd64 static → copy → installs the unit → restart + verifies the listener).
- `Makefile`: targets `build`, `test`, `vet`, `run`, `oric-client`, `deploy`, `deploy-build`.
- Deployment as a dedicated systemd service `bbsoric` on port 6502 (free; the
  `telenet-bbs` service that existed was inactive), without affecting `telenet-serveur`/`telenet-compagnon`.
- **PRODUCTION RELEASE**: service `bbsoric` `enabled`+`active` on the LXC pavi3617, exposed
  publicly and validated on **`pavi.3617.fr:6502`** (banner + navigation `1`/CR from
  the public Internet). `DynamicUser=yes` (the `bmarty` user does not exist on the LXC;
  avoids running as root).

### Added (Sprint 2 — keyboard emission / interactive BBS)
- `oric-client/term.s`: **keyboard emission (TX)**. Full scan of the 8×8 matrix
  (PSG-via-VIA protocol taken from `Oric asteroids/src/asm/input.s`), ASCII table per
  position (from `src/io/keyboard.c`), debounce (1 character per press), ACIA send and
  **local echo** on screen. The terminal reads/displays (RX) and sends keystrokes (TX).
- `internal/server/session.go`: `ReadLine` now terminates the line on **CR alone** (`$0D`,
  what the Oric sends on RETURN) in addition to LF/CRLF, without blocking reads.
- Test `TestCROnlyLineTermination`.
- `docs/img/sprint2-keyboard-nav.png`: **interactivity proof** — input `1` + RETURN from
  the emulated Oric → "System information" screen displayed (local echo + server navigation).
- Validation: spaced keys all transmitted (`a/b/c/RETURN`); end-to-end menu navigation
  via `--type-keys`. (Caveat: `--type-keys` in a very fast burst during banner loading
  can lose keystrokes — not representative of human typing.)

### Added (Sprint 2 — BBS menu engine)
- `internal/bbs/menu.go`: menu engine. Colored main menu (OASCII) + screens
  **System information**, **About**, **Guestbook** (placeholder), choice-based navigation
  (1/2/3/Q), return to the menu via RETURN, clean exit. Helper `firstKey` (choice routing).
- `internal/bbs/welcome.go`: `WelcomeHandler` now chains banner + menu loop
  (replaces the "hello world" echo).
- Tests: `TestBannerAndMenu`, `TestMenuNavigationAndQuit`, `TestFirstKey` (integration via
  a real socket, byte-based reading robust to prompts without `\n`).
- `docs/img/sprint2-menu.png`: colored menu rendered in the emulator (visual validation).
- Note: navigation from the Oric will require the terminal's keyboard emission (TX) — next step.

### Added (Sprint 1 — Oric terminal + emulator validation)
- `oric-client/term.s`: minimal Oric terminal in 6502 (`xa` assembly). Reads the 6551 ACIA
  `$031C` (9600 8N1, polling) and writes **directly to VRAM `$BB80`** to render the
  OASCII serial attributes; handles CR/LF/scroll, 40-column clamp; loaded/executed at `$1000`.
- `oric-client/build.sh`: assembly + autorun `.tap` generation (via `bin2tap`).
- `scripts/test-emulateur.sh`: headless integration test (server + `oric1-emu` over TCP
  serial + PPM/PNG screen capture).
- `docs/img/sprint1-banner.png`: **visual proof** — the colored banner (yellow/cyan/green/white)
  displays correctly in the emulator, validating the attribute table and the whole network chain.
- `oric-client/README.md`: Oric terminal doc.
- `docs/emulator-testing.md`: validated and automated test procedure (mandatory ROM,
  fast-load, RX FIFO, capture timings).

### Added (Sprint 1 — OASCII layer)
- `internal/oascii`: Oric display layer. Chainable `Builder`
  (`Ink/Paper/Blink/DoubleHeight/AltCharset/Text/Newline`), `Sticky` mode (re-emission
  of attributes per line), low-level encoders `InkAttr/PaperAttr/TextAttr`, color constants
  (Oric palette) and dimensions (`Cols=40`, `Rows=28`).
- Serial Teletext attribute table **extracted from the reference emulator's ULA decoder**
  (`Oric1/oric1-emu`, `src/video/video.c`) → guaranteed reliability. 7 unit tests
  comparing the emitted bytes to the emulator's values.
- `internal/bbs`: **colored** welcome banner via OASCII (yellow title, cyan subtitle,
  green/white info). Attribute byte stream verified via hexdump.
- `docs/oascii.md`: layer specification (the real nature of "OASCII", attribute
  table, palette, API, 40-column layout pitfalls).

### Added (Sprint 0)
- Project initialization (Sprint 0).
- `README.md`: presentation, targets, Oric specifics.
- `ROADMAP.md`: agile plan by sprints (0 to 5) and open decisions.
- `docs/state-of-the-art.md`: analysis of existing retro BBS servers (PETSCII BBS, RetroBBS, Magnetar, TheOldNet) and of the connection ecosystem (LOCI, WiFiModem, Oricutron).
- `docs/architecture.md`: target technical architecture and "OASCII" rendering layer (serial Teletext attributes).
- `docs/agile/backlog.md`: initial product backlog (user stories).
- `.gitignore`.

### Changed
- Account for the **public Internet server** constraint: an "Internet Exposure" section
  in `docs/architecture.md` (public port, lack of TLS, attack surface, hosting, availability);
  security/exposure/hosting raised as cross-cutting concerns in `ROADMAP.md`.

### Decided
- **Server language**: Go 1.26.
- **Hosting**: cloud VPS with a fixed IP.
- **Public port**: `6502`.

### Added (code — Sprint 0, story A3)
- Go module `github.com/bmarty/bbsoric`.
- `internal/server`: TCP server (1 task/connection) with Internet-exposure guardrails
  (global and per-IP limit, idle timeout, slog logging), `Session` API
  (`Write/Println/ReadLine`) with minimal filtering of telnet IAC commands. Public method
  `Serve(ctx, listener)` (tests / socket activation).
- `internal/bbs`: `WelcomeHandler` — 40-column welcome screen + command loop (HELP/QUIT).
- `cmd/bbsd`: configurable daemon (`-addr`, `-max-conns`, `-max-conns-per-ip`, `-idle`, env `BBS_ADDR`),
  clean shutdown on SIGINT/SIGTERM.
- Tests: `internal/server` (echo, IAC filtering, per-IP limit), `internal/bbs` (banner + QUIT, centering).
  All green (`go test ./...`).
- `docs/emulator-testing.md`: 100% local test pipeline via the **single** emulator
  `Oric1/oric1-emu` (Phosphoric) in `--serial tcp:`, + emulated picowifi modem.

### Fixed
- ACIA addressing clarified: `$031C` (Telestrat/oric1-emu) and `$03A0-$03BF` (LOCI MIA), replacing
  the initial value `0x380`.
- **Single test emulator**: all documentation (README, architecture, roadmap, state of the art,
  emulator-testing) now points exclusively to `/home/bmarty/Oric1/oric1-emu`. Removal of
  references to `oric2/Phosphoric` and to Oricutron as test tools (Oricutron is still cited solely
  as part of the "state of the art" catalog of public emulators).
