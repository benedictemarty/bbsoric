# Client review (Oric terminal) — 26/06/2026

Engineer-level review of the 6502 terminal (`client/term.s`, `xmodem.s`,
`sedoric.s`, `altcharset.s`) and the given test suites. Severity: 🔴 high, 🟠 medium,
🟡 low.

## Resolved

| # | Sev | Issue | Fix |
|---|-----|-------|-----|
| LOCI | 🔴 | Option "2 = LOCI" targeted `$03A0` (the **MIA** area), not the modem → MIA/ACIA collision, PSG disrupted, **keyboard frozen on the directory** | `mm_loci` now points to **`$0380`** (the ACIA of the LOCI WiFi modem, cf. `PicoWiFiModemUSB` firmware). Validated with `--loci --serial picowifi`: `2`→`1`→`CONNECT`. Cf. `phosphoric-findings.md` F1. |
| 2 | 🔴 | **Out-of-bounds plot**: `set_cursor_xy` (`1F col row`) without bounds → write outside VRAM from untrusted network input (third-party BBS) | Clamp `row<28`, `col<40` before address computation (`term.s set_cursor_xy`). |
| 3 | 🔴 | **Unbounded XMODEM reception**: write starting at `$4000` with no ceiling → overflow (screen, ROM) from the network | Refused if buffer ≥ `$B800`: `CAN` + "FICHIER TROP GROS" message (`xmodem.s xr_block`). |
| 5a | 🔴 | **No uppercase**: `asciitab` lowercase only, no SHIFT → mixed-case passwords cannot be entered | `scan_shift` (reads LSHIFT col4/row4, RSHIFT col7/row4); `key_scan` maps `a-z`→`A-Z` when SHIFT is held. Validated in emulator (`\L` → TX `$59 'Y'`, `$5A 'Z'`). |
| 5b | 🔴 | **No backspace**: `input_line` ignores `<$20`; the server `ReadLine` does not erase → impossible to correct an entry | **DEL** key (col5/row5) → `$08`; `putbyte` handles `$08` (destructive erase); `input_line` and the server `ReadLine` remove the last character (`$08`/`$7F`). Server test `TestReadLineBackspace`. |
| 11 | 🟡 | Permanent `sei` not explained | Comment added (bare-metal terminal, keyboard+serial handled in-house; Sedoric re-SEIs). |
| 10 | 🟡 | Zero-page allocation documented in prose, not centralized | ZP map added at the top of `term.s` (+ `SHIFTF=$F3`). |

## Review correction

- **#4 (chat invisible while typing)**: **disproved**. The `main` loop already
  interleaves RX (rendering) and keyboard scan (1 key/iteration); messages
  pushed by the server **are displayed** while the user types during a
  session. The `get_key` blocking only concerns the **pre-connection menus**
  (modem, directory) where no serial data arrives — acceptable.

## Deferred (structural / to validate on hardware) — with justification

| # | Sev | Issue | Why deferred |
|---|-----|-------|--------------|
| 1 | 🔴 | **RX byte loss during a `scroll_up`** (memmove ~1 KB ≈ several bytes lost at 9600 baud); no flow control | The real fix = **RTS/CTS** or XON/XOFF + server pacing, to validate on **real hardware** (the emulated 6551 + `--serial-buffer` mask the defect). To address before iron use. Risk too high without HW. |
| 6 | 🟠 | No reading of **modem result codes** (`CONNECT`/`NO CARRIER`) nor **DCD** → "looks frozen" if the connection fails, no hang-up detection | Requires a mini AT-response parser + DCD monitoring; a feature in its own right, to be designed (not a simple fix). |
| 7 | 🟠 | No **telnet IAC filtering** on the client side → third-party BBS that negotiate display control characters | The Oric BBS emits **no** IAC. A **partial** telnet parser (without SB sub-negotiation) would be **worse** than the documented limitation. Full telnet is a **feature**, not a fix. |
| 8 | 🟠 | ACIA error bits (overrun/framing) never read → silent loss | Linked to #1; without flow control, reading overrun brings no recovery. To address with #1. |
| 9 | 🟠 | Single-file Sedoric backup (`BBSFILE.BIN`, overwrites) | Acceptable in alpha; a name derived from the transfer requires a protocol (the server does not send a name). |
| 12 | 🟡 | Weak client test coverage (`test-emulateur.sh` smoke test fragile, based on cycle counter) | Keyboard scan + SHIFT was **validated end-to-end** in the emulator (serial trace). An automated multi-step test (manual entry) remains fragile via `--type-keys`; backlog. |

## Validation (this iteration)

- `make client`: assembled (3876 bytes). `.dsk` rebuilt.
- Emulator: LOCI `$0380` `2`→`1`→`CONNECT` (banner rendered); SHIFT `\L` → TX
  uppercase; normal rendering not regressed (plot clamp OK on valid coords).
- Server: `go test -race ./...` green; `TestReadLineBackspace` (4 cases) green.
