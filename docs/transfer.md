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
    Still to add: **user-editable name** at reception and the **cassette** (`.TAP`)
    target, selected by available hardware.
  - **Binary telnet**: handled — the terminal forces the modem to raw mode (`ATNET0`).

See also: `docs/agile/backlog.md` (G1), `docs/hardware-connection.md`.
