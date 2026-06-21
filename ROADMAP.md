# ROADMAP — BBS Oric

Approche **agile**, livraisons incrémentales. Chaque sprint produit un incrément testable.

> **Contrainte transverse : serveur Internet public.** Le BBS est exposé sur Internet (écoute
> `0.0.0.0`, joignable par tout Oric via WiFiModem). Sécurité, exposition et hébergement sont des
> préoccupations de **chaque** sprint, pas seulement du Sprint 5. Voir `docs/architecture.md` §5.

## Sprint 0 — Cadrage & socle ⏳ (en cours)
- [x] État de l'art des serveurs BBS rétro (`docs/etat-de-l-art.md`)
- [x] Cadrage cibles : Oric-1/Atmos + LOCI + WiFiModem ; émulateurs Oricutron + Phosphoror
- [x] Initialisation dépôt Git, documentation agile, CHANGELOG, ROADMAP
- [ ] **DÉCISION** : confirmer le langage du serveur *(en attente de précision — « Autre »)*
- [x] **DÉCISION** : hébergement = **VPS cloud (IP fixe)** ; port public = **6502** (clin d'œil au CPU)
- [ ] Serveur telnet « hello world » écoutant sur `0.0.0.0:6502`, accessible via `nc` / SyncTerm
- [ ] Exposition Internet minimale : rate limiting, limite de connexions, logs de connexion

## Sprint 1 — Couche terminal Oric (« OASCII ») 🎯 cœur du projet
- [ ] Abstraction d'écran : `cls`, positionnement curseur, retour ligne
- [ ] Encodage des **attributs sériels Téletexte** : encre (8), fond (8), clignotement, double hauteur
- [ ] Helpers : `ink()`, `paper()`, `at(x,y)`, `println()`, ASCII-art
- [ ] Validation dans Oricutron (backend ACIA `loopback` puis `modem`)

## Sprint 2 — Moteur BBS
- [ ] Boucle de session multi-clients (1 connexion = 1 tâche)
- [ ] Système de menus / navigation
- [ ] Timeout d'inactivité, déconnexion propre
- [ ] Login / profils utilisateurs (persistance)

## Sprint 3 — Modules de contenu
- [ ] Messagerie / forum (lecture, post)
- [ ] Page d'actualités / annonces
- [ ] Mini-jeu interactif (ex. Puissance 4 / morpion) pour valider l'interactivité

## Sprint 4 — Connexion matérielle réelle
- [ ] Doc de connexion WiFiModem + LOCI (AT, ACIA `0x380`)
- [ ] Programme client/terminal Oric minimal (BASIC ou cc65) si nécessaire
- [ ] Test sur Oric réel ; écran d'accueil ASCII-art Oric

## Sprint 5 — Déploiement
- [ ] Conteneurisation (Docker) + persistance
- [ ] Exposition publique + monitoring / logs
- [ ] Documentation utilisateur (comment se connecter depuis un Oric)

---

## Décisions actées
- **Hébergement** : VPS cloud avec IP fixe (exposition Internet publique 24/7).
- **Port public** : `6502`.

## Décisions ouvertes (ADR à formaliser)
1. **Langage serveur** — « Autre » retenu, techno précise **à confirmer**. *(Sprint 0, bloquant pour A3)*
2. **Adressage ACIA** — supporter `0x380` (LOCI) et `#31C` (Telestrat/Oricutron) côté client.
3. **Protocole telnet** — vrai telnet (NUL strippés) vs « fake telnet » (NUL passés). À trancher au Sprint 1.
