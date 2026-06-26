# Backlog produit — BBS Oric

> Priorisé. `[ ]` = à faire, `[~]` = en cours, `[x]` = fait. Estimation indicative en points.

## Épopée A — Socle & connexion (Sprint 0–1)

- [x] **A1** (1) En tant qu'équipe, je veux un dépôt versionné et documenté pour tracer le travail.
- [x] **A2** (1) En tant que dev, je veux **confirmer le langage du serveur** (→ **Go 1.26**).
- [x] **A3** (3) En tant qu'utilisateur, je veux me connecter en telnet et voir un écran d'accueil
  (« hello world »), afin de valider la chaîne réseau de bout en bout. *(testé via `nc`)*
- [x] **A4** (2) En tant que dev, je veux tester la connexion **dans un émulateur** sans matériel.
  *(terminal `oric-client/term.s` + `scripts/test-emulateur.sh` ; bannière colorée validée à l'écran)*

## Épopée B — Rendu OASCII (Sprint 1)

- [x] **B1** (5) En tant qu'utilisateur Oric, je veux des écrans **en couleur** (encre/fond) correctement
  rendus malgré les attributs sériels. *(bannière colorée validée À L'ÉCRAN dans oric1-emu)*
- [x] **B2** (3) En tant que dev, je veux une **API d'écran** (`ink/paper/blink/text/newline`) qui masque les
  codes d'attribut. *(Builder OASCII ; `cls`/curseur gérés côté terminal Oric par écriture VRAM)*
- [x] **B3** (2) En tant que dev, je veux une **table d'attributs Oric vérifiée** sur émulateur.
  *(extraite de `oric1-emu` src/video/video.c ; 7 tests unitaires)*

## Épopée C — Moteur BBS (Sprint 2)

- [x] **C1** (3) En tant qu'utilisateur, je veux **naviguer dans des menus** et revenir en arrière.
  *(menu principal + 3 écrans, retour via RETURN ; tests Go + validation écran émulateur)*
- [x] **C2** (3) En tant que serveur, je veux gérer **plusieurs connexions simultanées** sans blocage.
  *(1 goroutine/connexion, couche `server`)*
- [x] **C3** (2) En tant qu'utilisateur, je veux être **déconnecté proprement** après inactivité.
  *(idle timeout couche `server`)*
- [x] **C5** (3) En tant qu'utilisateur Oric, je veux **taper au clavier** pour naviguer (TX terminal).
  *(scan matrice complet + écho local + CR ; navigation validée à l'écran via `--type-keys`)*
- [~] **C4** (3) En tant qu'utilisateur, je veux **m'identifier** et retrouver mon profil.
  *(ADR-0001/0002 ; incréments 1–3 livrés : store haché `internal/user`, saisie touche
  unique `ReadKey`, moteur d'applets (type de page `applet`), applets login/register/guest,
  porte d'auth au CONNECT, câblage `-users` + déploiement. Validé end-to-end (`nc`). Reste
  côté client : `term.s` en mode caractère + no-echo mot de passe.)*

## Épopée D — Contenu (Sprint 3)

- [ ] **D1** (5) En tant qu'utilisateur, je veux **lire et poster des messages** (forum).
- [ ] **D2** (2) En tant qu'utilisateur, je veux voir des **actualités / annonces**.
- [ ] **D3** (3) En tant qu'utilisateur, je veux jouer à un **mini-jeu** (ex. Puissance 4).

## Épopée E — Réel & déploiement (Sprint 4–5)

- [x] **E1** (3) En tant qu'utilisateur, je veux une **doc de connexion** WiFiModem + LOCI.
  *(`docs/connexion-materielle.md` : ACIA `$031C`/LOCI `$0380`, AT, 9600 8N1, recette T1–T9)*
- [~] **E2** (5) En tant qu'utilisateur, je veux me connecter depuis un **Oric réel**.
  *(terminal validé dans l'émulateur ; test matériel en attente d'un Oric physique)*
- [x] **E3** (3) En tant qu'admin, je veux **déployer** le serveur (Docker) et le **superviser**.
  *(prod systemd + image Docker ~18 Mo + `/healthz`,`/metrics` + sonde/timer)*
- [x] **E4** (3) En tant qu'admin, je veux **sauvegarder et restaurer l'état** (comptes,
  fichiers, contenu) pour ne rien perdre en cas d'incident.
  *(`scripts/backup.sh`/`restore.sh`, timer quotidien + rotation, à chaud, test e2e
  `scripts/test-backup.sh`, doc `docs/backup.md` ; déploiement via `vps-deploy.sh`)*

## Épopée F — Studio « Forge » (outillage de contenu)

- [x] **F0** (3) En tant qu'équipe, je veux un dépôt en **3 sous-projets** (server/client/studio)
  avec les paquets partagés réutilisables. *(restructuration, ADR-0003)*
- [x] **F1** (5) En tant qu'éditeur, je veux **composer le site.json** (menu/page/applet) avec
  **aperçu couleur** et validation. *(forge web Go, internal/content réutilisé)*
- [x] **F2** (5) En tant qu'admin, je veux **déployer le contenu** sur **dev/int/prod** via des
  **profils** (valide→sauvegarde→écrase→reload, dry-run). *(validé end-to-end)*
- [ ] **F3** (3) En tant qu'éditeur, je veux **créer/gérer plusieurs sites** et leurs sauvegardes
  depuis l'UI.

## Épopée G — Transfert de fichiers (étude, non planifié)

- [~] **G1** (8) En tant qu'utilisateur, je veux **télécharger/téléverser** des fichiers via le BBS.
  *Côté **serveur fait** (`internal/xmodem`, `server/internal/files`, applets
  `download`/`upload`, `Session.Raw()`, flags `-files`/`-max-upload`, studio, doc
  `docs/transfert.md`). **Download ET upload Oric faits** : récepteur (checksum) +
  émetteur (CRC-16) XMODEM 6502 (`client/xmodem.s`), déclenchés par `1F FE`/`1F FD`,
  buffer RAM `$4000` — validés émulateur (`docs/img/xmodem-download.png`,
  `xmodem-upload.png`). **Reste** : **stockage** carte SD (LOCI)/Microdisc/cassette
  (buffer en RAM pour l'instant).*
  - **Voie B (Sedoric) — ✅ sauvegarde ML VALIDÉE sur V3.0 (24/06)** : écriture
    disquette prouvée (flag `--disk-writeback`) ; recette ML validée end-to-end
    (`JSR $04F2` overlay → variables → `JSR $DE9C` XSAVEB → `JSR $04F2`), un
    fichier écrit/persisté dans la `.dsk`. `client/sedoric.s` finalisée. **Reste**
    l'intégration `term.s` (déclencher après un download) + déploiement du
    terminal sous Sedoric résident. Voir `docs/sedoric-api.md`.

## Definition of Done (DoD)
- Code versionné, `CHANGELOG.md` et `ROADMAP.md` mis à jour.
- Tests passants pour la fonctionnalité livrée.
- Documentation à jour (`docs/`).
- Validé dans `Oric1/oric1-emu` (Phosphoric) quand applicable.
