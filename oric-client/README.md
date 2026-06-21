# Terminal Oric minimal (`term.s`)

Programme 6502 côté Oric qui transforme la machine en **terminal série** pour le
BBS : il lit le flux de l'ACIA 6551 et l'écrit **directement en mémoire écran**
(`$BB80`), de sorte que les octets de contrôle 0–31 deviennent de vrais
**attributs Téletexte sériels** (couleurs OASCII) au lieu d'être interprétés par
les routines ROM.

## Caractéristiques

- Init ACIA `$031C` : 9600 8N1, DTR on, polling (pas d'IRQ).
- Réception → écran : gère `CR` ($0D, retour début de ligne), `LF` ($0A, ligne
  suivante + scroll), clamp à 40 colonnes.
- **TX clavier** : scan complet de la matrice 8×8 (protocole PSG-via-VIA), table ASCII
  par position, anti-rebond (1 caractère par appui), envoi ACIA + **écho local**.
- Chargé/exécuté en `$1000`. ~490 octets.

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
- Adresse ACIA : `$031C` (défaut oric1-emu / Telestrat). Avec LOCI : MIA `$03A0-$03BF`
  (adapter la constante `ACIA_DATA`).
- Octets `$0A`/`$0D` réservés au contrôle de ligne : la couche OASCII évite de les
  émettre comme attributs (les attributs 0x0A/0x0D — double hauteur seule / blink+alt —
  sont à éviter dans le protocole actuel).
