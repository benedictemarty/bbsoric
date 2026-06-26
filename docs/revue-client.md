# Revue du client (terminal Oric) — 26/06/2026

Revue de niveau ingénieur du terminal 6502 (`client/term.s`, `xmodem.s`,
`sedoric.s`, `altcharset.s`) et suites données. Sévérité : 🔴 élevé, 🟠 moyen,
🟡 faible.

## Résolus

| # | Sév | Point | Correctif |
|---|-----|-------|-----------|
| LOCI | 🔴 | Option « 2 = LOCI » visait `$03A0` (espace **MIA**), pas le modem → collision MIA/ACIA, PSG perturbé, **clavier figé sur l'annuaire** | `mm_loci` pointe sur **`$0380`** (ACIA du modem WiFi LOCI, cf. firmware `PicoWiFiModemUSB`). Validé `--loci --serial picowifi` : `2`→`1`→`CONNECT`. Cf. `phosphoric-findings.md` F1. |
| 2 | 🔴 | **Plot hors limites** : `set_cursor_xy` (`1F col row`) sans borne → écriture hors VRAM depuis entrée réseau non fiable (BBS tiers) | Clamp `row<28`, `col<40` avant calcul d'adresse (`term.s set_cursor_xy`). |
| 3 | 🔴 | **Réception XMODEM non bornée** : écriture à partir de `$4000` sans plafond → débordement (écran, ROM) depuis le réseau | Refus si tampon ≥ `$B800` : `CAN` + message « FICHIER TROP GROS » (`xmodem.s xr_block`). |
| 5a | 🔴 | **Pas de majuscules** : `asciitab` minuscules seules, aucun SHIFT → mots de passe à casse mixte inconnectables | `scan_shift` (lit LSHIFT col4/row4, RSHIFT col7/row4) ; `key_scan` passe `a-z`→`A-Z` si SHIFT. Validé émulateur (`\L` → TX `$59 'Y'`, `$5A 'Z'`). |
| 5b | 🔴 | **Pas de backspace** : `input_line` ignore `<$20` ; `ReadLine` serveur n'efface pas → impossible de corriger une saisie | Touche **DEL** (col5/row5) → `$08` ; `putbyte` gère `$08` (effacement destructif) ; `input_line` et serveur `ReadLine` retirent le dernier caractère (`$08`/`$7F`). Test serveur `TestReadLineBackspace`. |
| 11 | 🟡 | `sei` permanent non expliqué | Commentaire ajouté (terminal bare-metal, clavier+série en propre ; Sedoric re-SEI). |
| 10 | 🟡 | Allocation zero-page documentée en prose, non centralisée | Carte ZP ajoutée en tête de `term.s` (+ `SHIFTF=$F3`). |

## Correction de la revue

- **#4 (chat invisible pendant la frappe)** : **infirmé**. La boucle `main`
  entrelace déjà RX (rendu) et scan clavier (1 touche/itération) ; les messages
  poussés par le serveur **s'affichent** bien pendant que l'utilisateur tape en
  session. Le blocage `get_key` ne concerne que les **menus pré-connexion**
  (modem, annuaire) où aucune donnée série n'arrive — acceptable.

## Différés (structurels / à valider sur matériel) — avec justification

| # | Sév | Point | Pourquoi différé |
|---|-----|-------|------------------|
| 1 | 🔴 | **Perte d'octets RX pendant un `scroll_up`** (memmove ~1 Ko ≈ plusieurs octets perdus à 9600 bauds) ; pas de contrôle de flux | Vrai correctif = **RTS/CTS** ou XON/XOFF + pacing serveur, à valider sur **matériel réel** (le 6551 émulé + `--serial-buffer` masquent le défaut). À traiter avant usage fer. Risque trop élevé sans HW. |
| 6 | 🟠 | Pas de lecture des **codes modem** (`CONNECT`/`NO CARRIER`) ni du **DCD** → « ça a l'air figé » si la connexion échoue, pas de détection de raccrochage | Demande un mini-analyseur de réponses AT + surveillance DCD ; fonctionnalité à part entière, à concevoir (pas un simple correctif). |
| 7 | 🟠 | Pas de **filtrage telnet IAC** côté client → les BBS tiers qui négocient affichent des caractères de contrôle | Le BBS Oric n'émet **aucun** IAC. Un parser telnet **partiel** (sans sous-négociation SB) serait **pire** que la limitation documentée. Le telnet complet est une **feature**, pas un correctif. |
| 8 | 🟠 | Bits d'erreur ACIA (overrun/framing) jamais lus → perte silencieuse | Lié à #1 ; sans contrôle de flux, lire l'overrun n'apporte pas de récupération. À traiter avec #1. |
| 9 | 🟠 | Sauvegarde Sedoric mono-fichier (`BBSFILE.BIN`, écrase) | Acceptable en alpha ; un nommage dérivé du transfert demande un protocole (le serveur n'envoie pas de nom). |
| 12 | 🟡 | Couverture de test du client faible (smoke `test-emulateur.sh` fragile, basé sur compteur de cycles) | Le scan clavier + SHIFT a été **validé end-to-end** dans l'émulateur (trace série). Un test automatisé multi-étapes (saisie manuelle) reste fragile via `--type-keys` ; backlog. |

## Validation (cette itération)

- `make client` : assemblé (3876 o). `.dsk` reconstruit.
- Émulateur : LOCI `$0380` `2`→`1`→`CONNECT` (bannière rendue) ; SHIFT `\L` → TX
  majuscules ; rendu normal non régressé (clamp plot OK sur coords valides).
- Serveur : `go test -race ./...` vert ; `TestReadLineBackspace` (4 cas) vert.
