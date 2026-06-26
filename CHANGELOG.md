# CHANGELOG — BBS Oric

Toutes les modifications notables de ce projet sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) ;
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]

### Ajouté (Transfert XMODEM — jauge de progression, 26/06/2026)
- **Barre de progression `[####------]  NN%`** pendant les transferts XMODEM,
  côté terminal (le canal étant du binaire brut, seul le terminal voit les blocs).
  - **Protocole** : le serveur envoie le **nombre total de blocs** (2 octets, bas/haut)
    juste après `1F FE` (`server/internal/bbs/xfer.go`). Un terminal antérieur
    ignore ces 2 octets (non-SOH) ; un terminal récent **exige** ce total
    (sinon pas de jauge). Download/upload : tests serveur verts.
  - **Terminal** (`client/xmodem.s`) : `handle_rx` lit le total (états 3/4) ; la
    barre (BARLEN=20 segments) se remplit par **comptage de Bresenham** (pas de
    mult/div 16 bits), pourcentage = segments×5, affichage en ligne fixe (row 25).
    L'upload calcule le total depuis `XSIZE`. Variables de jauge aliasées sur des
    cases zero-page **inactives pendant un transfert** (saisie/plot/dial) ; `PLOTST`
    réinitialisé après transfert (l'alias `XACC` l'écrase).
  - **Validé** dans l'émulateur (download local) : barre à **40%** à mi-transfert,
    **100% + « FICHIER RECU EN 4000 »** à la fin.
  - ⚠️ La prod doit être redéployée (le serveur courant n'annonce pas encore le
    total — un terminal v0.1.2 ne pourrait pas télécharger tant que la prod n'a
    pas le nouveau `xfer.go`).

### Déployé (Production — backspace serveur, 26/06/2026)
- **Prod `pavi.3617.fr` mise à jour** (`make deploy`) avec la gestion backspace
  (`Session.ReadLine` traite `$08`/`$7F`). Vérifié end-to-end : accueil rendu,
  formulaire login traitant une saisie corrigée par backspace, service `active`.

### Corrigé (Terminal Oric — revue ingénieur, 26/06/2026)
Revue complète du client 6502 (`docs/revue-client.md`). Correctifs livrés :
- **LOCI — mauvaise base ACIA** (`client/term.s`) : l'option « 2 = LOCI » visait
  **`$03A0`** (espace **MIA** du LOCI), pas le modem → collision MIA/ACIA, PSG
  perturbé, **clavier figé sur l'annuaire**. Corrigé vers **`$0380`** (ACIA du
  modem WiFi LOCI, cf. firmware `PicoWiFiModemUSB`). **Validé** émulateur
  `--loci --serial picowifi` : `2`→`1`→`CONNECT pavi.3617.fr` (bannière rendue).
  Détail : `phosphoric-findings.md` F1. Docs alignées (`$03A0`→`$0380`).
- **Plot hors limites** (`set_cursor_xy`) : clamp `row<28`/`col<40` — supprime une
  écriture hors VRAM pilotée par une entrée réseau non fiable (BBS tiers).
- **Réception XMODEM non bornée** : refus au-delà de `$B800` (`CAN` + « FICHIER
  TROP GROS ») — supprime un débordement de tampon depuis le réseau.
- **Majuscules (SHIFT)** : `scan_shift` + `key_scan` (`a-z`→`A-Z`) — l'identification
  avec mot de passe à casse mixte devient possible. Validé (trace : TX `Y`/`Z`).
- **Backspace** : touche **DEL** (col5/row5) → `$08` ; `putbyte`/`input_line`
  (client) et `Session.ReadLine` (serveur, `$08`/`$7F`) effacent le dernier
  caractère. Test `TestReadLineBackspace` (4 cas).
- Commentaire `sei` + carte zero-page documentée (`term.s`).
- **Différés documentés** (avec justification) : contrôle de flux RX (#1), codes
  modem/DCD (#6), telnet IAC (#7), overrun ACIA (#8), nom Sedoric (#9), tests
  client (#12). Voir `docs/revue-client.md`.

### Documenté (findings émulateur, 26/06/2026)
- **`phosphoric-findings.md`** (nouveau) : journal des défauts de l'émulateur
  Phosphoric repérés depuis bbsoric. F1 = `--loci` + `--acia-addr 03A0` fige le
  clavier (double mappage `$03A0` MIA/ACIA, la MIA masque l'ACIA et casse le scan
  clavier via le PSG). **Le picowifi EST le modem du LOCI** → bon modèle fidèle :
  `--loci --serial picowifi` (sans `--acia-addr` ; ACIA par défaut `$0380`), le
  terminal devant adresser **`$0380`** et non `$03A0`. → **à corriger côté terminal**
  (`client/term.s`) : l'option menu « `2` = `$03A0` » devrait viser `$0380`. Garde-fou
  Phosphoric ≥ 1.27.3 (message pointant vers `$0380`). Dépannage ajouté à
  `docs/connexion-materielle.md` et au skill `run-bbsoric`.

### Distribué (Release GitHub — terminal Oric alpha, 26/06/2026)
- **Release `v0.1.0-alpha`** (prerelease) sur le dépôt public :
  <https://github.com/benedictemarty/bbsoric/releases/tag/v0.1.0-alpha>.
  Assets : **`term.tap`** (cassette autorun, 3 668 o) et **`term-boot.dsk`**
  (disquette Sedoric bootable avec `TERM.COM`, 1 Mo) — tous deux reconstruits
  depuis `term.s` courant (`make client` + `client/build-disk.sh`). Notes incluant
  les commandes de lancement émulateur, dont le **piège LOCI** (ne pas combiner
  `--loci` avec `--acia-addr 03A0` : double mappage `$03A0` qui fige le scan
  clavier ; utiliser `--serial picowifi --acia-addr 03A0` sans `--loci`).

### Déployé (Production — chat en ligne, 25/06/2026)
- **Prod `pavi.3617.fr` mise à jour** via `make deploy` (binaire avec présence +
  applets `who`/`chat`, service `bbsoric` actif, écoute 6502 OK) puis push du
  `content/site.json` à jour (sauvegarde distante `site.json.bak-20260625-225909`,
  rechargé à chaud). **Vérifié end-to-end sur `pavi.3617.fr:6502`** : invité →
  menu principal → **Communauté** → *Qui est en ligne* / *Chat* (salon ouvert).
  Le chat temps réel est désormais utilisable en ligne.

### Mis à niveau (Studio « Forge » — applets who/chat)
- **`studio/web/app.js`** : `KNOWN_APPLETS` complété avec **`who`** et **`chat`**
  (la liste déroulante « ▶ applet » les propose désormais sans faute de frappe) ;
  ajout d'une **infobulle descriptive** par applet (`APPLET_DESC`). Le studio
  charge/édite/prévisualise le menu **Communauté** (écran rendu via
  `internal/render`, identique au serveur). Vérifié : `/api/site` charge la page,
  `/api/screen?page=communaute` rend 192 octets OASCII (HTTP 200), tests studio
  `-race` verts. Doc `studio/README.md` mise à jour (liste des applets connus).

### Ajouté (Sprint 7 — Qui est en ligne + chat entre appelants)
- **Communication temps réel entre sessions** (première brique de parité
  état-de-l'art, cf. `docs/etat-de-l-art.md` §6) :
  - **`server/internal/presence`** : registre en mémoire des connectés
    (« qui est en ligne ») + relais de chat **pub/sub à diffusion non bloquante**
    (un abonné lent ne fige jamais l'émetteur) avec **rappel** des messages
    récents. Tests : présence, tri par arrivée, backlog borné, non-blocage tampon
    plein, désabonnement.
  - **Applet `who`** : liste des appelants (pseudo + durée de présence, marqueur
    « (vous) »).
  - **Applet `chat`** : salon temps réel. **Un seul goroutine lit la session**
    (lecture octet par octet avec échéance courte, drainage des messages entre
    deux frappes) — pas de vol d'octets au moteur, écho local, `/q` pour quitter,
    messages système d'arrivée/départ, horodatage `HH:MM`.
  - Pseudo de présence posé à l'identification (`Invite-N` pour les invités,
    pseudo du compte sinon) ; `SessionState` étendu (`Presence`, `MemberID`,
    `Handle`) ; `WelcomeHandler.Presence` injecté depuis `cmd/bbsd`.
  - **Contenu** : menu **Communauté** (touche 6 du menu principal) → *Qui est en
    ligne* / *Chat*.
- **Tests** : package `presence` + intégration `who`/`chat` (deux clients TCP,
  relais de message vérifié). Suite complète verte, **`go test -race` propre**.
  Validé aussi en live (deux invités, message relayé avec pseudo + horodatage).

### Documentation (État de l'art — parité fonctionnelle / écarts, 25/06/2026)
- **`docs/etat-de-l-art.md` §6** : comparaison **fonctionnelle** du serveur à
  l'état de l'art (réf. petscii-bbs). Recense l'existant, puis l'écart principal —
  les **espaces de communication entre appelants** (le « Livre d'or » est statique,
  non inscriptible) — et priorise 6 features : mur one-liner (#2), message
  base/forums (#1), qui-est-en-ligne + chat (#3), messagerie privée (#4),
  actualités RSS→OASCII (#5), door game (#6).
- **`ROADMAP.md`** : nouveau **Sprint 7 — Communication entre appelants** reprenant
  ce backlog (chaque feature = un applet `bbs.Register` + store persisté).

### Communication (Annonce alpha publiée — Defence Force, 25/06/2026)
- **Annonce publique de la version alpha** (serveur + terminal Oric + studio
  « Forge ») sur le **forum Defence Force** :
  <https://forum.defence-force.org/viewtopic.php?t=2897>. Vidéo de démo :
  <https://youtu.be/YRFBYkpsKMc>. Texte source : `~/bbsoric-announce-defence-force.md`
  / `.bbcode.txt`. Trace complète : **`docs/communication.md`**.
- **Dépôt GitHub rendu public** : <https://github.com/benedictemarty/bbsoric>
  (historique réécrit au préalable via `git filter-repo` pour purger les IP
  internes ; placeholders dans `deploy/caddy-tls.md`).
- L'annonce lance un **appel à test sur matériel réel** (rendu terminal, timing
  XMODEM série, write Sedoric sur lecteur physique) — retours à consigner dans
  `docs/communication.md`.

### Corrigé (XMODEM terminal — démarrage rapide, plus de figeage sur gigue réseau)
- **`client/xmodem.s`** : la réception de bloc utilisait `xr_rx` **bloquant
  (sans timeout)** → si un octet d'un bloc tardait (segmentation/gigue TCP vers
  la prod), le terminal **se figeait indéfiniment** au lieu de re-NAK, d'où un
  démarrage de download de ~43 s. Remplacé par **`xr_rx_t`** (timeout ~1,3 s,
  préserve Y) ; sur octet manquant, `bcc xr_start` → **re-NAK rapide** (le
  serveur renvoie le bloc). Mesuré sur la prod : démarrage **~43 s → ~2 s**,
  fichier complet reçu (174 ACK). Le débit du transfert reste borné par le
  réseau (XMODEM stop-and-wait, 1 aller-retour par bloc).

### Validé (Téléchargement XMODEM client↔prod — bout-en-bout + diagnostic démarrage)
- **Download end-to-end PROUVÉ** : un terminal Oric émulé connecté à la **prod**
  (`pavi.3617.fr` via backend modem) télécharge `ASTERORIC.TAP` (22 Ko) jusqu'à
  « FICHIER RECU EN 4000 / Téléchargement terminé » (réception en RAM `$4000`).
  Rendu possible par le cadençage `--realtime` de Phosphoric (sinon timing
  non déterministe). Vidéo : `~/bbsoric-client-prod-demo.mp4`.
- **Diagnostic (trace série)** : le `1F FE` (RecvCmd) est bien reçu et
  `handle_rx` bascule en `xmodem_recv`, mais le **démarrage est lent** — le
  récepteur ré-émet le NAK 2-3 fois avec un **timeout long (~32 s)** avant de se
  synchroniser et d'ACK les blocs. Course de démarrage (ordre RecvCmd→1er bloc
  vs NAK du récepteur). **Piste d'optimisation** : raccourcir le timeout de
  re-NAK de `xmodem_recv` (`client/xmodem.s`) et/ou garantir le flush du
  `RecvCmd` avant `waitStart` côté serveur. Le transfert lui-même est intègre.

### Déployé (Production — alignement complet, 25/06/2026)
- **Prod `pavi.3617.fr` (LXC pavi3617) ré-alignée sur le repo** via `make deploy`
  (binaire à jour + service avec `-files`/`-users`/`-metrics-addr` + timers
  sauvegarde/monitoring) puis push du `content/site.json` courant
  (sauvegarde préalable `site.json.bak-20260625-111109`).
- **Vérifié end-to-end** sur `pavi.3617.fr:6502` : accueil → invité → menu
  principal → **Fichiers** → applet **Téléchargement** opérationnel (`-files`
  actif, bibliothèque vide). La prod expose désormais le même niveau fonctionnel
  que le repo (serveur = studio = client).

### Ajouté (Outillage — skill « run-bbsoric »)
- **`.claude/skills/run-bbsoric/`** : skill de lancement/pilotage. `SKILL.md`
  (build, run, test, studio, terminal) + **`driver.py`** : harnais qui pilote le
  serveur BBS par socket TCP (envoi de touches, lecture/rendu OASCII, smoke flow
  bannière + navigation menu, captures `/tmp/bbs-*.txt`). Vérifié de bout en bout
  (`make build` → `./bbsd` → driver `exit 0`, 4 écrans validés).

### ✅ Sedoric — sauvegarde disquette VALIDÉE end-to-end sur SEDORIC V3.0
- **Sauvegarde Sedoric prouvée depuis le langage machine** : un fichier
  (`TESTML  BIN`) est **écrit et persisté** dans `sedoric3.dsk` (entrée catalogue
  + write-back, md5 modifié) par la séquence ML, testée dans l'émulateur.
- **Recette V3.0** (manuel désassemblé SEDORIC 3.0, ANNEXE 15) — `JSR $04F2`
  (bascule ROM→overlay) → poser BUFNOM/VSALO0/FTYPE/DESALO/FISALO/LGSALO/EXSALO
  → `JSR $DE9C` (XSAVEB) → `JSR $04F2`. La bascule overlay **change selon la
  version** : `$04F2` en V3.0, `$0472` en 1.x/2.x. Confirmé d'abord par l'exemple
  « HELLO ANDRE » de l'ANNEXE 15, puis par XSAVEB.
- **Vecteurs publics confirmés identiques V1.0/V3.0** (dump vue CPU pendant SAVE) :
  `$FF7C = JMP $DE9C` (XSAVEB), `$FF76 = JMP $DE28` (XDEFSA). `$DE9C` débute par
  `SEI $78` (sert de détection « Sedoric résident »).
- **`client/sedoric.s` finalisée** : `OVL_TOGGLE = $04F2`, `XSAVEB = $DE9C`,
  variables aux adresses documentées (`$C04D`/`$C051`/`$C052`/`$C054`…), détection
  `$78`. Assemble (`build.sh` vert). Deux PDF (« Sedoric à nu » V1.0 + manuel
  désassemblé V3.0) fournis par l'utilisateur exploités.
- **Garde de présence Sedoric (sûre sans disque)** : `sed_save` vérifie d'abord,
  en RAM page 4 toujours mappée, la table de saut installée par Sedoric
  (`$04F2`/`$04F5` = `4C xx 04`) **avant** tout `JSR $04F2`. Validé : sous Sedoric
  la garde passe et le fichier est sauvé (`TESTG4 BIN`) ; sans disque `$04F2=$55`
  → garde refuse, pas de plantage. Le même terminal est donc sûr en cassette et
  sous Sedoric.
- **Intégration déjà câblée** : `term.s` (`handle_rx`) appelle `sed_save` après un
  download, `XSIZE` posé par le récepteur XMODEM.
- **✅ Disquette bootable du terminal** : `client/build-disk.sh` (reproductible)
  produit `term-boot.dsk` = disquette Sedoric master + **TERM.COM** (terminal
  injecté en RAM par fast-load tape puis `SAVE` Sedoric). Le terminal **tourne**
  depuis la disquette (`LOAD"TERM":CALL#1000` → menu modem affiché, ~2,6 M
  instructions exécutées). Le `BREAK` initial venait de l'option `,J` (résolu :
  `LOAD`+`CALL`). L'ACIA `$03A0` (LOCI) est un choix runtime (menu) pour cohabiter
  avec le Microdisc — pas de variante de build. Auto-démarrage hands-free =
  raffinement (remplacer le programme de boot du master). Voir `docs/sedoric-api.md`.
- *Détail outillage* : xa65 scinde les commentaires sur « : » (commentaires sans
  deux-points) ; `--type-keys` perd parfois le 1er caractère d'une ligne.

### Ajouté (Contenu — sous-menu Fichiers : download/upload accessibles)
- **`content/site.json`** : entrée **« Fichiers »** (touche `5`) au menu principal
  → page **`fichiers`** avec **Télécharger** (applet `download`) et **Téléverser**
  (applet `upload`), `next: fichiers` (retour au sous-menu), plus `Retour`.
  Les applets XMODEM (déjà codés/testés) sont enfin **joignables depuis l'UI**
  (ils étaient enregistrés mais câblés nulle part). JSON validé, cibles
  cohérentes, tests `content`/`bbs` verts.

### Ajouté (Infrastructure — sauvegarde & restauration de l'état)
- **`scripts/backup.sh`** : archive `tar.gz` horodatée de l'état persistant
  (comptes `users.json`, bibliothèque `files/`, contenu `site.json`) dans
  `/var/backups/bbsoric/`, avec **rotation** (14 par défaut) et **manifeste**.
  Sauvegarde **« à chaud »** (écritures serveur atomiques → pas d'arrêt requis).
- **`scripts/restore.sh`** : restauration d'une archive (`<fichier>`, `latest`
  ou `--list`) — arrêt service → mise à l'écart `*.pre-restore` → restauration
  → redémarrage (systemd réapproprie le `StateDirectory` sous `DynamicUser`).
- **`deploy/bbsoric-backup.{service,timer}`** : sauvegarde **quotidienne**
  (03h30, `Persistent=true`), durcie (`ReadWritePaths` au seul dossier backups).
- **`deploy/vps-deploy.sh`** : installe scripts backup/restore + active le timer.
- **`scripts/test-backup.sh`** : test bout-en-bout (13 cas) — sauvegarde,
  contenu d'archive, restauration après corruption, `latest`, rotation. **Vert.**
- **`docs/backup.md`** : procédure complète (cible, mécanisme, restauration,
  note `DynamicUser`, hors-site).

### Investigué (Sedoric — reverse complet du dispatch SAVE)
- **Carte reverse établie** (save-state au prompt + trace CPU + watchpoint
  `memory_set_trace`) : buffer commande **`$0035`**, **scanner auto-modifiant
  `$00E2`–`$00ED`** (opérande de `LDA $00E8` avancée via `$E9/$EA`), table de
  mots-clés **`$CA6F`** (match via `$DE/$DF`, séparateur `$22`), helper compare
  `$D5B5`, cluster save `$D33x`–`$D39x`, primitive FDC `$D075`, trampolines
  page 4 (`$04EF`→`$C4A0`).
- **Conclusion décisive** : le `SAVE` est **dispatché par la ROM BASIC**
  (`$F6xx`–`$F8xx`) puis le scanner Sedoric — `$C4A0` n'est exécuté qu'une fois
  en idle, pas sur le chemin du SAVE. Le dispatch dépend de nombreuses variables
  zéro-page → **pas d'entrée ML isolable triviale** ; appeler `SAVE` depuis du
  code autonome n'est pas un simple `JSR`.
- **Voie retenue** : mécanisme **documenté** d'exécution de commande Sedoric
  depuis l'ML (à obtenir via « Sedoric à nu »/manuel) — seule voie portable
  matériel réel ; alternative : injection clavier (type-ahead).
- **Déploiement** : `tap2sedoric` (oric1-emu) est un **stub** → pas de `.dsk`
  directe ; voie réaliste = **`CLOAD` du terminal sous Sedoric résident**.
- **`client/sedoric.s`** : code par vecteurs `$FF7x` marqué **superseded**
  (garde no-op sûre conservée). **`docs/sedoric-api.md`** : carte + approches
  recommandées + déploiement.

### Investigué (Stockage Microdisc/Sedoric — écriture disquette PROUVÉE)
- **Cause racine du « blocage » identifiée** : ce n'était ni les adresses de
  l'API Sedoric ni le mapping ROMDIS, mais le flag émulateur **`--disk-writeback`**
  (write-back opt-in, désactivé par défaut). Sans lui, le `SAVE` écrit l'image
  **en mémoire** mais rien n'est persisté dans la `.dsk` hôte.
- **Chaîne d'écriture validée end-to-end** dans `oric1-emu` : boot **Sedoric V3.0**
  résident (`-r basic11b.rom --disk-rom microdis.rom -d sedoric3.dsk`), `SAVE`
  binaire depuis le prompt → fichier réel écrit (entrée catalogue `TEST     BIN`,
  données + bitmap), persisté avec `--disk-writeback` (md5 `.dsk` modifié).
  Primitive FDC d'écriture secteur en `$D075` (cmds Type II `$A8`/`$AC`).
- **`microdis.rom` = `Oric DOS V0.6`** : page `$FF` vide → les vecteurs du PDF
  (`$FF73`…) n'y sont pas ; l'API Sedoric est installée en RAM overlay au boot.
- **`docs/sedoric-api.md`** : section « Écriture disquette VALIDÉE » (cause
  racine, recette reproductible, conséquences). **`client/sedoric.s`** : statut
  corrigé (l'appel par vecteurs PDF reste à recaler via trace du `SAVE`).
- *Reste (G1, voie B)* : tracer l'**entrée d'appel machine** du `SAVE` pour la
  reproduire depuis le terminal, et faire tourner le terminal **sous Sedoric
  résident** (`.dsk` bootable).

### En cours (Stockage Microdisc/Sedoric — voie B, exploration)
- **`docs/sedoric-api.md`** : API Sedoric extraite du désassemblage (vecteurs
  `$FF73`/`$FF76`/`$FF79`/`$FF7C`, variables `BUFNOM`/`DESALO`/`FISALO`) + séquences
  SAVE/LOAD.
- **`client/sedoric.s`** : `sed_save` (sauve `$4000` en fichier via l'API),
  **détection sécurisée** (ne plante pas sans Sedoric) ; `handle_rx` appelle
  `sed_save` après un download. Assemblé.
- **Découverte (tests émulateur)** : le **mapping ROM Microdisc** masque les
  vecteurs page `$FF`, et les adresses du PDF ne correspondent pas à `sedoric3.dsk`
  → l'appel n'est pas opérationnel tel quel. Recalage des adresses sur la version
  cible + gestion ROMDIS nécessaires (sous-projet de reverse, validation matériel
  réel recommandée). Sedoric boote bien dans l'émulateur. Backlog **G1**.

### Ajouté (Terminal Oric — envoi de fichier XMODEM, upload)
- **`client/xmodem.s`** : émetteur **XMODEM 6502** en **CRC-16** (`xmodem_send` +
  `crc_update`) — envoie `XSIZE` octets depuis le buffer RAM (`$4000`), ré-émission
  sur NAK/timeout, EOT. Le CRC évite le délai de bascule côté récepteur (le serveur
  démarre en CRC).
- **`client/term.s`** : `handle_rx` détecte **`1F FD`** (`oascii.SendCmd`, émis par
  l'applet `upload`) et lance `xmodem_send`.
- **Validé dans l'émulateur** : un Oric téléverse 256 octets, reçus intacts et
  stockés côté serveur — `docs/img/xmodem-upload.png` (« FICHIER ENVOYE » /
  « Recu : f (256 octets) »). Transfert **bidirectionnel** Oric ↔ serveur complet.
- *Reste* : **stockage** sur mémoire de masse (carte SD via LOCI / Microdisc /
  cassette) — aujourd'hui le buffer est en RAM `$4000` (backlog G1).

### Ajouté (Terminal Oric — réception de fichier XMODEM, download)
- **`client/xmodem.s`** : récepteur **XMODEM 6502** (mode somme de contrôle), reçoit
  un fichier en RAM (`$4000`), ACK/NAK, EOT. `xr_rx` préserve Y (que `ser_rx`
  écrase) — bug de boucle corrigé.
- **`client/term.s`** : `handle_rx` détecte la séquence **`1F FE`** envoyée par le
  serveur et bascule en réception (`xmodem_recv`) ; `build.sh` intègre `xmodem.s`.
- **Signalisation** : `oascii.RecvCmd()` (`1F FE`) / `SendCmd()` (`1F FD`) ;
  l'applet `download` émet `RecvCmd` avant l'envoi XMODEM.
- **Validé dans l'émulateur** : un Oric télécharge un fichier (128 o) du serveur,
  reçu intact en RAM — `docs/img/xmodem-download.png` (« FICHIER RECU EN 4000 »).
- *Reste* : upload 6502 (émetteur), stockage carte SD (LOCI)/Microdisc/cassette
  (aujourd'hui réception en RAM uniquement) — backlog G1.

### Ajouté (Transfert de fichiers — download/upload XMODEM, côté serveur)
- **`internal/xmodem`** : protocole XMODEM (blocs 128 o, somme de contrôle **et**
  CRC-16, ré-émission, élagage du remplissage `SUB`). Tests round-trip (checksum +
  CRC) via `net.Pipe`.
- **`server/internal/files`** : bibliothèque de fichiers sur disque (liste,
  lecture, écriture atomique), noms validés (anti path-traversal), taille max.
- **`server.Session.Raw()`** : canal d'octets brut pour les transferts binaires
  (court-circuite le filtrage telnet/ligne) + `ClearDeadline()`.
- **Applets `download`/`upload`** (`server/internal/bbs/xfer.go`) : download liste
  la bibliothèque et **envoie** un fichier par XMODEM ; upload **reçoit** et
  enregistre. Injectés via `AppContext.Files` / `WelcomeHandler.Files`. Tests
  end-to-end (`TestDownloadApplet`, `TestUploadApplet`).
- **`bbsd`** : flags `-files <dir>` et `-max-upload <octets>` ; `bbsoric.service`
  utilise `/var/lib/bbsoric/files`. Studio : `download`/`upload` dans le sélecteur
  d'applets. Doc : `docs/transfert.md`.
- *Reste à faire côté Oric* : mode transfert + XMODEM 6502 + stockage SD/disquette
  dans `client/term.s` (backlog G1).

### Ajouté (Rendu — repli automatique des lignes > 40 colonnes)
- **`internal/render`** : une ligne de texte qui dépasse **40 colonnes** est
  désormais **repliée** sur la ligne suivante (coupure aux espaces ; césure dure
  pour un mot plus long qu'une ligne) au lieu d'être tronquée par le terminal.
  Au passage à la ligne, les **attributs courants (encre/fond/…) sont ré-émis**
  pour conserver le même rendu (l'ULA les réinitialise à chaque début de ligne).
  Ne concerne que les pages « logiques » (`writeLine`/`Screen`) ; l'« écran brut »
  (`RawScreen`) reste émis tel quel. Test `TestWrapWidthAndColor`.

### Ajouté (Applets — réessai + page d'échec)
- **Réessai sur place** : l'applet `form` redemande les champs en cas d'échec
  jusqu'au succès ou épuisement des tentatives (`Form.Retries`, défaut 3).
  L'annulation (1er champ vide) reste un retour volontaire, pas un échec.
- **Page d'échec configurable** : nouveau `Outcome.Failed` ; en échec définitif,
  le moteur route vers **`Form.Fail`** (page form) ou **`Entry.Fail`** (entrée
  ▶ applet) si défini, sinon retour arrière / maintien au menu. Les applets
  `login`/`register` signalent aussi `Failed` après « Trop de tentatives ».
- **Validation** : `Form.Fail` / `Entry.Fail` doivent désigner une page existante.
- **Studio** : `formEditor` expose « En cas d'échec » (page) + « Tentatives » ;
  l'entrée ▶ applet a un sélecteur « page si échec » (à côté de succès).
- Tests `TestFormFailToPage`, `TestFormRetryThenSuccess`.
- **Contenu** (`content/site.json`) : les pages `login`/`register` routent vers une
  page **`echec`** dédiée (`fail: echec`) après épuisement des tentatives.

### Modifié (Studio — formulaire éditable depuis l'onglet Écran)
- **Onglet « Écran »** : le bloc sous la grille gère maintenant aussi le
  **formulaire de saisie** (applet `form`), pas seulement les entrées de menu.
  Une page `form` (ex. `login`) chargée dans l'éditeur d'écran affiche son
  `formEditor` (action, champs, **positions X/Y**, next) ; une page de menu garde
  son éditeur d'entrées + un bouton « + formulaire ». On peut donc composer un
  **login plein écran** d'un seul endroit : décor dans la grille + champs
  positionnés. `formEditor` rendu réutilisable (callback de rafraîchissement).

### Modifié (Studio — insertion d'applet par liste déroulante)
- **Éditeur d'entrées** (onglets Édition *et* Écran) : pour une entrée
  « ▶ applet », le nom se choisit désormais dans une **liste déroulante**
  (`login`/`register`/`guest`, + la valeur courante si personnalisée) au lieu
  d'un champ texte libre — plus de faute de frappe. `appletSelect` /
  `KNOWN_APPLETS`.

### Modifié (Studio — retrait du compositeur, navigation Écran plus visible)
- **Onglet « Édition »** : suppression du **compositeur de ligne** (canvas + palette
  `glyph-palette` + boutons `comp-*`), redondant avec l'éditeur d'écran case par
  case. Code/HTML/CSS associés retirés (`comp`, `drawComp`, `compAdd`,
  `compInsert`, `renderPalette`).
- **Onglet « Écran »** : l'éditeur de **navigation du menu** est désormais
  **découvrable** — affiché dès l'ouverture de l'onglet (avec un message d'invite
  quand aucune page n'est chargée), au lieu d'apparaître seulement après le
  chargement d'une page.

### Ajouté (Buffer écran « intelligent » — rendu différentiel)
- **`internal/oascii.Screen`** : buffer 40×28 qui maintient l'état composé ET
  l'état affiché par le terminal. `Render()` n'émet QUE les cellules modifiées,
  regroupées en segments (positionnés par plot X,Y), sans franchir les fins de
  ligne. Exact sur Oric (l'écran EST la VRAM : chaque cellule est indépendante,
  l'ULA recompose la ligne au balayage). Économise la liaison série 9600 bauds
  pour les écrans dynamiques (jeux, valeurs rafraîchies) — réémettre tout coûte
  ~1,2 s, un diff de quelques cellules est quasi instantané.
- API : `NewScreen`, `Put`/`PutText`/`Clear`/`At`/`Buffer`, `Render` (diff +
  mémorisation), `Reset` (force une réémission complète). Le diff saute même les
  cellules communes en tête d'un changement (« 000 »→« 042 » n'émet que « 42 »).
  Tests `TestScreen*`.

### Ajouté (Exemple — page de login plein écran + capture émulateur)
- **`docs/examples/example-login.json`** : page de connexion **plein écran** combinant un
  **décor `raw`** 40×28 (cadre, titres colorés, libellés « Pseudo »/« Mot de passe »)
  et un **`form`** dont les champs sont **positionnés** (`at:[20,11]`, `[20,14]`) par
  plot X,Y. L'applet `form` affiche un décor raw plein écran depuis (0,0)
  (`server/internal/bbs/form.go`).
- **Capture émulateur** `docs/img/example-login-plein-ecran.png` : rendu réel sur
  oric1-emu (ULA) — décor + invite du champ login placée à ses coordonnées.

### Ajouté (Positionnement curseur — plot X,Y)
- **Terminal Oric** (`client/term.s`) : machine à états sur le flux RX — la
  séquence **`1F col row`** repositionne le curseur d'écriture VRAM
  (`handle_rx`/`set_cursor_xy`, `SCRPTR = $BB80 + row*40 + col`). Assemblé (xa).
- **`internal/oascii`** : constante `PlotByte` (0x1F), `Plot(col, row)` et
  `Builder.At(col, row)` ; test `TestPlot`.
- **Champs positionnés** : `content.Field.At [col,row]` (validé : longueur 2 et
  dans l'écran 40×28). L'applet `form` émet la séquence de positionnement avant
  l'invite du champ ; sinon affichage séquentiel. Test `TestFormFieldPlot`.
- **Studio** : colonnes **X / Y** par champ dans l'éditeur de formulaire (vides =
  invite séquentielle). Doc : `docs/oascii.md` (section positionnement).

### Modifié (Contenu — login ET inscription par défaut en pages « form »)
- `content/site.json` : l'accueil ne lance plus les applets `login`/`register`
  directement ; les entrées 1 et 2 ciblent des **pages dédiées de type `form`** —
  `login` (action `login`, pseudo/mot de passe) et `register` (action `register`,
  pseudo/mot de passe/confirmer), `next: main`. Démontre le modèle déclaratif sur
  le contenu de production. Validé end-to-end (création de compte → compte persisté
  avec hash PBKDF2 ; connexion → accueil personnalisé).

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
