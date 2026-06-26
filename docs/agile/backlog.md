# Product backlog — BBS Oric

> Prioritised. `[ ]` = to do, `[~]` = in progress, `[x]` = done. Indicative estimate in points.

## Epic A — Foundation & connection (Sprint 0–1)

- [x] **A1** (1) As a team, I want a versioned and documented repository to track the work.
- [x] **A2** (1) As a dev, I want to **confirm the server language** (→ **Go 1.26**).
- [x] **A3** (3) As a user, I want to connect over telnet and see a welcome screen
  ("hello world"), in order to validate the network chain end-to-end. *(tested via `nc`)*
- [x] **A4** (2) As a dev, I want to test the connection **in an emulator** without hardware.
  *(`oric-client/term.s` terminal + `scripts/test-emulateur.sh`; coloured banner validated on screen)*

## Epic B — OASCII rendering (Sprint 1)

- [x] **B1** (5) As an Oric user, I want **colour** screens (ink/paper) correctly
  rendered despite the serial attributes. *(coloured banner validated ON SCREEN in oric1-emu)*
- [x] **B2** (3) As a dev, I want a **screen API** (`ink/paper/blink/text/newline`) that hides the
  attribute codes. *(OASCII Builder; `cls`/cursor handled on the Oric terminal side by VRAM writing)*
- [x] **B3** (2) As a dev, I want a **verified Oric attribute table** on the emulator.
  *(extracted from `oric1-emu` src/video/video.c; 7 unit tests)*

## Epic C — BBS engine (Sprint 2)

- [x] **C1** (3) As a user, I want to **navigate menus** and go back.
  *(main menu + 3 screens, back via RETURN; Go tests + emulator screen validation)*
- [x] **C2** (3) As a server, I want to handle **several simultaneous connections** without blocking.
  *(1 goroutine/connection, `server` layer)*
- [x] **C3** (2) As a user, I want to be **disconnected cleanly** after inactivity.
  *(idle timeout, `server` layer)*
- [x] **C5** (3) As an Oric user, I want to **type on the keyboard** to navigate (terminal TX).
  *(full matrix scan + local echo + CR; navigation validated on screen via `--type-keys`)*
- [~] **C4** (3) As a user, I want to **identify myself** and retrieve my profile.
  *(ADR-0001/0002; increments 1–3 delivered: hashed store `internal/user`, single-key
  input `ReadKey`, applet engine (`applet` page type), login/register/guest applets,
  auth gate at CONNECT, `-users` wiring + deployment. Validated end-to-end (`nc`). Remaining
  on the client side: `term.s` in character mode + password no-echo.)*

## Epic D — Content (Sprint 3)

- [ ] **D1** (5) As a user, I want to **read and post messages** (forum).
- [ ] **D2** (2) As a user, I want to see **news / announcements**.
- [ ] **D3** (3) As a user, I want to play a **mini-game** (e.g. Connect Four).

## Epic E — Real hardware & deployment (Sprint 4–5)

- [x] **E1** (3) As a user, I want a **connection doc** for WiFiModem + LOCI.
  *(`docs/hardware-connection.md`: ACIA `$031C`/LOCI `$0380`, AT, 9600 8N1, recipe T1–T9)*
- [~] **E2** (5) As a user, I want to connect from a **real Oric**.
  *(terminal validated in the emulator; hardware test awaiting a physical Oric)*
- [x] **E3** (3) As an admin, I want to **deploy** the server (Docker) and **supervise** it.
  *(prod systemd + Docker image ~18 MB + `/healthz`,`/metrics` + probe/timer)*
- [x] **E4** (3) As an admin, I want to **back up and restore the state** (accounts,
  files, content) so as not to lose anything in case of incident.
  *(`scripts/backup.sh`/`restore.sh`, daily timer + rotation, hot, e2e test
  `scripts/test-backup.sh`, doc `docs/backup.md`; deployment via `vps-deploy.sh`)*

## Epic F — "Forge" studio (content tooling)

- [x] **F0** (3) As a team, I want a repository in **3 sub-projects** (server/client/studio)
  with reusable shared packages. *(restructuring, ADR-0003)*
- [x] **F1** (5) As an editor, I want to **compose the site.json** (menu/page/applet) with
  **colour preview** and validation. *(forge web Go, internal/content reused)*
- [x] **F2** (5) As an admin, I want to **deploy the content** to **dev/int/prod** via
  **profiles** (validate→backup→overwrite→reload, dry-run). *(validated end-to-end)*
- [ ] **F3** (3) As an editor, I want to **create/manage several sites** and their backups
  from the UI.

## Epic G — File transfer (study, not planned)

- [~] **G1** (8) As a user, I want to **download/upload** files via the BBS.
  *Server side **done** (`internal/xmodem`, `server/internal/files`, applets
  `download`/`upload`, `Session.Raw()`, `-files`/`-max-upload` flags, studio, doc
  `docs/transfer.md`). **Oric download AND upload done**: receiver (checksum) +
  sender (CRC-16) XMODEM 6502 (`client/xmodem.s`), triggered by `1F FE`/`1F FD`,
  RAM buffer `$4000` — validated in the emulator (`docs/img/xmodem-download.png`,
  `xmodem-upload.png`). **Remaining**: SD card (LOCI)/Microdisc/cassette **storage**
  (RAM buffer for now).*
  - **Path B (Sedoric) — ✅ ML save VALIDATED on V3.0 (24/06)**: floppy write
    proven (`--disk-writeback` flag); ML recipe validated end-to-end
    (`JSR $04F2` overlay → variables → `JSR $DE9C` XSAVEB → `JSR $04F2`), a
    file written/persisted to the `.dsk`. `client/sedoric.s` finalised. **Remaining**
    is the `term.s` integration (trigger after a download) + deploying the
    terminal under resident Sedoric. See `docs/sedoric-api.md`.

## Definition of Done (DoD)
- Versioned code, `CHANGELOG.md` and `ROADMAP.md` updated.
- Tests passing for the delivered feature.
- Documentation up to date (`docs/`).
- Validated in `Oric1/oric1-emu` (Phosphoric) when applicable.
