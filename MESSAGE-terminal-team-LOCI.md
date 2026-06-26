# Terminal team message — the "LOCI" modem option must target `$0380`, not `$03A0`

*(Validated on the official LOCI firmware `github.com/sodiumlb/loci-firmware`.)*

Hi,

Confirmed bug in the modem menu of `term.tap` (`client/term.s`).

## The problem

The "LOCI" option initialises an ACIA at **`$03A0`**. But `$03A0-$03BF` is the
LOCI's **MIA register file** (`src/mia/sys/mem.h`:
*"Oric address 0x03A0-0x03BF"*), not the modem. Worse: `$03A2` is the **MIA UART
console** and `$03A3` is the **"ULA pattern match"** register
(`src/mia/sys/mia.c`). Writing the "ACIA control" at `$03A3` disturbs the ULA/PSG →
the keyboard scan no longer reads anything → **the terminal freezes on the directory**.

## The correct base

The LOCI's WiFi modem is exposed as an **ACIA 6551 at `$0380-$0383`**
(`src/mia/oric/acia.h`):

```c
#define ACIA_IO_DATA 0x0380
#define ACIA_IO_STAT 0x0381
#define ACIA_IO_CMD  0x0382
#define ACIA_IO_CTRL 0x0383
```

## Requested fix (`client/term.s`)

Make the "LOCI" option point to the base **`$0380`** (registers `$0380-$0383`)
instead of `$03A0`.

## Test (emulator, faithful model)

```bash
oric1-emu -t client/term.tap -f -r roms/basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

(without `--acia-addr`: the ACIA defaults to `$0380`). Do **not** launch with
`--acia-addr 03A0`.

## Note

`$03A0` only "worked" **without** `--loci` (a bare ACIA placed there — unfaithful to the
hardware; the picowifi only makes sense with the LOCI emulation). Phosphoric ≥ 1.27.3
warns and points to `$0380`. Technical details: `phosphoric-findings.md` (F1).
