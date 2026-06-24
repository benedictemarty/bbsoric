# ROADMAP — BBS Oric

Approche **agile**, livraisons incrémentales. Chaque sprint produit un incrément testable.

> **Contrainte transverse : serveur Internet public.** Le BBS est exposé sur Internet (écoute
> `0.0.0.0`, joignable par tout Oric via WiFiModem). Sécurité, exposition et hébergement sont des
> préoccupations de **chaque** sprint, pas seulement du Sprint 5. Voir `docs/architecture.md` §5.

## Sprint 0 — Cadrage & socle ⏳ (en cours)
- [x] État de l'art des serveurs BBS rétro (`docs/etat-de-l-art.md`)
- [x] Cadrage cibles : Oric-1/Atmos + LOCI + WiFiModem ; émulateur de test = `Oric1/oric1-emu` (Phosphoric)
- [x] Initialisation dépôt Git, documentation agile, CHANGELOG, ROADMAP
- [x] **DÉCISION** : langage serveur = **Go** (1.26)
- [x] **DÉCISION** : hébergement = **VPS cloud (IP fixe)** ; port public = **6502** (clin d'œil au CPU)
- [x] Serveur telnet « hello world » écoutant sur `0.0.0.0:6502`, testé via `nc` ✅
- [x] Exposition Internet minimale : limite de connexions globale + par IP, timeout d'inactivité, logs de connexion
- [x] Pipeline de test émulateur confirmé (oric1-emu/Phosphoric `--serial tcp:`) — voir `docs/test-emulateurs.md`

## Sprint 1 — Couche terminal Oric (« OASCII ») 🎯 cœur du projet — ⏳ en cours
- [x] Encodage des **attributs sériels Téletexte** : encre (8), fond (8), clignotement, double hauteur, charset alt
  — table extraite du décodeur ULA de `oric1-emu` (`src/video/video.c`), tests unitaires verts
- [x] `internal/oascii` : `Builder` (`Ink/Paper/Blink/DoubleHeight/AltCharset/Text/Newline`), mode `Sticky`
- [x] Bannière d'accueil colorée (handler) — flux d'octets vérifié au hexdump
- [x] Spec documentée : `docs/oascii.md`
- [x] **Terminal Oric** (`oric-client/term.s`, 6502/xa) : ACIA `$031C` → écriture directe VRAM `$BB80`
  (CR/LF/scroll, clamp 40 col), build `.tap` autorun via `bin2tap`
- [x] **Validation visuelle dans `oric1-emu`** : bannière colorée rendue correctement (jaune/cyan/vert/blanc)
  — capture `docs/img/sprint1-banner.png`, test automatisé `scripts/test-emulateur.sh`
- [ ] Positionnement curseur / `cls` direct (optionnel — l'écriture VRAM gère déjà le rendu ; à définir si besoin)

## Sprint 2 — Moteur BBS — ⏳ en cours
- [x] Boucle de session multi-clients (1 connexion = 1 goroutine) — couche `server`
- [x] Système de menus / navigation (`internal/bbs/menu.go` : menu principal + écrans
  Informations / À propos / Livre d'or, rendu OASCII couleur) — validé écran (`docs/img/sprint2-menu.png`)
- [x] Timeout d'inactivité, déconnexion propre — couche `server`
- [x] **Émission clavier (TX) côté terminal Oric** — scan matrice complet AY-via-VIA
  (`oric-client/term.s`), écho local, terminaison ligne sur CR. **Navigation interactive
  validée à l'écran** (`docs/img/sprint2-keyboard-nav.png`, test via `--type-keys`).
- [~] Login / profils utilisateurs (persistance) — **incréments 1–3 faits** (ADR-0001/0002) :
  - `internal/user` : modèle + store haché atomique (PBKDF2 stdlib), tests `-race`.
  - Saisie **touche unique** (menus) + **ligne/RETURN** (champs texte) : `server.ReadKey`.
  - **Moteur d'applets** : type de page `applet` (JSON) → applet Go enregistré par nom.
  - Applets **`login`/`register`/`guest`**, porte d'auth au CONNECT, accueil personnalisé.
  - Câblage `cmd/bbsd -users` + déploiement (`StateDirectory`). Validé end-to-end (`nc`).
  - **Terminal Oric** : vérifié — `term.s` émet **déjà** chaque frappe immédiatement (mode
    caractère), aucune modif requise (cf. ADR-0002). L'émulateur confirme le pipeline
    clavier→dial→CONNECT→RX.
  - **Reste** : capture émulateur du nouvel écran de login (limite du backend modem émulé
    qui compose les hôtes réels — prévoir entrée locale picowifi / test matériel) ;
    no-echo du mot de passe (optionnel).

## Sprint 3 — Modules de contenu
- [ ] Messagerie / forum (lecture, post)
- [ ] Page d'actualités / annonces
- [ ] Mini-jeu interactif (ex. Puissance 4 / morpion) pour valider l'interactivité
- [~] **Transfert de fichiers (XMODEM)** : côté **serveur fait** (`internal/xmodem`,
  bibliothèque `server/internal/files`, applets `download`/`upload`, flags `-files`/
  `-max-upload`, studio, doc `docs/transfert.md`) ; **reste le terminal Oric**
  (XMODEM 6502 + stockage SD/disquette/cassette). Voir backlog G1.
  - **Écriture disquette Sedoric PROUVÉE** (24/06) dans l'émulateur : `SAVE`
    persiste sur la `.dsk` avec le flag `--disk-writeback` (cause racine du faux
    blocage, ce n'était pas les adresses API).
  - **Reverse du dispatch SAVE complet** (24/06) : carte des routines/variables
    établie (`docs/sedoric-api.md`).
  - **✅ Sauvegarde Sedoric VALIDÉE end-to-end sur SEDORIC V3.0** : un fichier est
    écrit et persisté dans la `.dsk` depuis le langage machine. Recette (manuel
    désassemblé V3.0) : `JSR $04F2` (bascule overlay V3.0) → variables système →
    `JSR $DE9C` (XSAVEB) → `JSR $04F2`. Vecteurs publics confirmés identiques
    V1.0/V3.0. `client/sedoric.s` finalisée (assemble). **Reste** : déclenchement
    par `term.s` après un download + déploiement du terminal sous Sedoric résident.

## Sprint 4 — Connexion matérielle réelle — ⏳ en cours
- [x] **Doc de connexion WiFiModem + LOCI** (`docs/connexion-materielle.md`) : chaîne
  Oric→ACIA→modem→TCP, adressage ACIA `$031C` / LOCI `$03A0-$03BF`, registres 6551,
  commandes AT (`ATD`/`ATDT#`/`AT$CA`/`AT$CV1`), réglages 9600 8N1, dépannage.
- [x] **Programme client/terminal Oric** (`client/term.s`) — terminal autonome
  6502 (menu modem, répertoire, saisie manuelle, numérotation Hayes, mode terminal
  RX/TX), validé end-to-end dans l'émulateur. (réalisé Sprints 1–2)
- [x] **Écran d'accueil ASCII-art Oric** : bannière serveur enrichie d'un art « ORIC »
  5 lignes (glyphes 5×5), centré et conforme OASCII (≤ 40 colonnes), couleurs jaune/cyan.
- [ ] **Test sur Oric réel** — *en attente de matériel*. Protocole de recette
  matérielle (T1–T9) prêt : `docs/connexion-materielle.md` §7.

## Sprint 5 — Déploiement — ✅ terminé (EN PRODUCTION ✅)
- [x] **Déployé en production** sur le LXC pavi3617 (service systemd `bbsoric`, `enabled`+`active`)
  via `make deploy` (mécanisme repris de telenet). Binaire Go statique linux/amd64, `DynamicUser`.
- [x] **Exposition publique validée** : `pavi.3617.fr:6502` (telnet) — bannière + navigation OK
  depuis l'Internet public.
- [x] **Monitoring / alerting dédié** : endpoint HTTP local `/healthz` + `/metrics`
  (format Prometheus, drapeau `-metrics-addr`), sonde `scripts/monitor.sh`
  (healthz/TCP + alerte mail) déclenchée par `bbsoric-monitor.timer` (5 min).
  Déploiement intégré à `vps-deploy.sh`. Doc : `docs/monitoring.md`.
- [x] **Conteneurisation (Docker)** : `Dockerfile` multi-stage (binaire statique →
  image alpine ~18 Mo, non-root, healthcheck `/healthz`), `docker-compose.yml`
  (volume comptes, restart), cibles `make docker-build/up/down`. Build et exécution
  validés (BBS sur 6502, healthcheck `ok`). Doc : `docs/docker.md`. (prod = systemd)
- [x] **Documentation utilisateur** : `docs/guide-utilisateur.md` (connexion depuis
  un Oric réel et depuis un PC pour tester, navigation, comptes, dépannage).
- [x] **Sauvegarde & restauration de l'état** : `scripts/backup.sh` (archive
  `tar.gz` horodatée comptes+fichiers+contenu, rotation, à chaud) +
  `scripts/restore.sh`, timer systemd quotidien (`bbsoric-backup.{service,timer}`),
  test e2e `scripts/test-backup.sh` (13 cas verts), doc `docs/backup.md`.
  Déploiement intégré à `vps-deploy.sh`.

## Studio « Forge » — outillage de contenu ⏳ (en cours)
Sous-projet `studio/` : app web Go locale pour éditer le(s) `site.json` et déployer par
profils. Voir `docs/adr/0003-studio-forge.md`.
- [x] **Restructuration** du dépôt en 3 sous-projets `server/` `client/` `studio/`
  (`internal/{content,oascii}` partagés à la racine).
- [x] **Forge** : éditeur web (pages menu/page/applet), aperçu OASCII couleur, validation
  par `internal/content`, enregistrement atomique.
- [x] **Déploiement par profils** (dev/int/prod) : valide → sauvegarde → écrase → reload,
  dry-run ; `dev` local (hot-reload), `int`/`prod` ssh/scp. Validé end-to-end.
- [x] **Menu sur fond d'écran** : une page `raw` (buffer 40×28 composé) se combine
  avec des `entries` — décor en fond + navigation par touches (page ou applet),
  éditeur de navigation intégré à l'onglet « Écran ».
- [x] **Pages de saisie déclaratives** (`content.Form`) : applet générique `form`
  (login/inscription) piloté par des champs déclarés (clé/libellé/secret) + action ;
  éditeur de formulaire dans le studio. La logique sensible (hachage) reste serveur.
  **Réessai sur place** (`Retries`) + **page d'échec** (`Fail`, aussi pour les
  entrées ▶ applet) en plus du `Next` de succès.
- [x] **Positionnement curseur (plot X,Y)** : séquence terminal `1F col row`
  (`oascii.Plot`/`Builder.At`), champs de formulaire positionnables (`Field.At`),
  colonnes X/Y dans le studio. Décor + invites placées à coordonnées absolues.
- [x] **Buffer écran différentiel** (`oascii.Screen`) : rendu « dirty cells » —
  n'émet que les cellules modifiées (segments + plot X,Y). Base des écrans
  dynamiques (jeux, animations) sur liaison série lente.
- [ ] Multi-sites avancé (création de nouveaux fichiers depuis l'UI), gestion des sauvegardes.
- [ ] Authentification si le studio devait être exposé (aujourd'hui local-only).

---

## Décisions actées
- **Langage serveur** : Go 1.26 (`cmd/bbsd`, `internal/server`, `internal/bbs`).
- **Hébergement** : VPS cloud avec IP fixe (exposition Internet publique 24/7).
- **Port public** : `6502`.
- **Test** : émulateur **unique** `Oric1/oric1-emu` (Phosphoric) via socket TCP (`--serial tcp:`).

## Décisions ouvertes (ADR à formaliser)
1. **Adressage ACIA** — supporter `$03A0-$03BF` (LOCI) et `$031C` (Telestrat/oric1-emu) côté client.
2. **Protocole telnet** — négociation IAC complète vs filtrage minimal (actuel). À trancher au Sprint 1.
3. **Rendu OASCII** — table d'attributs Téletexte Oric exacte à valider sur émulateur (Sprint 1).
