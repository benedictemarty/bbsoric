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
| **Client** | Oric-1 / Atmos + [LOCI](https://www.raxiss.com/article/id/38-LOCI) + WiFiModem USB (ACIA modem @ `0x380`) |
| **Émulateurs de test** | [Oricutron](https://github.com/pete-gordon/oricutron) (backend ACIA `modem`/`loopback`), Phosphoror |
| **Serveur** | Voir [`docs/architecture.md`](docs/architecture.md) — langage à confirmer (recommandation : Python 3 / asyncio) |

## État du projet

🚧 **Sprint 0 — Cadrage & socle.** Voir [`ROADMAP.md`](ROADMAP.md) et [`docs/agile/backlog.md`](docs/agile/backlog.md).

## Pourquoi c'est différent d'un BBS C64/Atari

L'Oric n'utilise **pas** PETSCII. Son mode TEXT est proche du **Téletexte/Minitel** : les couleurs et
attributs (encre, fond, clignotement, double hauteur) sont posés par des **codes de contrôle qui
occupent une case écran** (attributs sériels). Le cœur technique du projet est cette couche de rendu
« OASCII » — voir [`docs/architecture.md`](docs/architecture.md).

## Documentation

- [`ROADMAP.md`](ROADMAP.md) — plan par sprints
- [`CHANGELOG.md`](CHANGELOG.md) — historique des modifications
- [`docs/architecture.md`](docs/architecture.md) — architecture technique & spécificités Oric
- [`docs/etat-de-l-art.md`](docs/etat-de-l-art.md) — analyse des serveurs BBS rétro existants
- [`docs/agile/backlog.md`](docs/agile/backlog.md) — backlog produit (user stories)
