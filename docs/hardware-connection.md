# Real hardware connection — connecting to the BBS Oric from a real Oric

> **Sprint 4 — Real hardware connection.**
> This document describes how to reach the BBS Oric (`pavi.3617.fr:6502`) from a
> **physical Oric-1 or Atmos** equipped with a serial interface and a WiFi modem.
> The client software (`client/term.s`) is validated **end-to-end in the
> emulator**; the test on **real hardware** is still to be done (see §7,
> for lack of hardware available at the time of writing).

---

## 1. Overview of the chain

```
┌──────────────┐   bus     ┌───────────────┐   serial   ┌────────────┐   WiFi/TCP   ┌──────────────┐
│   Oric-1 /   │  6502 +   │  Serial       │  TTL/RS232 │  WiFi      │ ───────────► │  BBS Oric    │
│   Atmos      │  ACIA ────│  interface    │ ──────────►│  modem     │   telnet     │  pavi.3617   │
│  (term.tap)  │  $031C or │  (ACIA 6551)  │  9600 8N1  │ (Hayes AT) │   :6502      │  .fr:6502    │
└──────────────┘  $0380    └───────────────┘            └────────────┘              └──────────────┘
```

The Oric does **no TCP, no WiFi, no TLS**. It drives an **ACIA 6551** (UART)
and sends **Hayes AT commands** over the serial port. It is the **WiFi modem**
that opens the TCP connection (and terminates the TLS where applicable). On the
Oric side, everything boils down to: *writing bytes to the ACIA, reading bytes back*.

---

## 2. Supported serial interfaces (ACIA addressing)

The `term.s` client targets an **ACIA 6551** via a runtime pointer `ACIAPTR`. The
startup menu chooses the base:

| Menu choice | ACIA base | Typical setup |
|-----------|-----------|-----------------|
| `1` | **`$031C`** | "Standard" Telestrat ACIA / `oric1-emu` default. Serial cards plugged into the expansion bus at this base. |
| `2` | **`$0380`** | **LOCI** (modern Oric expansion card) — its **WiFi modem** (PicoWiFiModemUSB) is exposed as an ACIA at `$0380`. ⚠️ Do not confuse with the LOCI **MIA space** `$03A0–$03BF` (card registers, not the modem). |

Both setups expose the **same 6551 register** (offsets from the base):

| Offset | Register | Use |
|-------:|----------|-------|
| `+0` | Data | read = received byte (RX), write = byte to transmit (TX) |
| `+1` | Status | bit `RDRF`=`$08` (received data available), bit `TDRE`=`$10` (transmit ready) |
| `+2` | Command | `$0B` = DTR on, IRQ off, no echo |
| `+3` | Control | `$1E` = **9600 baud, 8N1** |

> ✅ The LOCI modem base **`$0380`** is confirmed by the reference firmware
> `PicoWiFiModemUSB` (Oric program: ACIA `$0380`) and validated in the emulator
> (`--loci --serial picowifi`, cf. `phosphoric-findings.md` F1). It remains to be confirmed
> on **real hardware** (cf. `docs/architecture.md` §4). The menu
> allows switching without recompiling; if neither one responds, check the
> pinout and the base of your card (see §6 troubleshooting).

### Unhandled cases

- **DTL 2000**: a **V23/Minitel** modem (6850 + PIA, no Hayes AT or TCP). It does
  not allow reaching an Internet telnet BBS → out of scope.

---

## 3. The WiFi modem

The BBS was developed and validated against a **WiFi modem with Hayes firmware** of type
**Pico W / `picowifi` (firmware v0.2.0)**, the functional equivalent of the common retro
WiFi modems ("WiFiModem232", "Tirreno", "RetroWiFiModem" families…). Any
modem exposing a compatible Hayes AT command set and a serial rate fixed at
**9600 8N1** is suitable.

### First configuration (once, from an AT terminal)

```
AT                      ; must respond OK
AT+CWJAP="SSID","pwd"   ; join the WiFi network (depending on firmware)
AT&W                    ; save the config
```

> The exact WiFi association syntax **depends on the firmware** of your modem
> (`AT+CWJAP`, `ATWIFI`, interactive menu…). Refer to its manual. Once
> the WiFi is memorized, the modem reconnects on its own at power-up.

### Serial settings on the modem side

- **Rate: 9600 baud, 8 bits, no parity, 1 stop (9600 8N1)** — must
  match exactly the `$1E` programmed in the ACIA (§2).
- Flow control **disabled** (the Oric does simple polling, no RTS/CTS).

---

## 4. Placing a call (AT commands issued by the Oric)

`term.s` **composes the Hayes dial command itself** — the user
just chooses a directory entry or enters host/port. The commands
actually issued on the ACIA:

| Protocol | Command issued | Effect |
|-----------|----------------|-------|
| telnet / raw | `ATD<host>:<port>` + CR | opens a cleartext TCP connection |
| **TLS** | `ATDT#<host>:<port>` + CR | the `#` opens a **TLS call terminated by the modem**; the Oric receives cleartext |

Concrete examples (what the modem receives):

```
ATD pavi.3617.fr:6502         ; BBS Oric in clear (telnet)
ATDT# pavi.3617.fr:6992       ; BBS Oric via TLS (the modem decrypts)
```

The modem responds `CONNECT` when the link is established, then the BBS stream
(OASCII bytes) flows transparently. On the Oric screen, control bytes
0–31 become **serial Teletext attributes** (colors).

### TLS — reminder

The 8-bit Oric **does no crypto**. TLS is entirely handled by the modem:

- `AT$CA`: loads **one** root certificate (CA) — buffer ~8 KB (one CA, not a
  whole system bundle).
- `AT$CV1`: enforces **verification** of the server certificate (otherwise `VERIFY_NONE`).
- `ATGET https://…`: direct HTTPS GET (port 443) — outside the BBS stream.

Validated in the emulator (backend `--serial picowifi`, OpenSSL build): TLSv1.3,
BBS banner rendered through the tunnel (`docs/img/tls-dial.png`,
`docs/img/tls-verified-atcv1.png`).

---

## 5. Step-by-step procedure (from a real Oric)

1. **Plug** the serial interface (ACIA card `$031C` or LOCI `$0380`) into the
   Oric's expansion bus, the WiFi modem connected to the serial port, modem powered on and
   associated with the WiFi (§3).
2. **Load the terminal** `term.tap`:
   - Cassette / `.tap` reader: `CLOAD"TERM"` (autorun, the program starts on its own).
   - The `.tap` is produced by `client/build.sh` (autorun, loading at `$1000`).
3. **Modem menu**: type `1` (ACIA `$031C`) or `2` (LOCI `$0380`) depending on the card.
4. **Directory**: type the number of the desired entry, e.g. `1` =
   `BBS Oric (prod) pavi.3617.fr`, or `M` for **manual entry**
   (host, port, telnet/TLS protocol).
5. The terminal **composes `ATD…`** and displays "Dialing in progress…". On
   `CONNECT`, the **BBS banner** is displayed in color.
6. **Navigate**: menus are driven by keyboard (single key for menus,
   line + `RETURN` for text fields).

"By hand" equivalent (without `term.s`, for diagnostics) from any
AT terminal: `ATD pavi.3617.fr:6502` then `Enter`.

---

## 6. Troubleshooting

| Symptom | Leads |
|----------|-------|
| The modem menu does not respond / frozen screen | wrong ACIA base → try the other choice (`1`/`2`); check the actual base of the card. **LOCI emulator:** the (picowifi) modem of the real LOCI is an ACIA at **`$0380`**, not `$03A0`. Run `--loci --serial picowifi` (without `--acia-addr`) and address **`$0380`**; do NOT force `--acia-addr 03A0` (= MIA space → masks the ACIA + freezes the keyboard). Cf. `phosphoric-findings.md` (F1). |
| `ATD` with no effect, no `CONNECT` | serial rate ≠ 9600 8N1; modem not associated with WiFi; wrong host/port; flow control active on the modem side. |
| Garbage characters / unreadable text | rate mismatch (check `$1E` ACIA ↔ 9600 of the modem); TX/RX wiring swapped. |
| Missing colors (white text only) | normal on a generic terminal; the Oric renders serial attributes by writing directly to VRAM (`term.s`). |
| TLS fails | `AT$CV1` active without a loaded CA (`AT$CA`) → switch back to `VERIFY_NONE`, or load the right CA; TLS port = `6992`. |
| The TLS `#` is not accepted | modem firmware too old (terminated TLS requires picowifi v0.2.0+). |

---

## 7. Test on a real Oric — checklist (to run on hardware)

> **Status: awaiting hardware.** The pipeline is validated in the emulator
> (`scripts/test-emulateur.sh`); the checklist below is the **hardware acceptance
> protocol** to run as soon as a physical Oric + serial interface +
> WiFi modem are available. Report the results (OK/KO + photo) in
> `docs/img/` and check off in `ROADMAP.md`.

- [ ] **T1 — Loading**: `term.tap` loads and starts (modem menu displayed).
- [ ] **T2 — ACIA backend**: the right choice (`1`=`$031C` or `2`=`$0380`) initializes
      the ACIA without hanging.
- [ ] **T3 — Directory**: entry `1` composes `ATD pavi.3617.fr:6502`, modem
      responds `CONNECT`.
- [ ] **T4 — Color banner**: the OASCII welcome screen is displayed with the
      correct colors (yellow/cyan/green), 40 columns respected (photo).
- [ ] **T5 — Keyboard navigation**: menus drivable (single key), text
      fields (line + RETURN), return to menu.
- [ ] **T6 — Manual entry**: `M` → host/port/protocol → connection OK.
- [ ] **T7 — TLS**: entry `5` (`pavi.3617.fr:6992`) composes `ATDT#`, TLS tunnel
      established, banner rendered (photo).
- [ ] **T8 — Disconnection**: `Q` quits cleanly ("See you soon"), the modem
      hangs up.
- [ ] **T9 — Stability**: session of several minutes without screen corruption
      or character loss.

---

## References

- `client/term.s` — 6502 terminal (serial I/O, menu, dialing, terminal mode).
- `client/README.md` — build / emulator details.
- `docs/architecture.md` §4 (hardware targets), §5 (Internet exposure).
- `docs/oascii.md` — encoding of serial Teletext attributes.
- `docs/emulator-testing.md` — `oric1-emu` test pipeline.
</content>
</invoke>
