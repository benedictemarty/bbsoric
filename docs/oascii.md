# Couche OASCII — affichage Oric

## Qu'est-ce qu'« OASCII » ?

Nom de marque du projet (clin d'œil à **PETSCII** et **ATASCII**), MAIS attention à
la différence de fond :

| Machine | « xASCII » | Nature réelle |
|---------|-----------|---------------|
| C64 | PETSCII | jeu de caractères **propriétaire** (codes ≠ ASCII) |
| Atari | ATASCII | jeu de caractères **propriétaire** (codes ≠ ASCII) |
| **Oric** | **OASCII** | **ASCII standard** pour les caractères + **attributs Téletexte sériels** |

Sur l'Oric, la lettre `A` est au code `65` comme en ASCII. « OASCII » ne désigne donc
**pas** un encodage de caractères, mais le **modèle d'affichage** de l'Oric : le mode
TEXT 40×28 de type Téletexte, où couleurs et attributs sont posés par des **octets de
contrôle (0–31) qui occupent une case écran** et s'appliquent jusqu'à la fin de la ligne.

## Table d'attributs (source de vérité)

Extraite du décodeur ULA de l'émulateur de référence **`Oric1/oric1-emu`**
(`src/video/video.c`, fonction `decode_attr`). Un octet écran est un attribut sériel
si ses bits 6 et 5 sont nuls (valeur 0–31) ; effet selon `val & 0x18` :

| Octet | Groupe | Effet |
|-------|--------|-------|
| `0–7`   (`0x00`) | **INK** (encre) | encre = `val & 7` |
| `8–15`  (`0x08`) | **attributs texte** | bit0 = charset alternatif · bit1 = double hauteur · bit2 = clignotement |
| `16–23` (`0x10`) | **PAPER** (fond) | fond = `val & 7` |
| `24–31` (`0x18`) | **mode vidéo** | change `vid_mode` ; en plus : `28` = inverse OFF, `29` = inverse ON |

> L'ULA **réinitialise** les attributs au début de **chaque ligne** : encre = blanc (7),
> fond = noir (0). Une couleur ne « déborde » donc pas sur la ligne suivante.

## Positionnement du curseur (« plot X,Y »)

Au-delà du flux séquentiel, une **extension propre au terminal Oric** permet de
positionner le curseur d'écriture à des coordonnées absolues :

| Séquence | Effet |
|----------|-------|
| `1F` `col` `row` | place le curseur en (`col` 0–39, `row` 0–27) ; les octets suivants s'écrivent à partir de là |

L'octet `0x1F` (hors des plages d'attributs réellement émises) est suivi de
**deux octets bruts** (colonne puis ligne). Le terminal (`client/term.s`,
`handle_rx`/`set_cursor_xy`) intercepte la séquence et repositionne son pointeur
VRAM. API Go : `oascii.Plot(col, row)` ou `Builder.At(col, row)`. Les terminaux
génériques (telnet/PC) ne comprennent pas cette commande — c'est une fonction Oric
(utilisée p. ex. pour positionner les champs d'un formulaire dans un décor).

## Palette (8 couleurs, bits R/G/B)

Depuis `palette[8][3]` de `video.c` :

| # | Nom | RGB |
|---|-----|-----|
| 0 | Black   | `000000` |
| 1 | Red     | `FF0000` |
| 2 | Green   | `00FF00` |
| 3 | Yellow  | `FFFF00` |
| 4 | Blue    | `0000FF` |
| 5 | Magenta | `FF00FF` |
| 6 | Cyan    | `00FFFF` |
| 7 | White   | `FFFFFF` |

## API Go (`internal/oascii`)

```go
b := oascii.New()                  // état Oric par défaut : encre blanc, fond noir
b.Ink(oascii.Yellow)               // émet l'octet 0x03
b.Paper(oascii.Blue)               // émet l'octet 0x14 (16+4)
b.Blink(true)                      // émet l'octet 0x0C
b.Text("BBS ORIC")                 // ASCII imprimable (0–31 → espace de sécurité)
b.Newline()                        // CR LF (+ ré-émission si Sticky)
sess.Write(b.String())             // octets 0–31 préservés
```

Encodeurs bas niveau testés contre l'émulateur : `InkAttr(c)`, `PaperAttr(c)`,
`TextAttr(blink, doubleHeight, altCharset)`.

### Mode « sticky »
`b.Sticky(true)` ré-émet automatiquement les attributs courants (non par défaut)
après chaque `Newline()`, pour conserver une couleur sur plusieurs lignes malgré la
réinitialisation par ligne de l'ULA. Coût : chaque ré-émission consomme une case en
début de ligne.

## Pièges de mise en page (40 colonnes)

- **Un octet d'attribut occupe une case.** Une ligne « pleine largeur » de 40 caractères
  **précédée** d'un attribut fait 41 cases → elle déborde sur la ligne suivante. Pour une
  ligne pleine largeur colorée, n'utiliser que `Cols-1` caractères, ou la laisser en
  couleur par défaut (aucun octet émis).
- Le centrage doit tenir compte des octets d'attribut placés avant le texte (décalage
  d'une colonne par attribut).

## Validation sur émulateur

```bash
go run ./cmd/bbsd -addr 127.0.0.1:6502
cd ~/Oric1 && ./oric1-emu --serial tcp:127.0.0.1:6502 --acia-addr 031C
```
Voir [`test-emulateurs.md`](test-emulateurs.md). Le hexdump du flux serveur permet de
vérifier les octets d'attribut (ex. `03` = encre jaune avant le titre).
