# Connexion matérielle réelle — se brancher au BBS Oric depuis un vrai Oric

> **Sprint 4 — Connexion matérielle réelle.**
> Ce document décrit comment joindre le BBS Oric (`pavi.3617.fr:6502`) depuis un
> **Oric-1 ou Atmos physique** équipé d'une interface série et d'un modem WiFi.
> Le logiciel client (`client/term.s`) est validé **de bout en bout dans
> l'émulateur** ; le test sur **matériel réel** reste à réaliser (voir §7,
> faute de matériel à disposition au moment de l'écriture).

---

## 1. Vue d'ensemble de la chaîne

```
┌──────────────┐   bus     ┌───────────────┐   série    ┌────────────┐   WiFi/TCP   ┌──────────────┐
│   Oric-1 /   │  6502 +   │  Interface    │  TTL/RS232 │  Modem     │ ───────────► │  BBS Oric    │
│   Atmos      │  ACIA ────│  série (ACIA  │ ──────────►│  WiFi      │   telnet     │  pavi.3617   │
│  (term.tap)  │  $031C ou │  6551)        │  9600 8N1  │ (Hayes AT) │   :6502      │  .fr:6502    │
└──────────────┘  $03A0    └───────────────┘            └────────────┘              └──────────────┘
```

L'Oric ne fait **ni TCP, ni WiFi, ni TLS**. Il pilote une **ACIA 6551** (UART)
et envoie des **commandes Hayes AT** sur le port série. C'est le **modem WiFi**
qui ouvre la connexion TCP (et termine le TLS le cas échéant). Côté Oric, tout se
résume à : *écrire des octets sur l'ACIA, en lire en retour*.

---

## 2. Interfaces série supportées (adressage ACIA)

Le client `term.s` cible une **ACIA 6551** via un pointeur runtime `ACIAPTR`. Le
menu de démarrage choisit la base :

| Choix menu | Base ACIA | Montage typique |
|-----------|-----------|-----------------|
| `1` | **`$031C`** | ACIA « standard » Telestrat / défaut `oric1-emu`. Cartes série branchées sur le bus d'extension à cette base. |
| `2` | **`$03A0`** | **LOCI** (carte d'extension Oric moderne) — MIA LOCI mappée `$03A0–$03BF`. |

Les deux montages exposent le **même registre 6551** (offsets depuis la base) :

| Offset | Registre | Usage |
|-------:|----------|-------|
| `+0` | Données | lecture = octet reçu (RX), écriture = octet à émettre (TX) |
| `+1` | Statut | bit `RDRF`=`$08` (donnée reçue dispo), bit `TDRE`=`$10` (émission prête) |
| `+2` | Commande | `$0B` = DTR on, IRQ off, pas d'écho |
| `+3` | Contrôle | `$1E` = **9600 bauds, 8N1** |

> ⚠️ La base `$03A0` du LOCI est **à confirmer sur le matériel réel** (cf.
> `docs/architecture.md` §4 et `ROADMAP.md` « Décisions ouvertes »). Le menu
> permet de basculer sans recompiler ; si aucun des deux ne répond, vérifier le
> brochage et la base de votre carte (voir §6 dépannage).

### Cas non gérés

- **DTL 2000** : modem **V23/Minitel** (6850 + PIA, sans Hayes AT ni TCP). Il ne
  permet pas de joindre un BBS telnet Internet → hors périmètre.

---

## 3. Le modem WiFi

Le BBS a été développé et validé contre un **modem WiFi à firmware Hayes** de type
**Pico W / `picowifi` (firmware v0.2.0)**, équivalent fonctionnel des modems WiFi
rétro répandus (familles « WiFiModem232 », « Tirreno », « RetroWiFiModem »…). Tout
modem exposant un jeu de commandes Hayes AT compatible et un débit série fixé à
**9600 8N1** convient.

### Première configuration (une fois, depuis un terminal AT)

```
AT                      ; doit répondre OK
AT+CWJAP="SSID","mdp"   ; rejoindre le réseau WiFi (selon firmware)
AT&W                    ; sauvegarder la config
```

> La syntaxe exacte d'association WiFi **dépend du firmware** de votre modem
> (`AT+CWJAP`, `ATWIFI`, menu interactif…). Reportez-vous à sa notice. Une fois
> le WiFi mémorisé, le modem se reconnecte seul à l'allumage.

### Réglages série côté modem

- **Débit : 9600 bauds, 8 bits, sans parité, 1 stop (9600 8N1)** — doit
  correspondre exactement au `$1E` programmé dans l'ACIA (§2).
- Contrôle de flux **désactivé** (l'Oric fait du polling simple, pas de RTS/CTS).

---

## 4. Composer un appel (commandes AT émises par l'Oric)

`term.s` **compose lui-même** la commande de numérotation Hayes — l'utilisateur
choisit juste une entrée du répertoire ou saisit hôte/port. Les commandes
réellement émises sur l'ACIA :

| Protocole | Commande émise | Effet |
|-----------|----------------|-------|
| telnet / raw | `ATD<hôte>:<port>` + CR | ouvre une connexion TCP en clair |
| **TLS** | `ATDT#<hôte>:<port>` + CR | le `#` ouvre un appel **TLS terminé par le modem** ; l'Oric reçoit du clair |

Exemples concrets (ce que le modem reçoit) :

```
ATD pavi.3617.fr:6502         ; BBS Oric en clair (telnet)
ATDT# pavi.3617.fr:6992       ; BBS Oric via TLS (le modem déchiffre)
```

Le modem répond `CONNECT` quand le lien est établi, puis le flux BBS (octets
OASCII) circule de façon transparente. À l'écran de l'Oric, les octets de
contrôle 0–31 deviennent des **attributs Téletexte sériels** (couleurs).

### TLS — rappel

L'Oric 8 bits **ne fait aucune crypto**. Le TLS est entièrement géré par le modem :

- `AT$CA` : charge **un** certificat racine (CA) — buffer ~8 Ko (un CA, pas un
  bundle système entier).
- `AT$CV1` : impose la **vérification** du certificat serveur (sinon `VERIFY_NONE`).
- `ATGET https://…` : GET HTTPS direct (port 443) — hors flux BBS.

Validé dans l'émulateur (backend `--serial picowifi`, build OpenSSL) : TLSv1.3,
bannière BBS rendue à travers le tunnel (`docs/img/tls-dial.png`,
`docs/img/tls-verified-atcv1.png`).

---

## 5. Procédure pas à pas (depuis un Oric réel)

1. **Brancher** l'interface série (carte ACIA `$031C` ou LOCI `$03A0`) sur le bus
   d'extension de l'Oric, modem WiFi raccordé au port série, modem sous tension et
   associé au WiFi (§3).
2. **Charger le terminal** `term.tap` :
   - Cassette / lecteur `.tap` : `CLOAD"TERM"` (autorun, le programme démarre seul).
   - Le `.tap` est produit par `client/build.sh` (autorun, chargement `$1000`).
3. **Menu modem** : taper `1` (ACIA `$031C`) ou `2` (LOCI `$03A0`) selon la carte.
4. **Répertoire** : taper le numéro de l'entrée voulue, par ex. `1` =
   `BBS Oric (prod) pavi.3617.fr`, ou `M` pour une **saisie manuelle**
   (hôte, port, protocole telnet/TLS).
5. Le terminal **compose `ATD…`** et affiche « Numérotation en cours… ». Au
   `CONNECT`, la **bannière BBS** s'affiche en couleur.
6. **Naviguer** : les menus se pilotent au clavier (touche unique pour les menus,
   ligne + `RETURN` pour les champs texte).

Équivalent « à la main » (sans `term.s`, pour diagnostic) depuis n'importe quel
terminal AT : `ATD pavi.3617.fr:6502` puis `Entrée`.

---

## 6. Dépannage

| Symptôme | Pistes |
|----------|--------|
| Le menu modem ne répond pas / écran figé | mauvaise base ACIA → essayer l'autre choix (`1`/`2`) ; vérifier la base réelle de la carte. **Émulateur :** ne pas combiner `--loci` avec `--acia-addr 03A0` → la MIA LOCI masque l'ACIA et casse le scan clavier (annuaire gelé). Cf. `phosphoric-findings.md` (F1) ; bonne commande : `--serial picowifi --acia-addr 03A0` **sans** `--loci`. |
| `ATD` sans effet, pas de `CONNECT` | débit série ≠ 9600 8N1 ; modem non associé au WiFi ; hôte/port erronés ; contrôle de flux actif côté modem. |
| Caractères parasites / texte illisible | désaccord de débit (vérifier `$1E` ACIA ↔ 9600 du modem) ; câblage TX/RX inversé. |
| Couleurs absentes (texte blanc seul) | normal sur un terminal générique ; l'Oric rend les attributs sériels en écrivant directement la VRAM (`term.s`). |
| TLS échoue | `AT$CV1` actif sans CA chargé (`AT$CA`) → repasser en `VERIFY_NONE`, ou charger le bon CA ; port TLS = `6992`. |
| Le `#` du TLS n'est pas accepté | firmware modem trop ancien (TLS terminé requiert picowifi v0.2.0+). |

---

## 7. Test sur Oric réel — checklist (à exécuter sur matériel)

> **Statut : en attente de matériel.** Le pipeline est validé dans l'émulateur
> (`scripts/test-emulateur.sh`) ; la checklist ci-dessous est le **protocole de
> recette matérielle** à dérouler dès qu'un Oric physique + interface série +
> modem WiFi sont disponibles. Reporter les résultats (OK/KO + photo) dans
> `docs/img/` et cocher dans `ROADMAP.md`.

- [ ] **T1 — Chargement** : `term.tap` se charge et démarre (menu modem affiché).
- [ ] **T2 — Backend ACIA** : le bon choix (`1`=`$031C` ou `2`=`$03A0`) initialise
      l'ACIA sans blocage.
- [ ] **T3 — Répertoire** : entrée `1` compose `ATD pavi.3617.fr:6502`, modem
      répond `CONNECT`.
- [ ] **T4 — Bannière couleur** : l'écran d'accueil OASCII s'affiche avec les
      bonnes couleurs (jaune/cyan/vert), 40 colonnes respectées (photo).
- [ ] **T5 — Navigation clavier** : menus pilotables (touche unique), champs
      texte (ligne + RETURN), retour menu.
- [ ] **T6 — Saisie manuelle** : `M` → hôte/port/protocole → connexion OK.
- [ ] **T7 — TLS** : entrée `5` (`pavi.3617.fr:6992`) compose `ATDT#`, tunnel TLS
      établi, bannière rendue (photo).
- [ ] **T8 — Déconnexion** : `Q` quitte proprement (« A bientot »), le modem
      raccroche.
- [ ] **T9 — Stabilité** : session de plusieurs minutes sans corruption d'écran
      ni perte de caractères.

---

## Références

- `client/term.s` — terminal 6502 (E/S série, menu, numérotation, mode terminal).
- `client/README.md` — détails build / émulateur.
- `docs/architecture.md` §4 (cibles matérielles), §5 (exposition Internet).
- `docs/oascii.md` — encodage des attributs Téletexte sériels.
- `docs/test-emulateurs.md` — pipeline de test `oric1-emu`.
