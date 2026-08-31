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
- **HIRES** (`1F FC`) : **rendu réel** en pixels, **composé à l'identique de l'écran Oric**.
  Le flux d'opcodes (`point/line/box/fillbox/circle/char/blit RLE`) est rasterisé dans une
  VRAM 200×40 puis affiché 240×200 en **demi-blocs Unicode `▀`** (1 pixel/colonne, 2 pixels/
  ligne, couleur par pixel via les attributs sériels HIRES) **suivi des 3 dernières lignes
  de la grille texte** (rows 25–27) qui restent en TEXT sous le décor — exactement comme sur
  l'Oric (200 px graphiques + 3 lignes texte). `1F FB` rend la main au mode texte.
  *(Note : le contenu de prod actuel n'envoie pas encore de menu dans ces 3 lignes ; oterm
  les compose donc vides, comme le ferait le firmware — et les affiche fidèlement dès qu'une
  page les remplit via un plot en row 25–27.)*
- **Téléchargement XMODEM** (`1F FE`) : **pris en charge**. Quand le BBS envoie un fichier
  (menu Fichiers, catalogue), `oterm` déroule le transfert et enregistre le fichier sous son
  nom réel dans le répertoire `-dl` (défaut `.`). L'**upload** demandé par le serveur
  (`1F FD`) est **annulé proprement** (un client texte ne choisit pas de fichier local).
- **Double hauteur** : approximée (un terminal ne fait pas de demi-hauteur pixel).

## Architecture

```
pcterm/
  cmd/oterm/          client : connexion TCP + clavier brut + boucle de rendu ANSI + transferts
  internal/ula/       décodeur OASCII -> grille 40×28 + curseur, rendu ANSI (pur, testé),
                      charset BBS -> Unicode, bascule mode HIRES, détection de transfert
  internal/hires/     rasteriseur du flux HIRES -> image 240×200 couleur, rendu demi-blocs
  internal/xfer/      transferts XMODEM (download : en-tête + réception + écriture fichier)
```

Le décodeur `internal/ula` réutilise les constantes et la sémantique de
`internal/oascii` (partagé avec le serveur et le studio) : **zéro divergence** de format.
Il est **pur** (sans I/O) et couvert par des tests unitaires (`ula_test.go`). Seule
dépendance externe : `golang.org/x/term` (mode brut du clavier, portable multi-OS).

### Fidélité HIRES vérifiée

Le rasteriseur `internal/hires` est prouvé **identique au client Oric** à deux niveaux :
- **pixel-exact** contre une référence indépendante (port Go du rasteriseur JS du studio) —
  0 pixel divergent sur toutes les primitives, 720 cas de lignes (`ref_test.go`) ;
- **VRAM byte-exacte** contre le **vrai firmware** exécuté dans l'émulateur `oric1-emu` —
  7999/8000 octets identiques (`scripts/test-emulateur-hires.sh` + `emu_test.go`), la seule
  différence étant l'attribut de bascule mode ULA à `$BB80`, artefact firmware hors dessin.
