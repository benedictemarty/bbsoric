# Terminal BBS autonome pour Oric (`term.s`)

Programme 6502 côté Oric qui transforme la machine en **terminal BBS autonome** :
au démarrage il choisit le modem, affiche un **répertoire**, **compose lui-même**
la commande de numérotation Hayes (`ATD`), puis bascule en terminal. La réception
est écrite **directement en mémoire écran** (`$BB80`) pour que les octets de
contrôle 0–31 deviennent de vrais **attributs Téletexte sériels** (couleurs OASCII).

## Déroulé

1. **Menu modem** : `1` = ACIA 6551 direct (`$031C`), `2` = LOCI / Pico W (`$03A0`).
2. **Répertoire** : BBS Oric (prod), ParticlesBBS, Altair, Heatwave, ou `M` = saisie manuelle.
3. **Saisie manuelle** : hôte, port, protocole (`1`=telnet/raw, `2`=TLS).
4. Numérotation `ATD<hôte:port>` autonome → **mode terminal**.

## Caractéristiques techniques

- **E/S série abstraites** via `ACIAPTR` (pointeur ZP sur la base ACIA) + primitives
  `ser_tx`/`ser_rx_ready`/`ser_rx` → un seul binaire pour les 2 backends 6551 (adresses
  `$031C` et `$03A0`, validées end-to-end : `CONNECT to pavi.3617.fr:6502`).
- ACIA 9600 8N1, DTR on, polling (pas d'IRQ).
- Réception → écran : `CR`/`LF`/scroll, clamp 40 colonnes.
- **TX clavier** : scan matrice 8×8 (PSG-via-VIA), anti-rebond, écho local. `input_line`
  pour la saisie manuelle.
- Chargé/exécuté en `$1000`. ~1,5 Ko.

## Modems et protocoles

- **DTL 2000** non géré : c'est un modem **V23/Minitel** (6850 + PIA, pas de Hayes AT ni
  de TCP moderne) — il ne sert pas à joindre un BBS telnet Internet.
- **TLS/SSL** : l'Oric 8 bits **ne fait aucune crypto**. Le TLS est terminé par le **modem
  WiFi** (Pico W). Côté Oric, « protocole » ne fait que choisir la commande envoyée :
  telnet/raw → `ATD hôte:port` (fonctionnel) ; TLS → commande sécurisée spécifique au
  Pico W (matériel réel uniquement ; le backend `modem` de l'émulateur ne fait que du TCP).

## Construire

```bash
./build.sh        # xa term.s -> term.bin -> bin2tap -> term.tap (autorun)
```
Surcharge : `BIN2TAP=/chemin/bin2tap ./build.sh`.

## Tester dans l'émulateur

Depuis la racine du dépôt :
```bash
oric-client/build.sh
scripts/test-emulateur.sh /tmp/oric.ppm
```
Le script lance le serveur, démarre `oric1-emu` connecté en série TCP, et capture
le rendu. Résultat de référence : [`../docs/img/sprint1-banner.png`](../docs/img/sprint1-banner.png).

## Notes techniques

- `xa` ne supporte pas l'UTF-8 ni le `:` dans les commentaires → commentaires ASCII.
- Adresse ACIA choisie au runtime via le menu modem (`ACIAPTR` = `$031C` ou `$03A0`).
- Octets `$0A`/`$0D` réservés au contrôle de ligne : la couche OASCII évite de les
  émettre comme attributs (les attributs 0x0A/0x0D — double hauteur seule / blink+alt —
  sont à éviter dans le protocole actuel).
- **Test `--type-keys`** : l'outil maintient une touche enfoncée jusqu'à une touche
  identique ou la fin de chaîne → la navigation multi-écrans s'automatise mal (doubler la
  touche force un relâchement). Chaque étape a été validée séparément ; en frappe humaine
  (presser/relâcher) tout s'enchaîne normalement.
