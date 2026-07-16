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
  `xmodem-upload.png`). User-editable name at reception **done (S1, 27/06)**
  (`edit_dlname`/`user_to_sedoric`, runtime-tested). Cassette (.TAP) **storage**:
  spike S3 done → **deferred** (feasible via emulator CSAVE capture, but ≈ Sedoric
  R&E effort for low value; ROM hooks noted in `docs/transfer.md`).*
  - **Path C (LOCI SD) — ✅ implemented (26/06)**: `client/loci.s` writes the
    `$4000` buffer to the LOCI SD card via the MIA API (`OPEN`/`WRITE_XSTACK`/
    `CLOSE` at `$03A0`), used as a fallback by `save_received` when Sedoric is
    absent. LOCI presence detected via signature opcodes `$03B3/$03B5/$03B7`.
    **Validated at runtime** in `oric1-emu --loci-flash` (standalone harness): a
    256-byte file is written to the SD sandbox byte-identical to the source,
    `loci_save` returns `A=1`. MIA opcodes/flags/XSTACK convention audited against
    the emulator source. Path push made NUL-terminated for real-LOCI conformance.
  - **Path B (Sedoric) — ✅ ML save VALIDATED on V3.0 (24/06)**: floppy write
    proven (`--disk-writeback` flag); ML recipe validated end-to-end
    (`JSR $04F2` overlay → variables → `JSR $DE9C` XSAVEB → `JSR $04F2`), a
    file written/persisted to the `.dsk`. `client/sedoric.s` finalised. **Remaining**
    is the `term.s` integration (trigger after a download) + deploying the
    terminal under resident Sedoric. See `docs/sedoric-api.md`.

## Epic H — DataWindow (typed data grids, telenet parity)

- [x] **H1** (8) As a user, I want to **browse and edit typed data** (a paginated grid
  with CRUD), so the BBS can host directories/registries, not just static pages.
  *(Increment 1, 27/06: SQLite engine `server/internal/datawindow` (port of telenet,
  `modernc.org/sqlite` pure-Go), model + validation in `internal/content`, `datawindow`
  grid applet on the `oascii.Screen` diff buffer (inverse selection, `+/- S/R V N/E/D
  F/C Q`), `-data` flag. Engine/content/integration tests + driver smoke. Docs
  `docs/datawindow.md`, ADR-0004, demo `docs/examples/datawindow-demo.json`.)*
- [~] **H2** (5) As an editor, I want to **manage sources and data from the studio**
  (Forge), and interactive column sort / prefix search / REST sources (telenet parity).
  *(Interactive column sort (key `T`) + REST API sources (`type_source:"api"`,
  read-only) done 27/06. Remaining: studio editor, prefix search.)*

## Epic I — Code quality & hardening (post-analysis, 16/07/2026)

> Issued from the full source analysis of 16/07/2026 (server / studio / shared
> `internal/` / Oric firmware). Priority order: real bugs → security → robustness →
> hygiene. Each story cites the offending `file:line`.

### Real bugs
- [x] **I1** (2) As a connected user, I want my **pseudo to appear in "who's online" and chat
  even when I log in via a `form` page**, so presence is consistent across auth paths.
  *(Done 16/07: `form.go` `applyFormAction` now calls `setPresenceHandle` in the login and
  register cases; regression test `TestFormLoginSetsPresence` — verified failing without the fix.)*
- [x] **I2** (2) As a user downloading a file, I want the **real size to be sent whole**
  (not truncated to 16 bits), so oversized files never corrupt the terminal save.
  *(Done 16/07 — server-side guard: `downloadApplet` rejects a file larger than
  `maxDownloadSize` (0xFFFF, the header's 16-bit size limit) with an explicit message;
  test `TestDownloadTooLarge`. Widening the header to 3 bytes needs a matching `client/xmodem.s`
  change → tracked as **I2b**.)*
- [ ] **I2b** (3) As a user, I want to download files **larger than 64 KB** by widening the
  download header size field, with a matching terminal firmware change. *(Requires emulator
  validation per DoD; deferred from I2.)*

### Security
- [ ] **I3** (3) As an admin deploying content, I want the **remote backup/reload commands
  to be injection-safe**, so a crafted profile value cannot run arbitrary shell on the target.
  *(`studio/internal/deploy/deploy.go:269,304` interpolate `ContentPath`/`Service` into
  shell strings run via `ssh` — `test -f %s && cp …`, `systemctl reload %s` — without
  escaping; `parseProfile` accepts any value. Quote/validate, or use arg-separated form.)*
- [ ] **I4** (3) As the server, I want **auth attempts rate-limited over time / per IP**, so
  re-navigating to the login screen cannot be used for brute-force.
  *(`login.go` caps 3 tries **per applet pass** only; PBKDF2 100k slows but does not block.)*
- [ ] **I5** (2) As an admin, I want an **admin role gating DataWindow CRUD**, so not every
  logged-in account can write editable grids.
  *(`server/internal/bbs/datawindow.go:82` — `editable = dw.Editable && LoggedIn()`; no role.)*

### Robustness & tests
- [ ] **I6** (3) As a dev, I want **XMODEM hardened and fully tested**, so transfers fail cleanly.
  *(`internal/xmodem`: the **checksum branch is never exercised** by tests — round-trip
  always starts the receiver in CRC; error handling conflates timeouts with real I/O
  errors (loops instead of surfacing); no `CAN` emitted on abort; `Receive` has no size
  bound. Add checksum-path tests, distinguish error kinds, emit CAN, bound the buffer.)*
- [ ] **I7** (2) As a content author, I want **`content.Validate` to catch more errors up
  front**, so a bad site fails at load, not at runtime.
  *(`internal/content`: referenced applet existence never checked (`content.go:158-177`);
  column `Pattern` regex never compiled at validation; DataWindow default width hard-coded
  `8` in `datawindow.go:153` — dedupe with the renderer to avoid drift.)*
- [ ] **I8** (2) As a Forge user, I want the **HTTP layer to enforce methods and report real
  status codes**, so client errors are distinguishable.
  *(`studio/cmd/forge/main.go:181` etc. don't require POST and always return 200 with
  `{ok:false}`; `readBody` errors are ignored throughout.)*

### Hygiene (low effort)
- [ ] **I9** (1) As a dev, I want the **phantom `"type"` field removed** from `content` test
  JSON (and documented), since `Page` has no such field and Go silently ignores it.
  *(`content_test.go:18,22,55`, `store_test.go:18,33,54` vs `content.go:44-55`.)*
- [ ] **I10** (1) As a dev, I want the **attribute re-emission invariant deduplicated** between
  `oascii.Builder.Newline` (sticky) and `render.reemitState`, and centering made rune-safe.
  *(`oascii.go:206-216` vs `render.go:64-79`; `render.center/rule` use byte `len`.)*
- [ ] **I11** (1) As a dev, I want the **firmware minor cleanups**: fix the misleading
  `term.s:274` comment (`" -"` — the byte actually sent is `:`/`$3A`), drop the dead
  `hires.s:747-748` (`lda hy1`/`sta hy1`), and note the tight XMODEM buffer margin vs the
  `$B800` charset in `xmodem.s`.

## Definition of Done (DoD)
- Versioned code, `CHANGELOG.md` and `ROADMAP.md` updated.
- Tests passing for the delivered feature.
- Documentation up to date (`docs/`).
- Validated in `Oric1/oric1-emu` (Phosphoric) when applicable.
