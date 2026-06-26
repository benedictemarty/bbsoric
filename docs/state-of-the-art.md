# State of the art — Retro BBS servers

> Synthesis done at Sprint 0 to frame the BBS Oric project.

## 1. The "modern server + 8-bit client" model

Current retro BBSes (PETSCII BBS / ATASCII style) do **not** run on the period machine:
a **modern server** (PC / Raspberry) listens on **TCP/telnet**, and the 8-bit machine connects to it
via a **WiFi modem** (ESP8266 or Pico W, Hayes AT commands) plugged into a serial interface.
The server produces screens in the machine's native character set.

## 2. Reference projects

| Project | Stack | Target | Contribution |
|--------|-------|-------|--------|
| [sblendorio/petscii-bbs](https://github.com/sblendorio/petscii-bbs) | Java 21 | C64/128, ASCII, **Minitel/Videotex**, Prestel | **Architecture reference.** 1 thread/connection, classes `PetsciiThread`/`AsciiThread` + `doLoop()`, API `print/readKey/cls`. The Videotex support is very close to the Oric need (Teletext). |
| [retrocomputacion/retrobbs](https://github.com/retrocomputacion/retrobbs) | Python 3 | C64/Plus4/MSX (Turbo56K) | Multimedia, **neutral encoding** since v0.50 → good terminal abstraction model. |
| [Magnetar-BBS](https://github.com/Commodore64HomeBrew/Magnetar-BBS) | Native 6502 | C64 | BBS that runs *on* the machine (RR-NET/SD2IEC). Opposite philosophy to ours. |
| [TheOldNet BBS](https://github.com/TheOldNet/theoldnet-bbs) | Java (petscii-bbs fork) | C64 | Production example (web browsing in PETSCII). |

## 3. Connection ecosystem

- **WiFi modems**: [RetroWiFiModem](https://github.com/mecparts/RetroWiFiModem) (ESP8266),
  [PicoWiFiModem](https://github.com/mecparts/PicoWiFiModem),
  [PicoWiFiModemUSB](https://github.com/sodiumlb/PicoWiFiModemUSB) (explicitly mentions **LOCI / Oric**).
- **LOCI** ([Raxiss](https://www.raxiss.com/article/id/38-LOCI)): Oric-1/Atmos bus extension, ROM/floppy
  emulation + **USB CDC modem exposed as an ACIA at address `0x380`**. This is our serial entry point
  on Oric-1/Atmos (which have no native RS232, unlike the Telestrat / ACIA 6551).
- **Emulators**:
  - [Oricutron](https://github.com/pete-gordon/oricutron) — configurable ACIA backend:
    `none`, `loopback` (test), `com` (real/virtual serial port), `modem` (sockets + AT + telnet port 23).
    ACIA emulated at `#31C` (Telestrat address). Allows a **100% software test pipeline**.
  - Phosphoror — Oric emulator (cross-test).

## 4. The Oric case: what does not yet exist

No turnkey Oric BBS spotted. The key differences vs C64/Atari:

1. **No PETSCII.** Character set close to ASCII, but the **TEXT 40×28 display is Teletext-style**:
   colors/attributes set by **control codes occupying a screen cell** (serial attributes).
   → The rendering must be designed as **Teletext/Videotex**, not as a classic ANSI terminal.
2. **Non-native serial** on Oric-1/Atmos (extension required: LOCI, Microdisc+, etc.).
3. **Constrained screen model**: 40 columns, attributes that "consume" columns → the layout
   must anticipate the space taken by the color codes.

## 5. Implications for our architecture

- Draw inspiration from the **1 connection = 1 task** model + petscii-bbs screen API.
- Build an **"OASCII"** layer dedicated to Oric serial Teletext attributes (core of Sprint 1).
- Test first **in `Oric1/oric1-emu`** (Phosphoric, `--serial tcp:`) before any hardware
  — it is the project's reference emulator (cf. `emulator-testing.md`).

## 6. Functional parity — gaps and plan (update 25/06/2026)

The initial framing (sections 1–5) focused on the **architecture**. This section
compares the **features** of the BBS Oric server to the state of the art (petscii-bbs
reference) to guide the next increments.

### 6.1 What the server already offers
- Single-key menus, multiple sessions (1 goroutine/connection).
- **Guest** access + **user accounts** (PBKDF2, atomic hashed store).
- Declarative **forms** (login/registration + generic), retry + failure page.
- **File library** with **XMODEM download/upload**.
- Static pages, **cursor positioning** (plot X,Y), **differential screen buffer**.
- Operations: Prometheus metrics, `/healthz`, backup/restore.

### 6.2 Gaps vs state of the art

The historical core of a BBS is the **communication spaces between
callers** — this is what distinguishes a BBS from a simple menu system.
Today the "Guestbook" is a **static page**, not writable.

| # | Feature | Reference | Effort | Impact | Oric/OASCII note |
|---|----------|-----------|:------:|:------:|------------------|
| 1 | **Message base / forums** (read + post, threads, persisted) | petscii-bbs (core) | ●●● | ●●● | Paginated reading via differential buffer |
| 2 | **One-liner wall / writable guestbook** (persisted message) | universal | ● | ●● | Reuses `form` + atomic JSON store |
| 3 | **Who is online + chat / paging** inter-caller ✅ | BBS signature feature | ●● | ●● | **Done**: `server/internal/presence` + `who`/`chat` applets (Sprint 7) |
| 4 | **Private messaging** between accounts | petscii-bbs | ●● | ●● | Reuses `internal/user` |
| 5 | **News feed / RSS → OASCII** | petscii-bbs "internet services" | ●● | ●● | Showcase, network-bounded |
| 6 | **Door game** (online game) | petscii-bbs (many) | ●●● | ● | Leverages the differential buffer |

### 6.3 Recommended order
- ✅ **#3 who-is-online / chat** — **delivered in Sprint 7**: the cheapest
  differentiator (engine already multi-session), `presence` registry + applets
  `who`/`chat` (real-time room, non-blocking broadcast).
1. **#2 writable one-liner wall** — quick win, establishes the **"persisted write
   applet" pattern** (atomic JSON store modeled on `internal/user`).
2. **#1 message base** — *the* feature that moves from "menus" to "BBS" in the
   state-of-the-art sense; the #2 pattern generalizes to it (threads, paginated reading).
3. Then #4 private messaging, #5 news, #6 door game.

> These features fit into the **content-driven** architecture
> (`content/site.json` + applets registered via `bbs.Register`), without breaking:
> each feature = one applet + (if needed) a persisted store.

## Sources
- https://github.com/sblendorio/petscii-bbs
- https://github.com/retrocomputacion/retrobbs
- https://github.com/Commodore64HomeBrew/Magnetar-BBS
- https://github.com/TheOldNet/theoldnet-bbs
- https://www.raxiss.com/article/id/38-LOCI
- https://github.com/sodiumlb/PicoWiFiModemUSB
- https://github.com/pete-gordon/oricutron
- https://forum.defence-force.org/viewtopic.php?t=1138
- https://wiki.defence-force.org/doku.php?id=oric:hardware:serial
</content>
