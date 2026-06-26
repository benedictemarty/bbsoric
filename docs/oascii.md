# OASCII layer — Oric display

## What is "OASCII"?

A project brand name (a nod to **PETSCII** and **ATASCII**), BUT note the
fundamental difference:

| Machine | "xASCII" | Real nature |
|---------|-----------|---------------|
| C64 | PETSCII | **proprietary** character set (codes ≠ ASCII) |
| Atari | ATASCII | **proprietary** character set (codes ≠ ASCII) |
| **Oric** | **OASCII** | **standard ASCII** for characters + **serial Teletext attributes** |

On the Oric, the letter `A` is at code `65` just like in ASCII. "OASCII" therefore
does **not** designate a character encoding, but the **display model** of the Oric: the
Teletext-style 40×28 TEXT mode, where colors and attributes are set by **control
bytes (0–31) that occupy a screen cell** and apply until the end of the line.

## Attribute table (source of truth)

Extracted from the ULA decoder of the reference emulator **`Oric1/oric1-emu`**
(`src/video/video.c`, function `decode_attr`). A screen byte is a serial attribute
if its bits 6 and 5 are zero (value 0–31); effect according to `val & 0x18`:

| Byte | Group | Effect |
|-------|--------|-------|
| `0–7`   (`0x00`) | **INK** | ink = `val & 7` |
| `8–15`  (`0x08`) | **text attributes** | bit0 = alternate charset · bit1 = double height · bit2 = blink |
| `16–23` (`0x10`) | **PAPER** (background) | background = `val & 7` |
| `24–31` (`0x18`) | **video mode** | changes `vid_mode`; in addition: `28` = inverse OFF, `29` = inverse ON |

> The ULA **resets** attributes at the start of **each line**: ink = white (7),
> background = black (0). A color therefore does not "spill over" onto the next line.

## Cursor positioning ("plot X,Y")

Beyond the sequential stream, an **Oric-terminal-specific extension** allows
positioning the write cursor at absolute coordinates:

| Sequence | Effect |
|----------|-------|
| `1F` `col` `row` | places the cursor at (`col` 0–39, `row` 0–27); the following bytes are written starting from there |

The byte `0x1F` (outside the actually emitted attribute ranges) is followed by
**two raw bytes** (column then row). The terminal (`client/term.s`,
`handle_rx`/`set_cursor_xy`) intercepts the sequence and repositions its VRAM
pointer. Go API: `oascii.Plot(col, row)` or `Builder.At(col, row)`. Generic
terminals (telnet/PC) do not understand this command — it is an Oric feature
(used e.g. to position the fields of a form within a layout).

## Palette (8 colors, R/G/B bits)

From `palette[8][3]` in `video.c`:

| # | Name | RGB |
|---|-----|-----|
| 0 | Black   | `000000` |
| 1 | Red     | `FF0000` |
| 2 | Green   | `00FF00` |
| 3 | Yellow  | `FFFF00` |
| 4 | Blue    | `0000FF` |
| 5 | Magenta | `FF00FF` |
| 6 | Cyan    | `00FFFF` |
| 7 | White   | `FFFFFF` |

## Go API (`internal/oascii`)

```go
b := oascii.New()                  // default Oric state: ink white, paper black
b.Ink(oascii.Yellow)               // emits byte 0x03
b.Paper(oascii.Blue)               // emits byte 0x14 (16+4)
b.Blink(true)                      // emits byte 0x0C
b.Text("BBS ORIC")                 // printable ASCII (0–31 → safety space)
b.Newline()                        // CR LF (+ re-emit if Sticky)
sess.Write(b.String())             // bytes 0–31 preserved
```

Low-level encoders tested against the emulator: `InkAttr(c)`, `PaperAttr(c)`,
`TextAttr(blink, doubleHeight, altCharset)`.

### "Sticky" mode
`b.Sticky(true)` automatically re-emits the current attributes (not by default)
after each `Newline()`, to keep a color across several lines despite the
per-line reset by the ULA. Cost: each re-emission consumes a cell at the
start of the line.

## Layout pitfalls (40 columns)

- **An attribute byte occupies a cell.** A "full-width" line of 40 characters
  **preceded** by an attribute is 41 cells → it spills over onto the next line. For a
  colored full-width line, use only `Cols-1` characters, or leave it in the
  default color (no byte emitted).
- Centering must account for the attribute bytes placed before the text (a one-column
  offset per attribute).

## Validation on emulator

```bash
go run ./cmd/bbsd -addr 127.0.0.1:6502
cd ~/Oric1 && ./oric1-emu --serial tcp:127.0.0.1:6502 --acia-addr 031C
```
See [`emulator-testing.md`](emulator-testing.md). The hexdump of the server stream lets you
verify the attribute bytes (e.g. `03` = yellow ink before the title).
