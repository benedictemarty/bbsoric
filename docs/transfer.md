# File transfer — download / upload (XMODEM)

The BBS Oric offers a **file library** (the "mass storage" on the server
side) from which you **download** and to which you **upload**, via the
historic **XMODEM** protocol.

> **Status.** On the **server** side: implemented and tested (XMODEM download/upload,
> on-disk library). On the **Oric terminal** side: the XMODEM receiver/sender
> and writing to mass storage (SD card via LOCI, Microdisc, cassette)
> remain to be done in `client/term.s` (cf. backlog **G1**). In the meantime,
> transfers are tested with a **standard XMODEM client** (PC: `sx`/`rx`, or a
> terminal emulator supporting XMODEM).

## Enabling the library

```
bbsd ... -files /var/lib/bbsoric/files -max-upload 65536
```

- `-files <dir>`: library directory (created if absent). Empty = transfer
  disabled (the applets display "Bibliotheque indisponible").
- `-max-upload <bytes>`: max size of an upload (default 64 KB; 0 = unlimited).

File names are **validated** (simple name, no `/`, `\` or `..`) to
prevent any escape from the directory.

> **Two consumers of the `-files` library.** The low-level `download`/`upload` applets
> below operate on the **shared** `-files` directory. In the production content they are
> **no longer wired flat** into the "Fichiers" menu: public downloads go through the
> **Catalogue** (`datawindow`, `docs/datawindow.md`) and the "Fichiers" section is now the
> **personal per-account space** (`mesfichiers`, `docs/userfiles.md`). Both reuse the same
> XMODEM plumbing documented here.

## Wiring the applets into the content

Two applets are provided: **`download`** and **`upload`**. They are wired as
menu entries (type "▶ applet", selectable in the studio):

```jsonc
{ "title": "FICHIERS", "entries": [
  { "key": "T", "label": "Telecharger", "applet": "download", "next": "fichiers" },
  { "key": "E", "label": "Televerser",  "applet": "upload",   "next": "fichiers" },
  { "key": "R", "label": "Retour",      "target": "__back__" }
]}
```

- **`download`**: lists the files (choice by digit 1–9), then **sends** the
  file to the client over XMODEM (the client starts a **reception**).
- **`upload`**: asks for a name, then **receives** the file over XMODEM (the client
  starts a **send**) and saves it in the library.

## Technical details

- **Protocol**: `internal/xmodem` — 128-byte blocks, checksum **or**
  CRC-16 (imposed by the receiver via `NAK`/`C`), re-transmission on error. The
  last block is padded with `SUB` (0x1A), trimmed on reception.
- **Raw channel**: during a transfer, the applet uses `Session.Raw()` which
  bypasses the telnet/line filtering (binary reading). `Session.ClearDeadline()`
  then restores the normal inactivity timeout.
- **XMODEM limitation**: the exact size is not transmitted (`SUB` padding) — faithful
  for text; for a binary that truly ends with 0x1A, plan an envelope
  format (YMODEM) later.

### Download size limit — the binding constraint is RAM, not the header (31/08/2026)

The download **header** codes the real size on **16 bits** (`downloadHeader`, so
`maxDownloadSize = 0xFFFF`), which *looks* like the 64 KB ceiling. It is **not** the
real limit. The Oric terminal receives the whole file into a **RAM buffer at `$4000`**
(`client/xmodem.s`), then saves the buffer to disk. On an Oric-1/Atmos, RAM ends at
`$BFFF` (`$C000`+ = ROM) and the TEXT screen sits at `$BB80`, so the buffer plateaus at
roughly **`$4000..$BB80` ≈ 31.6 KB** (48 KB at the very most). **A 64 KB file cannot fit
in the buffer at all**, regardless of the header width.

Consequence: widening the header size field (the former **I2b**) unlocks **no** real
capacity, would push `XSIZE` to 3 bytes (touching the gauge and save paths, risking the
**hardware-validated ≤32 KB path**), and would let the server offer a file the client
cannot receive. The real enabler for large files is **streaming reception straight to
disk** (write each XMODEM block to Sedoric/LOCI instead of buffering all of `$4000`),
which is a substantial firmware change requiring emulator validation. I2b is requalified
accordingly in `docs/agile/backlog.md`.

#### Streaming reception straight to LOCI SD — done (I2b-a, 01/09/2026)

The buffer ceiling is **lifted on LOCI-equipped machines**. LOCI writes incrementally
(`OPEN` → `WRITE_XSTACK` per block → `CLOSE`), so nothing forces the whole file into
`$4000` — only the fact that the classic receiver filled `$4000` first. For a file
**larger than 30720 bytes** (`$4000..$B7FF`), `handle_rx` now edits the filename, opens
the SD file (`loci_open`), and receives in **streaming mode**: `xmodem_recv` (flag
`xstream`) writes each validated 128-byte block **directly to the SD card**
(`loci_write_chunk`) **before** the ACK — the sender is stop-and-wait, so it waits for
the ACK and no serial byte is lost during the synchronous SD write — never keeping more
than 128 bytes in RAM. The last block is truncated to the real size (`xdrem` counter,
XMODEM padding dropped) and the file is closed at EOT.

- **File-size ceiling is now the SD card** (and, since the **v4 header** widened the size
  field to **3 bytes**, up to the gauge limit **~8 MB** — see I2b-c below; before v4 the
  16-bit size field capped it at 64 KB).
- **Sedoric-only machines: large files via numbered slices** (I2b-b, done 01/09/2026) —
  `XSAVEB` saves one **contiguous** RAM region in a single call and cannot stream, so a
  Sedoric-only terminal instead **accumulates** in `$4000` like the buffered path and, **each
  time the buffer fills** (~30 KB), saves a numbered slice `FICHIER.001`, `.002`… then resets
  `$4000`; the final partial slice is saved at EOT at the real size. New sink `xsink=2`
  (`xr_sed_write_slice`/`xr_sed_final`, `set_slice_ext` for the `.00N` extension, `sed_present`
  probe). Host-side the slices are concatenated back. Proven in `oric1-emu` under resident
  Sedoric (`scripts/test-stream-sedoric-emu.sh`: two slices persisted to the `.dsk`, catalog
  entries + data byte-checked).
- **Latent bug fixed on the way**: the server offers files up to 64 KB, so a >30 KB file
  reaching a **non-LOCI** terminal used to overflow the buffer *and then still save the
  partial buffer* under the real (larger) size. `xmodem_recv` now returns **A=1/0**
  (success/abort) and `handle_rx` saves **nothing** on overflow/CAN.
- **Runtime proof**: `scripts/test-stream-loci-emu.sh` (gated; skipped without
  `oric1-emu`/ROM). A Go XMODEM sender (`internal/xmodem.Send`) ships a **40000-byte**
  file (> 30720, not a multiple of 128) over TCP serial; a standalone 6502 harness drives
  `xmodem_recv` in streaming mode; the host file written by the LOCI backend is
  **byte-identical** to the source. Gotcha: under `--loci-flash` the default ACIA moves to
  the LOCI picowifi modem at `$0380`, so the test forces `--acia-addr 031C` to keep the
  serial ACIA at `$031C` alongside the LOCI MIA at `$03A0`.

#### Files > 64 KB — 24-bit download header (I2b-c, 05/09/2026)

Now that reception **streams to disk**, the download header's 16-bit size field became the
last thing pinning the ceiling at 64 KB. The header is widened to **v4**: the real-size
field goes from **2 to 3 bytes** (little-endian lo/mid/hi), lifting the theoretical limit to
16 MB; the **binding** cap is now the gauge (block count on 2 bytes ⇒ `0xFFFF × 128 ≈ 8 MB`),
so `maxDownloadSize = 0xFFFF*128`. End-to-end:

- **Server** (`server/internal/bbs/xfer.go`): `downloadHeader` appends the third size byte;
  `maxDownloadSize` raised. `TestDownloadHeader` covers a 100000-byte file (bit 16 set).
- **Firmware** (`client/term.s`, `client/xmodem.s`): `dlsize` and `xdrem` become **3 bytes**;
  a new header state (PLOTST 9) reads the high size byte; `hr_is_large` treats any nonzero
  high byte as large; the streaming counters (`xr_flush`, `xr_sed_write_slice`, `xr_sed_final`)
  do **24-bit** subtraction. `XSIZE` stays 16-bit (only the ≤30 KB buffered path uses it).
- **oterm** (`pcterm/internal/xfer`): reads the 17-byte header (3-byte size); loopback test
  updated.
- **Runtime proof**: `scripts/test-download-large-emu.sh` streams an **80000-byte** file
  (> 65535) through `xmodem_recv`; the LOCI host file is **byte-identical** — proving the
  24-bit `xdrem` decrement over > 512 blocks. (Thin wrapper over the LOCI streaming harness,
  whose `xdrem` is now seeded on 3 bytes.)

A **v3 terminal** (2-byte size) is not wire-compatible with a v4 server: terminal and server
must be flashed together (both are in this repo).

## Oric side

- **Download: done.** `client/xmodem.s` implements the **6502 XMODEM receiver**
  (checksum mode). The server sends the **`1F FE`** sequence (`oascii.RecvCmd`)
  before sending; `term.s` (`handle_rx`) then switches to `xmodem_recv`, which receives
  the file into **RAM (`$4000`)** and displays "FICHIER RECU EN 4000". Validated in
  the emulator (`docs/img/xmodem-download.png`).
  - **Download header v3.** After `1F FE` the server sends a fixed-length header
    (`downloadHeader`) for deterministic 6502 parsing: the **2 block-count bytes**
    (gauge), the **12-byte Sedoric 8.3 filename** (`sedoricName`), then the **2
    real-size bytes** (lo, hi). The terminal saves under that **real name**
    (`dlname` → `sed_save`) instead of the fixed `BBSFILE.BIN`, and clamps `XSIZE`
    to the real size (`handle_rx` states 6/7 → `dlsize`) so the saved file has its
    **exact length** (no XMODEM 128-byte padding) — `loci_save` writes a partial
    final block accordingly. Server and terminal versions must match.
  - **User-editable name (S1).** Before saving, the terminal prompts
    `NOM (RET=DEFAUT)` (`edit_dlname`): RETURN keeps the proposed name, otherwise
    the typed `NAME.EXT` is parsed (`user_to_sedoric`) into the 12-byte Sedoric
    format and used for the Sedoric/LOCI save.
  - **Raw modem required.** The terminal issues `ATNET0` at init so a telnet WiFi
    modem (e.g. picowifi) does not mangle the binary stream (`0xFF`/CR). See
    `docs/hardware-connection.md` §6.
- **Upload: done.** `xmodem_send` (CRC-16) sends `XSIZE` bytes of the `$4000` buffer.
  The server (`upload` applet) emits **`1F FD`** (`oascii.SendCmd`); `term.s` then switches
  to `xmodem_send`. Validated in emulator (`docs/img/xmodem-upload.png`, 256 bytes).
- **Remaining to do**:
  - **Storage targets**: the `$4000` buffer is saved to **Sedoric** (Microdisc)
    under the real name when Sedoric is resident, otherwise it falls back to the
    **LOCI SD card** (`client/loci.s`, MIA `OPEN`/`WRITE_XSTACK`/`CLOSE` at `$03A0`;
    LOCI detected via the signature opcodes at `$03B3/$03B5/$03B7`). Dispatch is in
    `save_received`: `sed_save` returns `A=1`/`A=0`, and on `A=0` `loci_save` runs.
    User-editable name at reception is **done** (`edit_dlname`). The **cassette**
    (`.TAP`) target is **deferred** (see spike below).
  - **Binary telnet**: handled — the terminal forces the modem to raw mode (`ATNET0`).

### Cassette (.TAP) save — spike S3 (deferred, 27/06/2026)
- **Feasible but not worth it now.** `oric1-emu` already **captures a ROM CSAVE to
  a host `.tap`** (no flag; it patches the Atmos tape routines), so a cassette save
  would be emulator-validatable. The relevant Atmos ROM hooks: `WriteFileHeader
  $E607`, `WriteLeader $E75A`, `PutByte $E65E`, `csave_end $E93C`; staging buffers
  filename `$027F`, header `$02A8..$02B0` (reversed on-tape order).
- **Why deferred.** Implementing the ML CSAVE recipe (header staging layout +
  entry point) is comparable in effort to the Sedoric save reverse-engineering
  (a multi-day job), for **low value**: Sedoric (Microdisc) and the LOCI SD card
  already cover persistent storage; tape-of-downloads is a niche fallback (slow,
  needs a physical recorder, manual tape swaps). A future story can pick it up
  cheaply from the hooks above.

See also: `docs/agile/backlog.md` (G1), `docs/hardware-connection.md`.
