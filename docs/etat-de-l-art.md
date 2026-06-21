# État de l'art — Serveurs BBS rétro

> Synthèse réalisée au Sprint 0 pour cadrer le projet BBS Oric.

## 1. Le modèle « serveur moderne + client 8 bits »

Les BBS rétro actuels (à la PETSCII BBS / ATASCII) ne tournent **pas** sur la machine d'époque :
un **serveur moderne** (PC / Raspberry) écoute en **TCP/telnet**, et la machine 8 bits s'y connecte
via un **modem WiFi** (ESP8266 ou Pico W, commandes Hayes AT) branché sur une interface série.
Le serveur produit des écrans dans le jeu de caractères natif de la machine.

## 2. Projets de référence

| Projet | Stack | Cible | Apport |
|--------|-------|-------|--------|
| [sblendorio/petscii-bbs](https://github.com/sblendorio/petscii-bbs) | Java 21 | C64/128, ASCII, **Minitel/Videotex**, Prestel | **Référence d'architecture.** 1 thread/connexion, classes `PetsciiThread`/`AsciiThread` + `doLoop()`, API `print/readKey/cls`. Le support Videotex est très proche du besoin Oric (Téletexte). |
| [retrocomputacion/retrobbs](https://github.com/retrocomputacion/retrobbs) | Python 3 | C64/Plus4/MSX (Turbo56K) | Multimédia, **encodage neutre** depuis v0.50 → bon modèle d'abstraction terminal. |
| [Magnetar-BBS](https://github.com/Commodore64HomeBrew/Magnetar-BBS) | Natif 6502 | C64 | BBS qui tourne *sur* la machine (RR-NET/SD2IEC). Philosophie opposée à la nôtre. |
| [TheOldNet BBS](https://github.com/TheOldNet/theoldnet-bbs) | Java (fork petscii-bbs) | C64 | Exemple de production (navigation web en PETSCII). |

## 3. Écosystème de connexion

- **Modems WiFi** : [RetroWiFiModem](https://github.com/mecparts/RetroWiFiModem) (ESP8266),
  [PicoWiFiModem](https://github.com/mecparts/PicoWiFiModem),
  [PicoWiFiModemUSB](https://github.com/sodiumlb/PicoWiFiModemUSB) (mentionne explicitement **LOCI / Oric**).
- **LOCI** ([Raxiss](https://www.raxiss.com/article/id/38-LOCI)) : extension bus Oric-1/Atmos, émulation
  ROM/floppy + **USB CDC modem exposé en ACIA à l'adresse `0x380`**. C'est notre voie d'entrée série
  sur Oric-1/Atmos (qui n'ont pas de RS232 natif, contrairement au Telestrat / ACIA 6551).
- **Émulateurs** :
  - [Oricutron](https://github.com/pete-gordon/oricutron) — backend ACIA configurable :
    `none`, `loopback` (test), `com` (port série réel/virtuel), `modem` (sockets + AT + telnet port 23).
    ACIA émulé à `#31C` (adresse Telestrat). Permet un **pipeline de test 100% logiciel**.
  - Phosphoror — émulateur Oric (test croisé).

## 4. Le cas Oric : ce qui n'existe pas encore

Aucun BBS Oric clé en main repéré. Les différences clés vs C64/Atari :

1. **Pas de PETSCII.** Jeu de caractères proche ASCII, mais l'affichage **TEXT 40×28 est de type Téletexte** :
   couleurs/attributs posés par des **codes de contrôle occupant une case écran** (attributs sériels).
   → Le rendu doit être conçu comme du **Téletexte/Videotex**, pas comme un terminal ANSI classique.
2. **Série non native** sur Oric-1/Atmos (extension requise : LOCI, Microdisc+, etc.).
3. **Modèle d'écran contraint** : 40 colonnes, attributs qui « consomment » des colonnes → la mise en page
   doit anticiper l'espace pris par les codes couleur.

## 5. Implications pour notre architecture

- S'inspirer du modèle **1 connexion = 1 tâche** + API d'écran de petscii-bbs.
- Construire une couche **« OASCII »** dédiée aux attributs Téletexte sériels Oric (cœur du Sprint 1).
- Tester d'abord **dans Oricutron** (backend `modem`/`loopback`) avant tout matériel.

## Sources
- https://github.com/sblendorio/petscii-bbs
- https://github.com/retrocomputacion/retrobbs
- https://github.com/Commodore64HomeBrew/Magnetar-BBS
- https://github.com/TheOldNet/theoldnet-bbs
- https://www.raxiss.com/article/id/38-LOCI
- https://github.com/sodiumlb/PicoWiFiModemUSB
- https://github.com/pete-gordon/oricutron
- https://forum.defence-force.org/viewtopic.php?t=1138
- https://wiki.defence-force.org/doku.php?id=oric:hardware:serial
