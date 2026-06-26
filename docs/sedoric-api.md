# Sedoric API — file save/load (Microdisc path B)

> Extracted from the official disassembly **"Sedoric 3.0 à nu"**. Used to
> write/read a real **Sedoric file** (catalog, name) from the Oric terminal, for
> Microdisc storage of file transfer (backlog **G1**, path B).
>
> **Prerequisite: resident Sedoric** (Oric booted from a Sedoric disk; Microdisc
> ROM active as overlay at `$E000`, Sedoric RAM at `$C000+`, vector table in page
> `$FF`). The terminal must therefore run **under Sedoric**, not as a .tap cassette
> on a diskless machine.

## Vector table (public API, page `$FF`)

| Vector | JMP | Routine | Role |
|---------|-----|---------|------|
| `$FF73` | `4C E5 E0` → `$E0E5` | **XLOADA** | loads the file named `BUFNOM` according to `VSALO0/1`, `DESALO` |
| `$FF76` | `4C 28 DE` → `$DE28` | **XDEFSA** | sets the default values for `XSAVEB` |
| `$FF79` | `4C E6 DF` → `$DFE6` | **XDEFLO** | sets the default values for `XLOADA` |
| `$FF7C` | `4C 9C DE` → `$DE9C` | **XSAVEB** | saves the `BUFNOM` file according to `VSALO0/1`, `DESALO`, `FISALO`, `EXSALO` |

(Other useful vectors: `$FF70` loads according to POSNMX/POSNMP/POSNMS; `$FF8B`
XWDESC writes the descriptors; `$FF88` XLIBSE free sector; etc.)

## System variables (Sedoric RAM)

| Address | Name | Role |
|---------|-----|------|
| `$C029`–`$C034` | **BUFNOM** | file name: 9 bytes name + 3 bytes extension (padded with spaces); `$C028` = drive number |
| `$C04A` | entry point flag | b7 = call mode |
| `$C050` | VSALO0 | "SAVE type" code (#00 = SAVEO, #40 = SAVEM…) |
| `$C051` | VSALO1 | flag |
| `$C052`/`$C053` | **DESALO** | start address (source for SAVE, target for LOAD) |
| `$C054`/`$C055` | **FISALO** | end address (SAVE) |
| `$C056`… | LGSALO / FTYPE | length (FISALO-DESALO) / file type |
| EXSALO | EXSALO | execution address (0000 if non-executable) |

## Call sequence — SAVE a buffer

```asm
; saves LGSALO bytes from $4000 into a named Sedoric file.
; 1) fill BUFNOM ($C029..) with "NOM      EXT" (9+3, spaces)
; 2) default values
        jsr $FF76          ; XDEFSA
; 3) override of the zone to save
        lda #$00           ; DESALO = $4000
        sta $C052
        lda #$40
        sta $C053
        clc                ; FISALO = $4000 + size
        lda #$00
        adc XSIZE
        sta $C054
        lda #$40
        adc XSIZE+1
        sta $C055
; 4) save
        jsr $FF7C          ; XSAVEB
```

## Call sequence — LOAD a file

```asm
; 1) fill BUFNOM with the name
        jsr $FF79          ; XDEFLO (defaults)
; 2) DESALO = target address ($4000) if loading at an imposed address
        lda #$00
        sta $C052
        lda #$40
        sta $C053
        jsr $FF73          ; XLOADA  (size read into LGSALO)
```

## Points to validate on a Sedoric environment (not tested here)

- **Exact format of `BUFNOM`** (drive at `$C028`, name/ext left-justified, status
  bytes `pstt` at the end depending on the version — see `$C029..$C038`).
- **Precise role of `VSALO0/VSALO1`** after `XDEFSA` (SAVE type, executable).
- **Call context**: interrupts, work page `$04`, bitmap (BUF2) — the save updates
  the catalog + the bitmap; check that no prior init is required beyond the Sedoric
  boot.
- **Error handling**: `XSAVEB`/`XLOADA` may raise DISK_FULL, etc.

## ✅ Disk writing VALIDATED in the emulator (24/06/2026)

The write chain was **proven end to end** in `oric1-emu`. The previous "blocker"
(vectors `$FF73` not found) was a **false problem**: it was neither the API
addresses nor the ROMDIS mapping, but an **emulator flag**.

### Root cause: write-back is opt-in

`oric1-emu` only writes changes to the host `.dsk` file **if `--disk-writeback`
is passed** (disabled by default to never overwrite a `.dsk` by accident —
`src/main.c:3630`, gate `disk_writeback`). Without this flag, the Sedoric `SAVE`
runs, does write the sectors into the **in-memory** image (FDC primitive at
`$D075`, Type II commands `$A8`/`$AC` on `$0310`), but **nothing is persisted** →
hence the false "it doesn't work" conclusion.

### Validation recipe (reproducible)

```sh
cd ~/Oric1
cp disks/sedoric3.dsk /tmp/sedtest.dsk
KEYS='13000000:\n\p1POKE#4000,65:POKE#4001,66\n\p1SAVE"TEST.BIN",A#4000,E#4002\n\p8'
./oric1-emu -n -r roms/basic11b.rom --disk-rom roms/microdis.rom -d /tmp/sedtest.dsk \
    --disk-writeback -c 32000000 --type-keys "$KEYS" --screenshot /tmp/sedok.ppm
# -> log "Disk write-back: drive A", md5 of the .dsk changes,
#    catalog entry "TEST     BIN" written. Boot = "SEDORIC V3.0".
```

Established facts:
- **Resident Sedoric V3.0 boot**: `-r basic11b.rom --disk-rom microdis.rom -d <dsk>`
  reaches the `Ready` prompt (Sedoric installed). To boot, `-r` is **mandatory**.
- **`SAVE"NOM.EXT",A#deb,E#fin`** from the prompt writes a **real file**
  (catalog + data + bitmap), persisted with `--disk-writeback`.
- `microdis.rom` is `Oric DOS V0.6`: its **page `$FF` is empty** (only the CPU
  vectors `$FFFA-$FFFF`). The API vectors from the PDF (`$FF73`…) are therefore
  not there — the Sedoric V3 API is installed **in RAM overlay** by the boot.

### Consequences for the project

1. **Storage test**: any emulator test of disk storage must pass
   `--disk-writeback`, otherwise the write does not persist (silent trap).
2. **Terminal `client/sedoric.s`**: the goal does **not** require relocating the
   PDF's `$FF73`. The reliable path is to invoke Sedoric the way the BASIC `SAVE`
   does. It remains to determine the **machine call entry** (Sedoric command
   interpreter, or save primitive) to call from the terminal — to be traced from
   the path of the `SAVE` validated above (`$D075` is the FDC sector-write
   primitive; the high-level entry is above it).
3. **Deployment**: the terminal will have to run **under resident Sedoric**
   (booted from a Sedoric `.dsk`), see "The deployment wall" below.

## Chosen path: command injection (decided 24/06/2026)

Rather than calling an internal SAVE routine with a reverse-engineered parameter
convention (fragile, image-specific), the terminal **injects a Sedoric command
line and has it executed** — exactly the validated path of the BASIC `SAVE`.
Robust and close to real hardware.

### Established reverse map (image `sedoric3.dsk`, `oric1-emu`)

Reversed via save-state at the prompt + targeted CPU trace + memory watchpoint
(`memory_set_trace`, type `MEM_READ`). Since `$` appears only in the disassembly
(decimal cycle column), memory accesses are isolated unambiguously.

| Element | Address | How validated |
|---|---|---|
| **Command line buffer** | **`$0035`** | RAM dump: the line `SAVE"…",A#…,E#…` resides there (Oric BASIC input buffer) |
| **Buffer scanner (self-modifying)** | **`$00E2`–`$00ED`** | trace: the operand of `LDA $00E8` is the byte `$E9/$EA`, incremented to advance; skips spaces (`CMP #$20`) |
| Buffer read (absolute) | `LDA $0035`, `$0039`…`$0051` | trace: byte-by-byte reads of the buffer during dispatch |
| Sedoric keyword table | ~`$CA6F` | RAM dump: `SAVE FIELD RSEC INIT INSTR…`; matched via pointer `$DE/$DF`, quote separator `$22` |
| String-compare helper | `$D5B5` | trace: `LDA ($DE),Y` / `CMP $24/$25` |
| Save routine (cluster) | `$D33A`/`$D342`/`$D398`/`$D39E` | trace: lightly nested JSRs just before the write |
| **FDC sector-write primitive** | **`$D075`** | trace: Type II commands `$A8`/`$AC` on `$0310` |
| Page 4 trampolines | `$04EF`→`JMP $C4A0`, `$0474`, `$0477` | trace: indirect jumps RAM ↔ Sedoric |

### Conclusion: dispatch is interleaved with the BASIC ROM

Decisive point of the reverse: **`SAVE` is NOT dispatched by an isolable Sedoric
entry**. When you type the command + Return, it is the **BASIC ROM**
(`$F6xx`–`$F8xx`, line-input routines) that processes the line **then** calls the
Sedoric scanner; `$C4A0` (prompt core) is executed only **once while idle**, not
on the `SAVE` path. The dispatch depends on numerous zero-page variables
(`$A9` position/line-ready, `$24/$25`, `$DE/$DF`, `$E9/$EA`, `$2E`, `$0252`,
`$02F2`…) set by the BASIC input chain.

**Consequence**: calling `SAVE` from **standalone** machine code (the terminal)
does not reduce to a `JSR <entry>` with the command at `$0035`. You must either
faithfully reproduce the BASIC input context (fragile, specific to this image),
or use a documented Sedoric mechanism to execute a command from ML.

### Recommended approaches (by decreasing robustness)

1. **Documented Sedoric mechanism** — find in the "Sedoric à nu" doc (or the
   manual) the **official ML command-execution entry** (Sedoric exposes one:
   command in buffer + call to a stable entry point, designed for this). This is
   the only *version-portable* and *real-hardware* path. To be preferred.
2. **Keyboard injection (type-ahead)** — deposit the command in the keyboard
   buffer and return control to the Sedoric prompt, which executes it "as typed".
   Robust, but assumes the terminal can return cleanly to the prompt.
3. **Reproduction of the BASIC context** — set `$0035` + all the zero-page
   variables above and enter the BASIC chain. **Not recommended**: fragile, not
   portable, to be revalidated at every Sedoric version.

> Quick trace recipe (save-state at the prompt) to iterate:
> `--load-state sed.state --type-keys '…SAVE…\n' --trace t.log --trace-max 4000000`
> then grep the `$0035` accesses (buffer read) and `$D075` (FDC write).

### Deployment validated without disk repackaging

The terminal `client/term.s` is a **cassette**; yet `tap2sedoric` (`oric1-emu`
tool) is an **unimplemented stub** → no direct `.dsk` fabrication. **Realistic and
testable** deployment path: boot Sedoric, then **`CLOAD` the terminal from the
cassette** — Sedoric **stays resident**, the terminal runs with the disk API
available. (Conceptually validated: resident Sedoric + tape.)

## Documented API confirmed + V1.0 doc / V3.0 image gap (24/06/2026)

The **"Sedoric à nu"** doc (`sednb3_0.pdf`) gives the official API. Checks made
against the emulator's **sedoric3.dsk (SEDORIC V3.0)** image.

### ✅ Public vector table — IDENTICAL V1.0 and V3.0

By dumping the **CPU view `$C000-$FFFF` during a SAVE** (overlay mapped), we read
on V3.0 exactly the PDF vectors (which describes SEDORIC 1.0):

| Vector | Content (V3.0 measured) | = PDF V1.0 |
|---|---|---|
| `$FF7C` XSAVEB | `4C 9C DE` → JMP `$DE9C` | ✅ identical |
| `$FF76` XDEFSA | `4C 28 DE` → JMP `$DE28` | ✅ identical |

→ **The table `$FF43-$FFC6` is the stable interface** (that is its role). The
system variables (`$C04D` VSALO0, `$C051` FTYPE, `$C052` DESALO, `$C054`
FISALO…) are at the same addresses. The `client/sedoric.s` sequence (set the
variables + `JSR $FF7C`) is therefore **correct**.

### Overlay toggle — version-specific (V3.0 resolved)

To make the `$C000-$FFFF` routines visible, you must switch to the overlay RAM.
**The toggle address changes with the version**:

| Version | Overlay toggle | Source |
|---|---|---|
| Sedoric 1.0/2.x | `JSR $0472` | PDF "Sedoric à nu" |
| **Sedoric 3.0** | **`JSR $04F2`** | disassembled SEDORIC 3.0 manual, ANNEXE 15 |

On V3.0, `$0472` is not the toggle (page 4 reorganized by the 3.0 **bank**
management) → `JSR $0472` **crashes**. The 3.0 manual documents it explicitly: "in
machine language, do a `JSR $04F2` to access the overlay RAM, call the desired
subroutines, and finish with another `JSR $04F2` to return". A raw `$0314` write
crashes (XSAVEB requires the runtime context).

### ✅ V3.0 recipe VALIDATED end-to-end (24/06/2026)

Tested in the emulator on `sedoric3.dsk` (a file `TESTML  BIN` **written and
persisted** in the `.dsk` — catalog entry + write-back, md5 modified):

```asm
        jsr $04F2          ; ROM -> RAM overlay (toggle V3.0)
        ; set BUFNOM ($C029), VSALO0 ($C04D=#00), FTYPE ($C051=#40),
        ;   DESALO ($C052=$4000), FISALO ($C054=end), LGSALO ($C04F=size),
        ;   EXSALO ($C056=0), VSALO1 ($C04E=0)
        jsr $DE9C          ; XSAVEB (direct entry = target of vector $FF7C)
        jsr $04F2          ; RAM overlay -> ROM
```

`client/sedoric.s` implements this recipe (`OVL_TOGGLE = $04F2`,
`XSAVEB = $DE9C`, detection "XSAVEB starts with `SEI $78`"). To target 1.x/2.x,
set `OVL_TOGGLE = $0472`. **The uncertainty is lifted**; the `term.s` →
`sed_save` integration (after a download, size in `XSIZE`) is already wired.

### Sedoric presence guard (safe without disk)

`sed_save` first checks, in **always-mapped page 4 RAM** (before any
`JSR $04F2`), the **jump table** that Sedoric installs at boot at `$04F2`/`$04F5`
(`4C xx 04` = `JMP $04xx`). Validated:

- **Under Sedoric**: `$04F2 = $4C`, `$04F4 = $04` → guard OK, file saved
  (`TESTG4 BIN` written to the `.dsk`).
- **Without disk** (cassette terminal, Atmos alone): `$04F2 = $55` (RAM pattern) →
  guard refuses, **no** `JSR $04F2`, no crash.

Thus the same terminal is safe on cassette **and** under Sedoric; the save
activates only if Sedoric is actually resident. What remains is the **deployment**
of the terminal under resident Sedoric (booted from disk or `CLOAD`).

> Validation detail: the `--type-keys` harness sometimes loses the 1st character
> of a line (BASIC line number, with no impact on the DATA values) — provide a
> purge `\n` at the head. Confirmed first by the "HELLO ANDRE" example of ANNEXE
> 15 (`JSR $04F2`/`JSR $D637`/`JSR $04F2`), then by XSAVEB.

## ✅ Bootable terminal disk (deployment path B)

`client/build-disk.sh` builds a Sedoric disk containing the terminal,
**reproducibly** (validated in the emulator):

1. assembles `term.bin` (`build.sh`);
2. builds a **non-autorun** cassette of the terminal (autorun byte `$C7` → `$00`);
3. drives `oric1-emu`: boot Sedoric (master) + **fast-load** of the cassette — the
   terminal is injected into RAM `$1000` at ~3M cycles (phase 1 of the fast-load,
   *without CLOAD*) and **survives the Sedoric boot**; at the prompt,
   `SAVE"TERM",A#1000,E#1E26` writes **TERM.COM** onto a copy of the master;
4. `--disk-writeback` persists → `client/term-boot.dsk`.

**Launching the terminal** from the disk (validated — the modem menu appears):

```
LOAD"TERM":CALL#1000
```

> At the "TYPE DE MODEM" menu, choose **LOCI `$0380`** if a Microdisc is present:
> the ACIA `$031C` overlaps the Microdisc I/O range `$0310-$031F`. The terminal
> handles both addresses at runtime (`ACIAPTR`), no build variant is required.

**Fine-tuning notes**:
- The terminal **runs** under Sedoric (≈2.6 M instructions executed, menu
  displayed); the initial `BREAK ON BYTE #1000` came from the Sedoric `,J` option
  (`LOAD"TERM",J`), **not** from a runtime conflict → use `LOAD` + `CALL`.
- *Hands-free* auto-start: mechanism identified — at boot, Sedoric looks for
  **`BOOTUP.COM`** and executes `!BOOTUP` (SEDORIC 3.0 manual, disassembly
  `; found BOOTUPCOM ? executes !BOOTUP`). **But** on `sedoric3.dsk` (master/
  tools) the "WELCOME TO SEDORIC DOS V3.0" menu is **not** a replaceable directory
  file: `DESTROY"BOOTUP.COM"` answers *FILE NOT FOUND* → the menu is **built into
  the master's system** (on the master, `DESTROY"BOOTUP.COM"` = *FILE NOT FOUND*
  while the menu runs anyway).
- **Emulator blocker for hands-free**: creating a blank Sedoric disk on which to
  place `BOOTUP.COM` requires `INIT` (formatting) → FDC *Write Track*. But in
  `oric1-emu`, `FDC_OP_WRITE_TRACK` is set (`src/storage/disk.c`) but **without a
  data handler** = **no-op**: formatting writes nothing. `INIT` therefore cannot
  produce a bootable disk in the emulator.
- **Consequence**: hands-free auto-start **cannot be validated in the emulator**.
  On **real hardware** (where `INIT` formats normally), the path is: `INIT` a
  minimal Sedoric disk → copy `TERM.COM` onto it → create `BOOTUP.COM` = launcher
  (`LOAD"TERM":CALL#1000`). As it stands (emulator **and** master), **one command**
  launches the terminal (`LOAD"TERM":CALL#1000`).

## The deployment wall

The terminal `client/term.s` is today an **autorun cassette** (`$1000`) on a
**diskless** machine. To call these vectors, Sedoric must be **resident**. Two
options to decide:

1. **Terminal on a Sedoric disk**: build a Sedoric `.dsk` containing the terminal,
   boot Sedoric, `LOAD`+`RUN` it. The terminal then has access to the API.
2. **Terminal loaded after Sedoric boot** (cassette): boot Sedoric, `CLOAD` the
   terminal — Sedoric stays resident.

The full **test** will require a working Sedoric `.dsk` image + a debug cycle in
the emulator (`--disk-rom microdis.rom --disk sedoric.dsk`).

See also: `docs/transfer.md`, `docs/agile/backlog.md` (G1).
