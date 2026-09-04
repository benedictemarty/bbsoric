# ADR-0007 — Differential HIRES buffer for animation

- **Status**: Accepted
- **Date**: 2026-09-05
- **Deciders**: bmarty
- **Related to**: ADR-0005 (HIRES pages: unified command stream), the differential
  TEXT buffer (`oascii.Screen`)

## Context

ADR-0005 shipped **static** HIRES pages: a page is serialised once (`render.Hires`)
as a full command stream (`1F FC` + `HiOn` + optional bitmap blit + primitives +
`HiEnd`) and rendered by the terminal (`client/hires.s`). Resending an entire HIRES
frame is expensive: 8000 VRAM bytes (`$A000`), i.e. several seconds at 9600 baud even
RLE-compressed when the image is not uniform. That kills **animation** (a moving
sprite, a game, a gauge) where most of the screen is unchanged between frames.

TEXT mode already solves this with `oascii.Screen`: keep the *composed* and the
*shown* state, emit only changed cells. HIRES needs the same idea.

## Decision

Add a **server-side differential HIRES buffer** `oascii.HiresScreen`, the graphical
twin of `oascii.Screen`.

1. **Server-side only, no firmware change.** The existing `HiBlit` opcode already
   writes *N* bytes at `$A000+offset` and is emulator-validated. A differential frame
   is therefore just a set of `HiBlit` runs. The terminal is untouched.

2. **Compose → diff → emit runs.** `HiresScreen` holds the composed VRAM (8000 bytes)
   and the last-shown VRAM. A Go rasteriser draws into the composed buffer
   (`SetPixel`/`Line`/`Box`/`FillBox`/`Circle`/`SetBitmap`, monochrome, same pixel
   model as `hires.s`: 40 bytes/row, 6 px/byte, empty byte `0x40`). `Render()` diffs
   composed vs shown and emits **one `HiBlit` per changed run** (RLE-compressed),
   wrapped in `1F FC … HiEnd`, then updates shown.

3. **First frame switches, later frames only diff.** The first `Render()` emits `HiOn`
   (mode switch + clear → shown baseline `0x40`); subsequent frames emit **only**
   `HiBlit` runs (the terminal stays in HIRES between frames). `Render()` returns nil
   when nothing changed. `Reset()` forces a full re-emit (reconnection).

4. **Erasure is free.** The animation loop recomposes each frame (`Clear` + draw); the
   diff naturally emits both the vanished region (back to `0x40`) and the new one.

Colour stays out of scope for now: animation is monochrome (default ink). Per-line
ink/paper attributes remain the static-page model (`render.Hires`).

## Consequences

- **Animation becomes practical** on the serial link: a moving object only costs its
  own footprint per frame, not 8000 bytes. Demonstrated by `hiresanim` (bouncing ball,
  fixed frame) — only the ball is re-emitted each frame.
- **Single source of correctness.** A round-trip test replays the emitted stream into a
  VRAM and checks it reconstructs the composed buffer exactly (erasure + draw), so the
  wire is provably correct. Runtime smoke-tested in `oric1-emu`
  (`scripts/test-emulateur-hires-anim.sh`: the real firmware renders the differential
  stream and the sampled VRAM takes several distinct substantial states — the screen
  lives). Serial timing prevents reliably capturing two *clean* sprite frames, so the
  correctness proof stays the deterministic unit tests, not the emulator sampling.
- **Efficiency caveat.** The diff wins when the background is *complex* (poorly RLE-
  compressible) and changes are *localised*. A change scattered across many rows (e.g.
  a shifting outline) can cost more than a uniform full frame; authors should keep the
  moving part localised. Documented in `docs/hires.md`.
- **No flow control (yet).** There is no back-pressure from the terminal: the server
  must not push frames faster than the link + 6502 rendering can absorb, or the serial
  FIFO overflows and the HIRES stream desyncs. Two rules for animation authors, learnt
  the hard way with the demo: (1) keep each frame's runs **contiguous** — a
  full-screen *border* costs ~200 one-byte blits (left/right column, one pixel per
  row), a *horizontal* rail costs one; (2) **pace** frames (the demo uses a small
  filled sprite between horizontal rails at ~3 fps). Real flow-controlled transfer is
  the separate Sprint-10 "Later" item.
- **Future**: coalesce runs separated by tiny gaps; optional colour animation; a
  non-blocking input path so an animation can be interrupted by a keypress.
