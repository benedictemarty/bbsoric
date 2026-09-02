# Phosphoric team message — Jasmin disk write-back never fires (`.dsk` not persisted)

*(All references are to the Phosphoric / `oric1-emu` tree in `~/Oric1`. Nothing
invented — line numbers from the current sources.)*

Hi,

Short version: **the Jasmin FDC write path works in RAM, but a guest `SAVE` is
never written back to the `.dsk` file** — even with `--disk-writeback`. The
write-back routines only look at the **Microdisc**'s dirty flags, never the
Jasmin's. This blocks us on the bbsoric side (see "Why we need this" below).

## What works today

The Jasmin write path is faithful and *does* mark the drive dirty:

```c
// src/io/jasmin.c:95
void jasmin_write(jasmin_t* j, uint16_t addr, uint8_t value) {
    if (addr >= JASMIN_FDC_BASE && addr <= (JASMIN_FDC_BASE + 3)) {
        fdc_write(&j->fdc, (uint8_t)(addr & 3), value);
        if (j->fdc.disk_modified) {
            j->disk_dirty[j->drive & 3] = true;   // <-- Jasmin dirty flag set
            j->fdc.disk_modified = false;
        }
```

`jasmin_t` carries its own dirty array (`include/io/jasmin.h:83`:
`bool disk_dirty[JASMIN_MAX_DRIVES];`), and the emulator has both interfaces
(`include/emulator.h`): `microdisc_t microdisc; … bool has_microdisc;` and
`jasmin_t jasmin; … bool has_jasmin;`.

## The bug

Both write-back routines gate **only** on `emu->microdisc.disk_dirty[drv]`:

```c
// src/main.c:603  osd_writeback_drive()
if (!emu->disk_writeback || drv < 0 || drv >= MICRODISC_MAX_DRIVES) return false;
if (!emu->microdisc.disk_dirty[drv] || !emu->disks[drv] || !emu->disk_paths[drv])
    return false;
bool ok = sedoric_save(emu->disks[drv], emu->disk_paths[drv]);
...
emu->microdisc.disk_dirty[drv] = false;
```

```c
// src/control.c:600  control_writeback_drive()  — same logic, same gate
if (!emu->microdisc.disk_dirty[drv] || ...) return false;
```

`grep -rn 'jasmin.disk_dirty' src/` → **no hits**: nothing ever consumes
`emu->jasmin.disk_dirty`. So on a Jasmin machine the sequence is:

1. guest `SAVE` → `jasmin_write` → `fdc_write` mutates the in-RAM image,
   sets `emu->jasmin.disk_dirty[drv] = true`;
2. eject / hot-swap / quit → `*_writeback_drive` checks
   `emu->microdisc.disk_dirty[drv]` (still `false`) → **returns early, no flush**;
3. the `.dsk` on disk is unchanged → **the save is lost**.

## Suggested fix (your side — we won't touch Phosphoric)

In both `osd_writeback_drive()` (`src/main.c`) and `control_writeback_drive()`
(`src/control.c`), consult the *active* interface's dirty flag. Roughly:

```c
bool dirty = emu->has_jasmin ? emu->jasmin.disk_dirty[drv]
                             : emu->microdisc.disk_dirty[drv];
if (!emu->disk_writeback || drv < 0 || drv >= /* max for the active iface */) return false;
if (!dirty || !emu->disks[drv] || !emu->disk_paths[drv]) return false;
... sedoric_save(...) ...
if (emu->has_jasmin) emu->jasmin.disk_dirty[drv] = false;
else                 emu->microdisc.disk_dirty[drv] = false;
```

Two things to confirm on your side while you're there:
- **Same buffer.** That `sedoric_save(emu->disks[drv], …)` writes the *same*
  `emu->disks[drv]->data` buffer the Jasmin FDC mutates (i.e. `jasmin_set_disk`
  was wired to `emu->disks[drv]->data`), otherwise the flush would persist a
  stale image.
- **Drive bound.** The `drv >= MICRODISC_MAX_DRIVES` guard should use
  `JASMIN_MAX_DRIVES` when `has_jasmin` (they may differ).

## Why we need this (bbsoric context)

The BBS Oric terminal saves downloaded files **directly to disk** at reception.
Two sinks exist and are validated in the emulator: **LOCI** (SD, streaming) and
**Sedoric/Microdisc** (`XSAVEB`, buffered + numbered slices). We want to add a
**Jasmin** sink, and we validate every sink **byte-exact against Phosphoric**
(cf. the oterm K6 VRAM proof). For a "save to a Jasmin `.dsk`" test to mean
anything, Phosphoric must **persist** the guest's sector writes to the image
file. The read/boot path is already faithful (thanks!); the missing piece is the
write-back wiring above.

Minimal repro once fixed: boot a Jasmin ROM + a writable `.dsk` with
`--jasmin-rom … --disk-writeback`, `SAVE` a small file from the guest, quit, and
check the `.dsk` grew a catalogue entry (same shape as the Sedoric/Microdisc
write-back we already rely on).

Thanks!
