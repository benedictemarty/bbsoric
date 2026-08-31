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

## Portée et limites

- **TEXT complet** : caractères, CR/LF avec défilement, backspace, clamp 40 colonnes,
  commande de positionnement `plot` (`1F col row`), attributs sériels encre/fond/inverse
  et clignotement — décodés fidèlement au firmware `client/term.s` et à l'ULA.
- **Semi-graphiques / double hauteur** : approximés (un terminal texte ne rend pas la
  police BBS ni les demi-hauteurs pixel — pour un rendu pixel exact, voir le simulateur
  ULA du studio).
- **HIRES** (`1F FC`) et **XMODEM** (`1F FE`/`FD`) : **non rendus** par un client texte ;
  `oterm` affiche un message d'état et poursuit (le flux HIRES est avalé jusqu'à `1F FB`).

## Architecture

```
pcterm/
  cmd/oterm/          client : connexion TCP + clavier brut + boucle de rendu ANSI
  internal/ula/       décodeur OASCII -> grille 40×28 + curseur, rendu ANSI (pur, testé)
```

Le décodeur `internal/ula` réutilise les constantes et la sémantique de
`internal/oascii` (partagé avec le serveur et le studio) : **zéro divergence** de format.
Il est **pur** (sans I/O) et couvert par des tests unitaires (`ula_test.go`). Seule
dépendance externe : `golang.org/x/term` (mode brut du clavier, portable multi-OS).
