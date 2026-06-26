# BBS Oric

Serveur **BBS** (Bulletin Board System) pour les ordinateurs **Oric** (Oric-1 / Atmos / Telestrat),
dans l'esprit des serveurs rétro modernes comme [PETSCII BBS](https://github.com/sblendorio/petscii-bbs)
(Commodore) ou des BBS ATASCII (Atari).

Le serveur tourne sur une machine moderne (PC / Raspberry Pi), écoute en **TCP/telnet**, et génère
des écrans dans le jeu de caractères natif de l'Oric (mode TEXT 40×28 façon Téletexte, attributs
sériels de couleur). La machine Oric s'y connecte via un **modem WiFi** relié à une interface série.

## Cibles

| Élément | Choix |
|---------|-------|
| **Client** | Oric-1 / Atmos + [LOCI](https://www.raxiss.com/article/id/38-LOCI) + WiFiModem USB (ACIA série ; MIA LOCI `$03A0-$03BF`, ACIA standard `$031C`) |
| **Émulateur de test** | `Oric1/oric1-emu` (Phosphoric) **uniquement** — `--serial tcp:` vers le BBS. Voir [`docs/test-emulateurs.md`](docs/test-emulateurs.md) |
| **Serveur** | **Go** (`server/cmd/bbsd`) — voir [`docs/architecture.md`](docs/architecture.md) |

## État du projet

🟢 **EN PRODUCTION** — accessible en telnet sur **`pavi.3617.fr:6502`**.
Depuis un Oric (modem WiFi) : `ATD pavi.3617.fr:6502`.

Sprints 0→2 bouclés (socle réseau + couche OASCII + moteur de menus + terminal Oric RX/TX),
déployé via `make deploy`. Voir [`ROADMAP.md`](ROADMAP.md) et [`docs/agile/backlog.md`](docs/agile/backlog.md).

## Terminal Oric (téléchargement)

Le terminal Oric est distribué via les **[Releases GitHub](../../releases)** — les
images ne sont **pas** versionnées dans le dépôt (artefacts régénérables, voir
`.gitignore`). Dernière version : **[`v0.1.0-alpha`](../../releases/tag/v0.1.0-alpha)** :

| Fichier | Quoi |
|---------|------|
| **`term.tap`** | Image cassette autorun (`$1000`) — pour émulateur ou interface cassette |
| **`term-boot.dsk`** | Disquette Sedoric bootable (Microdisc) : boot puis `LOAD"TERM":CALL#1000` |

Ou reconstruire depuis les sources :

```bash
make client              # -> client/term.tap   (nécessite xa65)
client/build-disk.sh     # -> client/term-boot.dsk  (nécessite l'émulateur + ROM Microdisc + master Sedoric)
```

Lancer dans l'émulateur (ACIA `$031C`, ou LOCI `$0380` — voir
[`docs/connexion-materielle.md`](docs/connexion-materielle.md)) :

```bash
# ACIA standard $031C (menu modem : choix 1)
oric1-emu -t client/term.tap -f -r basic11b.rom \
  --serial modem:pavi.3617.fr:6502 --serial-buffer 512

# LOCI / Pico WiFi $0380 (menu modem : choix 2) — modèle fidèle, SANS --acia-addr
oric1-emu -t client/term.tap -f -r basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

## Déploiement

```bash
cp deploy/deploy.conf.example deploy/deploy.conf   # puis renseigner (gitignoré)
make deploy                                         # compile (linux/amd64) + service systemd bbsoric (6502)
```
Détails dans [`deploy/`](deploy/). `deploy.conf` (infra réelle) n'est pas versionné.

## Pourquoi c'est différent d'un BBS C64/Atari

L'Oric n'utilise **pas** PETSCII. Son mode TEXT est proche du **Téletexte/Minitel** : les couleurs et
attributs (encre, fond, clignotement, double hauteur) sont posés par des **codes de contrôle qui
occupent une case écran** (attributs sériels). Le cœur technique du projet est cette couche de rendu
« OASCII » — voir [`docs/architecture.md`](docs/architecture.md).

## Documentation

- [`ROADMAP.md`](ROADMAP.md) — plan par sprints
- [`CHANGELOG.md`](CHANGELOG.md) — historique des modifications
- [`docs/architecture.md`](docs/architecture.md) — architecture technique & spécificités Oric
- [`docs/oascii.md`](docs/oascii.md) — couche d'affichage OASCII (attributs Téletexte, palette, API)
- [`docs/etat-de-l-art.md`](docs/etat-de-l-art.md) — analyse des serveurs BBS rétro existants
- [`docs/agile/backlog.md`](docs/agile/backlog.md) — backlog produit (user stories)
