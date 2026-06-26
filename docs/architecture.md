# Technical architecture — BBS Oric

## 0. Repository organization (3 sub-projects)

```
bbsoric/
  internal/            SHARED packages (importable by server/ and studio/)
    content/  oascii/  render/
  server/              the Go BBS server
    cmd/bbsd/          daemon binary
    internal/          server-specific: bbs/ server/ user/
  client/              the Oric terminal (term.s, .tap build)
  studio/              the "forge" studio (content editing + deployment)
  content/  deploy/  docs/  scripts/
```

`content` and `oascii` stay in the root `internal/` (Go visibility) so that the studio
reuses **the same** validation and the same palette as the server, without duplication.

## 1. Overview

```
┌─────────────┐   telnet/TCP   ┌──────────────────────┐
│  Oric-1 /   │  (serial ACIA) │   BBS Oric server     │
│  Atmos      │◀──────────────▶│  (PC / Raspberry Pi)  │
│             │                │                       │
│ LOCI +      │                │  ┌─────────────────┐  │
│ WiFiModem   │   AT / Hayes   │  │ Network layer   │  │  TCP, telnet (IAC),
│ (serial ACIA)│               │  │ (1 task/conn.)  │  │  timeout
└─────────────┘                │  ├─────────────────┤  │
       ▲                       │  │ BBS engine      │  │  menus, sessions, login
       │ test                  │  ├─────────────────┤  │
┌─────────────┐                │  │ OASCII layer    │  │  Oric Teletext rendering
│  oric1-emu  │ --serial tcp:  │  │ (screen render) │  │  (serial attributes)
│  (Oric1)    │◀──────────────▶│  └─────────────────┘  │
└─────────────┘                └──────────────────────┘
```

## 2. Layers

### 2.1 Network layer
- TCP server in Go, **1 connection = 1 goroutine**.
- Minimal telnet negotiation (IAC) or "fake telnet" depending on decision (see ROADMAP §Decisions).
- Inactivity timeout, clean shutdown, session logging.

### 2.2 BBS engine
- Session loop (like petscii-bbs's `doLoop()`): display screen → read input → route.
- Menu stack / navigation, welcome screen, optional login.
- Persistence (users, messages) — format to be defined in Sprint 2.

### 2.3 OASCII layer (technical core — Sprint 1)
Encapsulates the Oric display specifics so that the BBS engine stays agnostic.

**Oric TEXT mode:** 40 columns × 28 lines, **Teletext** type. Attributes are **serial**:
a control code (value < 32) placed in a screen cell changes the rendering **from that cell
to the end of the line** (or until the next code). Consequences:

- Placing a color **consumes a column** → plan for the room in the layout.
- Attributes do not "cross" line ends (reset on each line).

**TEXT attribute codes (to confirm/complete in implementation):**

| Range | Effect |
|-------|-------|
| `0`–`7`   | Ink color (ink 0..7) |
| `8`–`15`  | Text attributes (flashing, double height, standard/alternate character sets) |
| `16`–`23` | Background color (paper 0..7) |
| `24`–`31` | Attributes (mode, etc.) |

> ⚠️ These ranges must be **verified on hardware/emulator** in Sprint 1 (exact Oric attribute
> table) before being frozen.

**Target API (language-agnostic):**
```
cls()                  clears the screen
at(x, y)               positions the cursor
ink(c)                 ink color (0..7)  → emits the attribute code
paper(c)               background color (0..7)
print(text)            writes text
println(text)          writes + newline
flush()                sends the buffer
read_key() / read_line()  keyboard read
```

## 3. Test pipeline (without hardware)

**Single** reference emulator: `/home/bmarty/Oric1/oric1-emu` (Phosphoric). Details and
commands in [`emulator-testing.md`](emulator-testing.md).

1. Start the BBS server on `127.0.0.1:6502`.
2. Connect the emulator: `./oric1-emu --serial tcp:127.0.0.1:6502 --acia-addr 031C`.
3. Variant: `--serial loopback` to test the ACIA alone; `nc 127.0.0.1 6502` for the server alone.

## 4. Real hardware (Sprint 4)

- Oric-1/Atmos + **LOCI** + USB WiFiModem. Addressing: LOCI MIA at **`$03A0-$03BF`** (see oric1-emu
  `--loci`); "standard" ACIA at **`$031C`** (Telestrat / oric1-emu default).
- The Oric client will have to target the right ACIA base depending on the setup.
- Full local test pipeline via the emulators: see [`emulator-testing.md`](emulator-testing.md).
- Hayes AT commands to establish the telnet connection to the server.

## 5. Internet exposure (first-order constraint)

The BBS is a **public Internet server**: it listens on `0.0.0.0:<port>` and is reachable from
any Oric connected via its WiFiModem. Consequences to integrate from the start:

- **Public port**: **`6502`** chosen (a nod to the Oric's CPU; avoids port 23, which is heavily scanned and
  often blocked outbound by ISPs). To be configured on the client side in `ATD <host>:6502`.
- **Hosting**: **cloud VPS with fixed IP** (24/7 public service, direct exposure without dynamic DNS).
- **No encryption**: Oric clients do not do TLS → the telnet stream is **in clear**.
  Therefore: never any sensitive secret on the user side, BBS passwords treated as non-confidential,
  server-side hashing nonetheless.
- **Attack surface**: a port open on the Internet is scanned constantly.
  - Controlled binding, **rate limiting** per IP, **simultaneous connection limit**.
  - Defensive reading of inputs (never any `eval`, bounded sizes, aggressive timeouts).
  - Connection logging (IP, timestamp) + log rotation.
  - Process isolation (dedicated unprivileged user / container).
- **Availability**: `systemd` service or container with auto-restart on the VPS.

> These points raise security and deployment as **cross-cutting** concerns, not as a final
> sprint. See the updated ROADMAP.

## 6. Architecture decisions (ADR)
ADRs are versioned in `docs/adr/`.
- **ADR-0001** — Login: gate at CONNECT triggered by the JSON start page (special
  target → component), PBKDF2 stdlib hashed persistence. (`docs/adr/0001-login-component-page.md`)
- **ADR-0002** — Input model: Oric terminal in character mode, `ReadKey` (menus) +
  `ReadLine` (text fields). (`docs/adr/0002-input-model.md`)
- **ADR-0003** — "Forge" studio: Go web app, shared `internal/`, deployment by profiles
  (dev/int/prod), studio = source of truth. (`docs/adr/0003-studio-forge.md`)

Decisions still open: see `ROADMAP.md` §"Open decisions" (ACIA addressing,
telnet negotiation, OASCII table).
