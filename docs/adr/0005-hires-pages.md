# ADR-0005 — HIRES pages: unified command stream (bitmap + primitives)

- **Status**: Accepted (server foundation); terminal + studio increments to follow
- **Date**: 2026-06-27
- **Deciders**: bmarty
- **Related to**: ADR-0001/0002 (JSON content, page types), ADR-0004 (DataWindow), the
  differential TEXT buffer (`oascii.Screen`)

## Context

Everything so far is **TEXT mode** (Oric 40×28 teletext, « OASCII »). The terminal
firmware (`client/term.s`) is text-only; it routes control on the `0x1F` byte
(`1F col row` = plot, `1F FE/FD` = XMODEM). The owner needs **HIRES graphics pages**,
and explicitly **both** of the two natural models:

- **bitmap** — a full-screen image / logo posted in one block;
- **primitives** — vector drawing commands (line, box, circle, …).

The hard constraint is the **serial link**: an Oric HIRES screen is **240×200 = 8000
bytes** of VRAM at `$A000`; a raw dump is ~8 s at 9600 baud. The differential TEXT
buffer exists for exactly this reason in text mode.

## Decision

1. **One wire protocol carries both models.** A new serial sub-command **`1F FC`**
   (0xFC is free and outside the valid column range 0..39, so unambiguous with a plot)
   opens a **HIRES command stream**: a sequence of 1-byte opcodes (+ fixed args, except
   `Blit` which carries its length) terminated by `HiEnd`. Primitives and the bitmap
   *blit* are just different opcodes in the same stream, so an author can combine a
   bitmap background **and** primitives on top.

2. **Opcodes** (`internal/oascii/hires.go`): `HiOn` (switch + clear), `HiInk`/`HiPaper`,
   `HiCurset`/`HiPoint`/`HiLine`/`HiBox`/`HiFillBox`/`HiCircle`/`HiChar`, and `HiBlit`
   (write N bytes to `$A000+off`). Coordinates fit in **one byte** (x:0-239, y:0-199).
   The terminal keeps a **pen** moved by curset/point/line.

3. **Bitmap is RLE-compressed** (`RLEEncode`/`RLEDecode`, count/value pairs) — enough
   for logos (large uniform runs). Reserved for **static** screens loaded once; refreshes
   use primitives. A differential HIRES buffer (mirroring `oascii.Screen`) is a later
   increment if animation needs it.

4. **Primitives are implemented IN the terminal**, not via BASIC ROM HIRES routines.
   The firmware is bare-metal (`sei`, keyboard+serial in-house); depending on the ROM
   interpreter's zero-page/context would be fragile. Self-contained setpixel/Bresenham/
   rect/fill is the safer path and matches the existing philosophy.

5. **Mixed mode.** HIRES occupies the top of the screen; the **bottom 3 lines stay TEXT**
   (`$BB80`) — reused for a menu/status, exactly the existing « menu over a background
   screen » pattern. A HIRES page with `entries` routes keys like a raw-background menu;
   without entries it waits for a key.

6. **Content model in `internal/content`** (`Hires`/`HiresOp`, page field `hires`),
   validated by `Site.Validate()` (bitmap size 8000, bounds 240×200, colours 0-7, known
   ops). **`render.Hires`** is the single source of the wire stream (server + studio),
   mirroring `render.Screen`.

## Increments

1. **Server foundation** (this ADR): content model + validation + `render.Hires` encoder
   + RLE + engine wiring + tests. ✅
2. **Terminal firmware**: HIRES interpreter in `term.s` (mode switch, primitives, RLE
   blit), visual validation in `oric1-emu`.
3. **Studio Forge**: HIRES editor (240×200 preview, primitive editor / image import).

## Consequences

- **+** Both models from day one, one protocol, compact on the wire (RLE + 1-byte opcodes).
- **+** Backward compatible: generic telnet clients ignore `1F FC …`; text pages untouched.
- **−** The terminal must implement primitives in 6502 (more firmware than calling the ROM)
  — accepted for robustness.
- **−** No HIRES differential yet; animation-heavy screens may need it later.

## Alternatives considered

- **Raw bitmap only** — too slow on serial for anything but a single splash; rejected as
  the *sole* model (kept as one of the two, compressed).
- **Call BASIC ROM HIRES routines** — compact firmware, but fragile (interpreter context);
  rejected in favour of self-contained primitives.
