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
- [ ] Login / profils utilisateurs (persistance)

## Sprint 3 — Modules de contenu
- [ ] Messagerie / forum (lecture, post)
- [ ] Page d'actualités / annonces
- [ ] Mini-jeu interactif (ex. Puissance 4 / morpion) pour valider l'interactivité

## Sprint 4 — Connexion matérielle réelle
- [ ] Doc de connexion WiFiModem + LOCI (AT, MIA LOCI `$03A0-$03BF` / ACIA `$031C`)
- [ ] Programme client/terminal Oric minimal (BASIC ou cc65) si nécessaire
- [ ] Test sur Oric réel ; écran d'accueil ASCII-art Oric

## Sprint 5 — Déploiement
- [ ] Conteneurisation (Docker) + persistance
- [ ] Exposition publique + monitoring / logs
- [ ] Documentation utilisateur (comment se connecter depuis un Oric)

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
