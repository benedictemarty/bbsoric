# Backlog produit — BBS Oric

> Priorisé. `[ ]` = à faire, `[~]` = en cours, `[x]` = fait. Estimation indicative en points.

## Épopée A — Socle & connexion (Sprint 0–1)

- [x] **A1** (1) En tant qu'équipe, je veux un dépôt versionné et documenté pour tracer le travail.
- [x] **A2** (1) En tant que dev, je veux **confirmer le langage du serveur** (→ **Go 1.26**).
- [x] **A3** (3) En tant qu'utilisateur, je veux me connecter en telnet et voir un écran d'accueil
  (« hello world »), afin de valider la chaîne réseau de bout en bout. *(testé via `nc`)*
- [~] **A4** (2) En tant que dev, je veux tester la connexion **dans un émulateur** sans matériel.
  *(pipeline documenté `docs/test-emulateurs.md` ; test interactif émulateur à dérouler)*

## Épopée B — Rendu OASCII (Sprint 1)

- [x] **B1** (5) En tant qu'utilisateur Oric, je veux des écrans **en couleur** (encre/fond) correctement
  rendus malgré les attributs sériels. *(bannière colorée, flux vérifié au hexdump)*
- [~] **B2** (3) En tant que dev, je veux une **API d'écran** (`ink/paper/blink/text/newline`) qui masque les
  codes d'attribut. *(Builder OASCII livré ; `cls`/positionnement curseur en attente du protocole client)*
- [x] **B3** (2) En tant que dev, je veux une **table d'attributs Oric vérifiée** sur émulateur.
  *(extraite de `oric1-emu` src/video/video.c ; 7 tests unitaires)*

## Épopée C — Moteur BBS (Sprint 2)

- [ ] **C1** (3) En tant qu'utilisateur, je veux **naviguer dans des menus** et revenir en arrière.
- [ ] **C2** (3) En tant que serveur, je veux gérer **plusieurs connexions simultanées** sans blocage.
- [ ] **C3** (2) En tant qu'utilisateur, je veux être **déconnecté proprement** après inactivité.
- [ ] **C4** (3) En tant qu'utilisateur, je veux **m'identifier** et retrouver mon profil.

## Épopée D — Contenu (Sprint 3)

- [ ] **D1** (5) En tant qu'utilisateur, je veux **lire et poster des messages** (forum).
- [ ] **D2** (2) En tant qu'utilisateur, je veux voir des **actualités / annonces**.
- [ ] **D3** (3) En tant qu'utilisateur, je veux jouer à un **mini-jeu** (ex. Puissance 4).

## Épopée E — Réel & déploiement (Sprint 4–5)

- [ ] **E1** (3) En tant qu'utilisateur, je veux une **doc de connexion** WiFiModem + LOCI.
- [ ] **E2** (5) En tant qu'utilisateur, je veux me connecter depuis un **Oric réel**.
- [ ] **E3** (3) En tant qu'admin, je veux **déployer** le serveur (Docker) et le **superviser**.

## Definition of Done (DoD)
- Code versionné, `CHANGELOG.md` et `ROADMAP.md` mis à jour.
- Tests passants pour la fonctionnalité livrée.
- Documentation à jour (`docs/`).
- Validé dans `Oric1/oric1-emu` (Phosphoric) quand applicable.
