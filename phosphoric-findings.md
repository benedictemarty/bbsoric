# Phosphoric findings (Oric emulator) encountered from bbsoric

Log of defects / pitfalls of the **Phosphoric** emulator (`~/Oric1`) spotted while
testing the BBS terminal. To report to the Phosphoric team (and some already fixed).

---

## F1 — `--loci` + `--acia-addr 03A0` freezes the keyboard (directory "frozen")

**Status: diagnosed + reproduced. Guard added on the Phosphoric side (v1.27.2).**

| Command | Result |
|----------|----------|
| `--serial picowifi --acia-addr 03A0` (without `--loci`) | ✅ connects (CONNECT 9600 + banner) |
| `--loci --serial picowifi --acia-addr 03A0` | ❌ **frozen on the directory** |

**Cause.** `--loci` places the LOCI MIA on `$03A0–$03BF` and `--acia-addr 03A0`
also forces the ACIA there → **double mapping** at the same place. In Phosphoric's I/O
callbacks, the MIA is routed **before** the ACIA → it **masks** the ACIA. But the MIA
drives the **PSG (AY)**, and the terminal's `key_scan` scans the keyboard via that same
PSG. Result: the scan reads nothing → `get_key` loops indefinitely → the directory
appears frozen (it waits for a key it will never see). Confirmed by the log:
`LOCI: pre-seeded PSG R7=$7F … + LOCI MIA enabled at $03A0-$03BF`.

**⚠️ Correction (the picowifi IS the LOCI's modem).** Recommending to "remove
`--loci`" was a mistake: the picowifi only makes sense **with** the LOCI emulation.
The real LOCI exposes its modem as an **ACIA at `$0380`** (confirmed by the reference
firmware `~/picowifi/PicoWiFiModemUSB`: "reference Oric program (via LOCI,
ACIA `$0380`)"). The `$03A0` is the **MIA space** (not the modem); it only "worked"
**without** `--loci`, by placing a bare ACIA at `$03A0` — an unfaithful workaround.

**Correct command (faithful to the hardware).** Keep `--loci`, **without** `--acia-addr`
(the ACIA defaults to `$0380`):

```bash
~/Oric1/oric1-emu \
  -t client/term.tap -f -r ~/Oric1/roms/basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

The terminal must then **address the modem at `$0380`**, not `$03A0`.

**Validated on the official LOCI firmware** (`github.com/sodiumlb/loci-firmware`):

| Fact | Source |
|------|--------|
| Modem/serial = **ACIA `$0380-$0383`** | `src/mia/oric/acia.h`: `ACIA_IO_DATA 0x0380` / `STAT 0x0381` / `CMD 0x0382` / `CTRL 0x0383` |
| `$03A0-$03BF` = **MIA registers** | `src/mia/sys/mem.h`: `.equ regs, 0x200400A0  // Oric address 0x03A0-0x03BF` |
| `$03A0`/`$03A2` = **MIA UART console** (≠ modem) | `src/mia/sys/mia.c` (0x03A0 "UART Tx/Rx flow control", 0x03A2 "UART Rx") |
| `$03A3` = **"ULA pattern match"** | `src/mia/sys/mia.c` `CASE_WRITE(0x03A3)` → explains why writing the "ACIA control" at `$03A3` breaks the ULA/PSG/keyboard |

**To fix on the terminal side (bbsoric).** The modem menu option "`2` = LOCI / `$03A0`"
targets the wrong base: for the real LOCI you need **`$0380`** (registers
`$0380-$0383`). → adjust `client/term.s` (cf. `MESSAGE-terminal-team-LOCI.md`).

**On the Phosphoric side (v1.27.2 → 1.27.3).** `oric1-emu` warns when `--loci` is active
and `--acia-addr` falls within `$03A0–$03BF`, and **now points to `$0380`**:
`WARNING: … Le modem LOCI (picowifi) est exposé à $0380 … laissez --loci SANS --acia-addr et adressez $0380.`

**Remaining to do (Phosphoric).** Make the MIA and a serial channel **coexist** in the
MIA space like the real LOCI (the USB-CDC modem is exposed there via the MIA protocol),
for software that would go through the MIA rather than through the ACIA `$0380`.
