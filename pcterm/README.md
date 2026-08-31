# oterm — terminal OASCII portable (PC)

Client **terminal** pour le BBS Oric, à lancer sur un PC moderne (**Linux, Windows,
macOS**). Il se connecte à `bbsd` en TCP et rend le flux **OASCII** (mode TEXT 40×28,
attributs Téletexte, couleurs, inverse) dans n'importe quel terminal **ANSI** — y
compris à travers SSH. Binaire **statique unique**, aucune dépendance système.

But : utiliser et tester le BBS **sans matériel Oric ni émulateur**, en complément du
simulateur ULA du studio (aperçu graphique dans le navigateur) et du firmware terminal
réel (`client/`).

## Lancer

```bash
make oterm                       # -> ./oterm
./oterm -addr 127.0.0.1:6502     # se connecte au bbsd local
# ou en prod :
./oterm -addr pavi.3617.fr:6502
```

Raccourci : `make oterm-run ADDR=host:port`.

**Quitter** : `Ctrl-]` (ferme proprement la session).

Le clavier est transmis **caractère par caractère** (modèle d'entrée du BBS, ADR-0002 :
une touche = un choix dans les menus, pas de `Entrée`).

## Portée

- **TEXT complet** : caractères, CR/LF avec défilement, backspace, clamp 40 colonnes,
  commande de positionnement `plot` (`1F col row`) — décodés fidèlement au firmware
  `client/term.s`.
- **Attributs sériels des trois familles** : encre/fond, **texte** (charset alternatif
  « police BBS » + clignotement) et **mode vidéo** (inverse ON/OFF `28`/`29`). Le charset
  BBS (filets `┌─┐│`, blocs `█▌▐▀▄░▒▓`, symboles `•►◄▲▼★…`) est rendu par ses **runes
  Unicode** — fidèle sans rasterisation.
- **HIRES** (`1F FC`) : **rendu réel** en pixels. Le flux d'opcodes (`point/line/box/
  fillbox/circle/char/blit RLE`) est rasterisé dans une VRAM 200×40 puis affiché 240×200
  en **demi-blocs Unicode `▀`** (1 pixel/colonne, 2 pixels/ligne, couleur par pixel via
  les attributs sériels HIRES). `1F FB` rend la main au mode texte.
- **Double hauteur** : approximée (un terminal ne fait pas de demi-hauteur pixel).
- **XMODEM** (`1F FE`/`FD`) : non pris en charge (message d'état) — un client texte ne
  fait pas de transfert de fichiers.

## Architecture

```
pcterm/
  cmd/oterm/          client : connexion TCP + clavier brut + boucle de rendu ANSI
  internal/ula/       décodeur OASCII -> grille 40×28 + curseur, rendu ANSI (pur, testé),
                      charset BBS -> Unicode, bascule mode HIRES
  internal/hires/     rasteriseur du flux HIRES -> image 240×200 couleur, rendu demi-blocs
```

Le décodeur `internal/ula` réutilise les constantes et la sémantique de
`internal/oascii` (partagé avec le serveur et le studio) : **zéro divergence** de format.
Il est **pur** (sans I/O) et couvert par des tests unitaires (`ula_test.go`). Seule
dépendance externe : `golang.org/x/term` (mode brut du clavier, portable multi-OS).
