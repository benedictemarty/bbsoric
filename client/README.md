# Standalone BBS terminal for Oric (`term.s`)

6502 program on the Oric side that turns the machine into a **standalone BBS terminal**:
at startup it chooses the modem, displays a **directory**, **composes the Hayes dialling
command itself** (`ATD`), then switches to terminal mode. Reception is written
**directly to screen memory** (`$BB80`) so that control bytes 0–31 become real
**serial Teletext attributes** (OASCII colours).

## Flow

1. **Modem menu**: `1` = direct ACIA 6551 (`$031C`), `2` = LOCI / Pico W (`$03A0`).
2. **Directory**: BBS Oric (prod), ParticlesBBS, Altair, Heatwave, or `M` = manual entry.
3. **Manual entry**: host, port, protocol (`1`=telnet/raw, `2`=TLS).
4. Standalone `ATD<host:port>` dialling → **terminal mode**.

## Technical characteristics

- **Abstract serial I/O** via `ACIAPTR` (ZP pointer to the ACIA base) + primitives
  `ser_tx`/`ser_rx_ready`/`ser_rx` → a single binary for both 6551 backends (addresses
  `$031C` and `$03A0`, validated end-to-end: `CONNECT to pavi.3617.fr:6502`).
- ACIA 9600 8N1, DTR on, polling (no IRQ).
- Reception → screen: `CR`/`LF`/scroll, 40-column clamp.
- **Keyboard TX**: 8×8 matrix scan (PSG-via-VIA), debounce, local echo. `input_line`
  for manual entry.
- Loaded/executed at `$1000`. ~1.5 KB.

## Modems and protocols

- **DTL 2000** not supported: it is a **V23/Minitel** modem (6850 + PIA, no Hayes AT nor
  modern TCP) — it cannot reach an Internet telnet BBS.
- **TLS/SSL**: the 8-bit Oric does **no crypto**. TLS is terminated by the **WiFi
  modem** (Pico W, firmware v0.2.0) which presents cleartext to the Oric. On the Oric side,
  "protocol" selects the dialling command:
  - telnet/raw → `ATD host:port`
  - **TLS → `ATDT#host:port`** (the `#` opens a TLS-terminated call)
  Validated **end-to-end in the emulator** (OpenSSL build, `--serial picowifi` backend):
  TLSv1.3, BBS banner rendered through the tunnel (`../docs/img/tls-dial.png`).
  Certificate verification: `VERIFY_NONE` by default; `AT$CA` loads **one** root CA
  (8 KB buffer, matching the firmware — not the whole system bundle) and `AT$CV1` enforces
  verification. `ATGET https://...` also allows an HTTPS GET (port 443).

## Build

```bash
./build.sh        # xa term.s -> term.bin -> bin2tap -> term.tap (autorun)
```
Override: `BIN2TAP=/path/bin2tap ./build.sh`.

## Test in the emulator

From the repository root:
```bash
oric-client/build.sh
scripts/test-emulateur.sh /tmp/oric.ppm
```
The script launches the server, starts `oric1-emu` connected over serial TCP, and captures
the rendering. Reference result: [`../docs/img/sprint1-banner.png`](../docs/img/sprint1-banner.png).

## Technical notes

- `xa` does not support UTF-8 nor `:` in comments → ASCII comments.
- ACIA address chosen at runtime via the modem menu (`ACIAPTR` = `$031C` or `$03A0`).
- Bytes `$0A`/`$0D` reserved for line control: the OASCII layer avoids emitting them
  as attributes (the attributes 0x0A/0x0D — double height alone / blink+alt —
  are to be avoided in the current protocol).
- **`--type-keys` test**: the tool holds a key down until an identical key or the end of
  the string → multi-screen navigation automates poorly (doubling the key forces a
  release). Each step was validated separately; with human typing (press/release)
  everything chains normally.
