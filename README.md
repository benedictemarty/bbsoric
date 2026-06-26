# BBS Oric

**BBS** (Bulletin Board System) server for **Oric** computers (Oric-1 / Atmos / Telestrat),
in the spirit of modern retro servers such as [PETSCII BBS](https://github.com/sblendorio/petscii-bbs)
(Commodore) or ATASCII BBSes (Atari).

The server runs on a modern machine (PC / Raspberry Pi), listens over **TCP/telnet**, and generates
screens in the Oric's native character set (TEXT mode 40×28, Teletext-style, serial colour
attributes). The Oric machine connects to it via a **WiFi modem** wired to a serial interface.

## Targets

| Item | Choice |
|---------|-------|
| **Client** | Oric-1 / Atmos + [LOCI](https://www.raxiss.com/article/id/38-LOCI) + WiFiModem USB (serial ACIA; MIA LOCI `$03A0-$03BF`, standard ACIA `$031C`) |
| **Test emulator** | `Oric1/oric1-emu` (Phosphoric) **only** — `--serial tcp:` to the BBS. See [`docs/emulator-testing.md`](docs/emulator-testing.md) |
| **Server** | **Go** (`server/cmd/bbsd`) — see [`docs/architecture.md`](docs/architecture.md) |

## Project status

🟢 **IN PRODUCTION** — reachable over telnet at **`pavi.3617.fr:6502`**.
From an Oric (WiFi modem): `ATD pavi.3617.fr:6502`.

Sprints 0→2 completed (network foundation + OASCII layer + menu engine + Oric RX/TX terminal),
deployed via `make deploy`. See [`ROADMAP.md`](ROADMAP.md) and [`docs/agile/backlog.md`](docs/agile/backlog.md).

## Oric terminal (download)

The Oric terminal is distributed via the **[GitHub Releases](../../releases)** — the
images are **not** versioned in the repository (regenerable artefacts, see
`.gitignore`). Latest version: **[`v0.1.0-alpha`](../../releases/tag/v0.1.0-alpha)**:

| File | What |
|---------|------|
| **`term.tap`** | Autorun cassette image (`$1000`) — for emulator or cassette interface |
| **`term-boot.dsk`** | Bootable Sedoric floppy (Microdisc): boot then `LOAD"TERM":CALL#1000` |

Or rebuild from sources:

```bash
make client              # -> client/term.tap   (requires xa65)
client/build-disk.sh     # -> client/term-boot.dsk  (requires the emulator + Microdisc ROM + Sedoric master)
```

Run in the emulator (ACIA `$031C`, or LOCI `$0380` — see
[`docs/hardware-connection.md`](docs/hardware-connection.md)):

```bash
# standard ACIA $031C (modem menu: choice 1)
oric1-emu -t client/term.tap -f -r basic11b.rom \
  --serial modem:pavi.3617.fr:6502 --serial-buffer 512

# LOCI / Pico WiFi $0380 (modem menu: choice 2) — faithful model, WITHOUT --acia-addr
oric1-emu -t client/term.tap -f -r basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

## Deployment

```bash
cp deploy/deploy.conf.example deploy/deploy.conf   # then fill in (gitignored)
make deploy                                         # builds (linux/amd64) + bbsoric systemd service (6502)
```
Details in [`deploy/`](deploy/). `deploy.conf` (real infrastructure) is not versioned.

## Why it differs from a C64/Atari BBS

The Oric does **not** use PETSCII. Its TEXT mode is close to **Teletext/Minitel**: colours and
attributes (ink, paper, blink, double height) are set by **control codes that occupy a screen
cell** (serial attributes). The technical heart of the project is this "OASCII" rendering layer —
see [`docs/architecture.md`](docs/architecture.md).

## Documentation

- [`ROADMAP.md`](ROADMAP.md) — plan by sprints
- [`CHANGELOG.md`](CHANGELOG.md) — change history
- [`docs/architecture.md`](docs/architecture.md) — technical architecture & Oric specifics
- [`docs/oascii.md`](docs/oascii.md) — OASCII display layer (Teletext attributes, palette, API)
- [`docs/state-of-the-art.md`](docs/state-of-the-art.md) — analysis of existing retro BBS servers
- [`docs/agile/backlog.md`](docs/agile/backlog.md) — product backlog (user stories)
