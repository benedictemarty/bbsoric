# ADR-0002 — Input model: character-mode terminal, ReadKey + ReadLine

- **Status**: Accepted
- **Date**: 2026-06-22
- **Sprint / Backlog**: Sprint 2 — item **C4** (login) and navigation comfort
- **Deciders**: bmarty
- **Related to**: ADR-0001 (login)

## Context

"Snappy" historical BBSes react to a **single keystroke without RETURN** for menu
navigation (cf. `readKey` of `petscii-bbs`), while requiring a **line terminated by
RETURN** for input fields (handle, password, future messages).

Today the Oric terminal (`oric-client/term.s`) **buffers a line and only emits it
at RETURN** (CR), and the server only offers `Session.ReadLine`. Consequence: pressing
`1` in a menu does nothing until RETURN is typed — not the BBS *feel*.

We want **both behaviours**, chosen according to context:
- **menus / "press a key"** → immediate reaction to a keystroke;
- **text fields** (handle, password) → multi-character input validated by RETURN.

## Decision

### 1. The Oric terminal is already in *character mode* (verified — no change required)

**Observation (2026-06-22)**: on re-reading, `oric-client/term.s` **already emits each
keystroke immediately**. The `main` terminal loop does `key_scan` → (if new key) `ser_tx`
of the byte → local echo `putbyte`, **without a line buffer**; the `input_line` buffer only
serves the manual host/port input (before connection), not the BBS session. The terminal
therefore behaves like a classic serial terminal in character mode, and keeps the local
echo. **No modification of `term.s` is necessary** for `ReadKey`/`ReadLine` on the
server side (the initial assumption of a buffered terminal was wrong).

### 2. The server exposes two primitives

- **`ReadKey() (byte, error)`** *(new)* — reads **one** significant byte: filters the
  IAC telnet sequences, ignores residual `CR`/`LF`/`NUL`, returns the first real
  key. Used for **menu choices** and "press a key" screens.
- **`ReadLine() (string, error)`** *(existing)* — **accumulates** bytes until CR.
  Used for **text fields** (handle, password). Already reading byte by byte, it
  works without modification whether the client emits in a burst or character by character.

### 3. Who uses what

| Screen | Primitive |
|-------|-----------|
| Menu (choosing an entry) | `ReadKey` |
| Content page ("a key to go back") | `ReadKey` |
| Login/registration component (handle, password) | `ReadLine` |

## Consequences

**Positive**
- BBS-style responsive navigation (one keystroke = one action), robust line-based text input.
- `ReadLine` unchanged works with the character-mode terminal (byte/byte reading).
- Clean separation: the `server` layer provides the primitives, the `engine`/components
  choose according to context.

**Negative / to watch**
- ~~`term.s` must be modified~~ → **no**: the terminal is already in character mode (cf.
  Decision 1). The **end-to-end validation in the emulator** of the new login screen
  remains to be done: the emulated modem backend dials the real host names of the
  directory and the `--type-keys` sync is fragile → plan a local entry in the picowifi config
  or a test on real hardware. The server is validated via `nc` + integration tests.
- With a "dumb" client (`nc`) that sends `1\r\n`, `ReadKey` consumes `1` and leaves
  `\r\n`; the next `ReadKey` ignores these residual `CR`/`LF` (hence the explicit skip).
  The real terminal in character mode does not emit a CR after a menu key.
- Local echo displays the password (already accepted in ADR-0001, TLS covers the transport).

## Rejected alternatives

1. **Everything line + RETURN** (status quo): simple but heavy navigation, not matching the
   requested BBS *feel*.
2. **Everything single-key**: impossible for multi-character text fields (handle,
   password).
3. **Telnet negotiation (character mode via IAC)** to drive the mode remotely: the
   home-made Oric terminal does not implement negotiation; we choose a terminal that emits
   in character mode by construction, simpler.
