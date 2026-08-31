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

- [x] **D1** (5) As a user, I want to **read and post messages** (forum).
  *(Done 17/07/2026 — applet `forum` + package `server/internal/forum` (atomic JSON store,
  thread→posts, persistent ID, bounded, ASCII-sanitised). Paginated list (`S/P`, `1-8`, `N`)
  and thread view (`R` reply). Server flag `-forum`; « Forum » in the Community menu. Reuses
  the wall's persisted-write pattern; `oascii.SanitizeText` shared. Store tests + TCP
  integration `TestForumCreateReadReply`; verified in the real server. Cf. ROADMAP Sprint 7 #1.)*
- [x] **D2** (2) As a user, I want to see **news / announcements**.
  *(Done 17/07/2026 — applet `news` + package `server/internal/news` (atomic JSON store,
  dated, bounded, ASCII-sanitised). Read for all, admin-only publish. Server flag `-news`;
  « Actualites » in the Services menu. Store tests + TCP integration. Cf. ROADMAP Sprint 8 S2.)*
- [x] **D3** (3) As a user, I want to play a **mini-game** (e.g. Connect Four).
  *(Done 17/07/2026 — applet `connect4` + package `server/internal/connect4` (pure logic,
  7×6 board, deterministic AI: win→block→center, win/draw detection). Services menu
  « Puissance 4 ». Logic unit tests + TCP integration; verified in the real server.)*

## Epic E — Real hardware & deployment (Sprint 4–5)

- [x] **E1** (3) As a user, I want a **connection doc** for WiFiModem + LOCI.
  *(`docs/hardware-connection.md`: ACIA `$031C`/LOCI `$0380`, AT, 9600 8N1, recipe T1–T9)*
- [~] **E2** (5) As a user, I want to connect from a **real Oric**.
  *(terminal validated in the emulator; hardware test awaiting a physical Oric.
  17/07: the terminal firmware, in `oric1-emu`, dials the **PRODUCTION** server via
  the emulated modem (`scripts/test-emulateur-prod.sh` → `--serial modem:pavi.3617.fr:6502`)
  and reaches the live catalogue — `docs/img/e2-prod-catalogue.png` (LOGICIELS grid,
  2604 entries, X=DL). Only the physical Oric + WiFi modem + LOCI/ACIA remains.)*
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
- [x] **F3** (3) As an editor, I want to **create/manage several sites** and their backups
  from the UI. *(done 31/08/2026: `Store.Create`/`Backup`/`Backups`/`Restore`, routes
  `POST /api/site/create`, `POST /api/backup`, `GET /api/backups`, `POST /api/restore`,
  boutons « Nouveau site » / « Sauvegarder » / « Restaurer… ». Sauvegardes horodatées dans
  `<content-dir>/backups/`, restauration avec sauvegarde de sécurité préalable, noms durcis
  contre la traversée. Tests `TestCreateBackupRestore`/`TestBackupInvalidName`.)*

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
- [x] **H2** (5) As an editor, I want to **manage sources and data from the studio**
  (Forge), and interactive column sort / prefix search / REST sources (telenet parity).
  *(Interactive column sort (key `T`) + REST API sources (`type_source:"api"`,
  read-only) done 27/06. Studio editor done (Sprint 9). Prefix search done 17/07/2026
  (`Engine.Lister` `prefixe` param, applet key `P`, studio `/api/grid`, legend `F/P/C`;
  test `TestListerPrefixe`). **Input masks + API auth/headers done 31/08/2026**:
  `ColonneDef.masque` (`content.ValiderMasque`, `Engine.Valider`, applet hint, studio
  column) ; `APIConfig.Headers` with `${VAR}` env-expansion (secrets out of `site.json`),
  studio headers editor ; tests `ValiderMasque`/`APISourceHeaders`/`ValidateAPIHeaders`.)*

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
- [ ] **I2b** (8) As a user, I want to download files **larger than 64 KB**. **Requalified
  31/08/2026** (analysis in `docs/transfer.md` §"Download size limit"): the 16-bit header is
  **not** the binding limit — the terminal buffers the whole file in **RAM at `$4000`**, which
  caps at **~32 KB** on a 48 KB Oric (`$4000..$BB80`), well below 64 KB. Widening the header
  alone unlocks nothing and risks the hardware-validated ≤32 KB path. The real enabler is
  **streaming reception straight to disk** (write each XMODEM block to Sedoric/LOCI instead of
  buffering `$4000`), a firmware change requiring emulator validation. Re-pointed from
  "widen header" (3) to "streaming-to-disk receiver" (8). *(Blocked on emulator/hardware.)*

### Security
- [x] **I3** (3) As an admin deploying content, I want the **remote backup/reload commands
  to be injection-safe**, so a crafted profile value cannot run arbitrary shell on the target.
  *(Done 16/07 — `validateProfileFields` (safe charset) rejects unsafe `HOST/USER/PORT/
  CONTENT_PATH/SERVICE` at both `Deploy` and `SaveProfile`; test `TestDeployRejectsShellInjection`.)*
- [x] **I4** (3) As the server, I want **auth attempts rate-limited over time / per IP**, so
  re-navigating to the login screen cannot be used for brute-force.
  *(Done 16/07 — new `server/internal/throttle` sliding-window limiter; 5 fails/IP/5 min,
  wired to `login` + `form` login; `TestLoginRateLimited` + unit tests.)*
- [x] **I5** (2) As an admin, I want an **admin role gating DataWindow CRUD**, so not every
  logged-in account can write editable grids.
  *(Done 16/07 — `User.Admin` (first account = sysop, flag editable in users.json);
  `editable = dw.Editable && IsAdmin()`; legend hides `N/E/D` from non-admins. Tests
  `TestFirstAccountIsAdmin`, `TestAdminFlagPersists`, `TestDataWindowGuestCannotCreate`.)*

### Robustness & tests
- [x] **I6** (3) As a dev, I want **XMODEM hardened and fully tested**, so transfers fail cleanly.
  *(Done 16/07 — timeout vs real I/O error distinguished (surfaced immediately); `CAN` emitted
  on abort; `Receive` bounded by `maxReceiveBytes` (`ErrTooLarge`). Checksum branch now tested:
  `TestSendChecksumMode`, `TestReadBlockChecksum`; plus `TestSendSurfacesIOError`,
  `TestReceiveRejectsOversize`.)*
- [x] **I7** (2) As a content author, I want **validation to catch more errors up front**,
  so a bad site fails at load, not at runtime.
  *(Done 16/07 — column `Pattern` regex compiled in `content` validation; `bbs.ValidateSiteApplets`
  detects an unknown referenced applet at startup (called in `main`); default column width
  deduped via `content.DefaultColWidth`. Tests `TestValidateColumnPattern`, `TestValidateSiteApplets`.)*
- [x] **I8** (2) As a Forge user, I want the **HTTP layer to enforce methods and report real
  status codes**, so client errors are distinguishable.
  *(Done 16/07 — mutating endpoints require POST (405 + `Allow`); `readBody` errors → 400;
  invalid save → 400 (detail in body). Tests `TestMutatingEndpointsRequirePOST`,
  `TestHandleSaveInvalidReturns400`.)*

### Hygiene (low effort)
- [x] **I9** (1) As a dev, I want the **phantom `"type"` field clarified**, since `Page` has no
  such field and Go silently ignores it.
  *(Done 16/07 — documented in `content.Page` + `docs/content.md` as a tolerated, preserved,
  never-interpreted descriptive hint (page kind inferred from populated fields). Not removed:
  it is used pervasively and preserved by the studio round-trip.)*
- [x] **I10** (1) As a dev, I want the **attribute re-emission invariant deduplicated** between
  `oascii.Builder.Newline` (sticky) and `render.reemitState`, and centering made rune-safe.
  *(Done 16/07 — shared `oascii.Builder.ReemitAttrs`; `render.center` counts runes. Test
  `TestReemitAttrs`.)*
- [x] **I11** (1) As a dev, I want the **firmware minor cleanups**: fix the misleading
  `term.s` comment (byte sent is `:`/`$3A`) and drop the dead `hires.s` block (`hy0` reassigned
  before use + `lda hy1`/`sta hy1` no-op). *(Done 16/07 — reassembled via `make client`.)*
  - Note: the tight XMODEM `$4000` buffer vs `$B800` charset margin is a **latent** concern,
    left as a documented observation (no behaviour change) — see analysis of 16/07.

## Epic J — Download catalogue (software / magazines / books)

> Browsable catalogue backed by DataWindow, sourced from the **OricProgramsLib** library.
> Honest constraint: XMODEM download to an Oric is bounded (~30 KB terminal buffer, 64 KB
> guard) — software (small `.tap`) is downloadable; magazines/books (PDF) are listed and
> browsable but not downloadable to an Oric.

- [x] **J1** (5) As a user, I want to **browse a catalogue** of software, magazines and books
  (title / author / year, with a detail screen) and **download** a software title.
  *(Done 16/07 — DataWindow `fichier_colonne` + grid key `X` → XMODEM (`sendFileDownload`);
  detail via `V`, filter `F`, sort `T`. Generator `scripts/gen-catalogue.py` from OricProgramsLib;
  demo `docs/examples/catalogue-demo.json`. Tests `TestDataWindowDownloadFromRow`,
  `TestValidateFichierColonne`. Verified in the real server.)*
- [x] **J2** (3) As an admin, I want to **populate `-files` with the downloadable files** and
  generate the **full** catalogue, so the catalogue can go live.
  *(Done 16/07 — generator is now size-aware: only files that actually fit (`--max-file-size`,
  default 30720 = Oric terminal buffer) are marked downloadable, with a `taille` column;
  `--copy-files <dir>` copies them into `-files` under a safe short (8.3) name. Full library:
  2604 software / **1911 downloadable**, 710 magazines, 192 books (~1.2 MB catalogue, 1874 files).
  Verified in the real server (filter + `X` → real `CENTI.TAP` download).)*
  - Remaining ops (per `make deploy`): choose to serve the catalogue standalone
    (`-content catalogue.json -files <dir>`) or merge it into `content/site.json`, then deploy.
- [x] **J3** (3) As a user, I want a **fixed per-page category filter** (a page shows only its
  category without pressing `F`).
  *(Done 16/07 — `filtre_fixe` (colonne = valeur) on the DataWindow descriptor, combinable
  (AND) with the `F` filter; applied in SQL (SQLite) and in-memory (API). The catalogue is now
  a single `catalogue` source presented as 3 filtered views. Tests `TestListerFiltreFixe`,
  `TestValidateFiltreFixe`. Verified in the real server.)*
- [x] **J4** (2) As a user, I want **richer detail** on the fiche `V`, using the extra
  OricProgramsLib metadata.
  *(Done 16/07 — the fiche now **wraps** long values (author, description) instead of
  truncating at 22 cols (`wrapValeur`); generator adds genre / publisher / language / players /
  screenshot-reference columns. Per-item download count is not in the library data → not added
  (not invented). Test `TestWrapValeur`. Verified in the real server.)*

## Epic K — Portable PC terminal (oterm)

> A client to use/test the BBS from a modern PC (Linux/Windows/macOS) without an Oric or
> emulator, complementing the studio's ULA preview (browser) and the real firmware (`client/`).

- [x] **K1** (5) As a user, I want a **portable terminal** that connects to `bbsd` and renders
  the OASCII stream (TEXT 40×28, Teletext colours, inverse) in any ANSI terminal.
  *(Done 31/08/2026 — new sub-project `pcterm/`: `internal/ula` decodes the OASCII stream into a
  40×28 grid + cursor faithfully to `client/term.s` (chars, CR/LF+scroll, backspace, 40-col clamp,
  `plot 1F col row`) and renders ANSI with serial attribute semantics mirroring the studio ULA sim;
  `cmd/oterm` does TCP + raw keyboard (char-by-char, ADR-0002) via `golang.org/x/term`, quit on
  `Ctrl-]`. HIRES (`1F FC`) and XMODEM (`1F FE/FD`) are not rendered (status message; HIRES stream
  swallowed to `1F FB`). Cross-compiles linux/windows/darwin. Tests `pcterm/internal/ula/ula_test.go`
  (text/scroll/clamp/backspace/plot/attributes/inverse/hires/xmodem); end-to-end smoke test against a
  live `bbsd` (welcome + main menu decoded). `make oterm`. Docs `pcterm/README.md`.)*
  - Not done at K1 (later): HIRES rendering, BBS charset, video-mode attributes → done in K2.
- [x] **K2** (5) As a user, I want oterm to render **HIRES graphics**, the **BBS alternate
  charset** and the **full serial attributes** (TEXT/LORES/HIRES), not just plain text.
  *(Done 01/09/2026 — new `pcterm/internal/hires`: Go rasteriser of the HIRES opcode stream
  (ported from `client/hires.s`: VRAM 200×40, pixel = bit(5−x%6), clear 0x40, per-line serial
  ink attribute, colour mode sacrifices x<6), rendered 240×200 as Unicode half-blocks `▀`
  (240×100, per-pixel colour). Standard font embedded (`font.go`, from `charset.js`) for
  `HiChar`. BBS alt charset (text attr bit0) mapped to Unicode runes (`ula/font.go`, table from
  `tools/genfont`). Video-mode group 0x18–0x1F handled: inverse ON/OFF (29/28) serial state.
  `1F FC`→HIRES, `1F FB`→text. Tests (rasteriser primitives + ula alt-charset/inverse/hires) +
  e2e via `render.Hires` + visual smoke test on `docs/examples/hires-demo.json` (24000
  half-blocks, 14 colour pairs). Docs `pcterm/README.md`.)*
  - Not done at K2 (later): XMODEM transfer → done in K3.
- [x] **K3** (3) As a user, I want oterm to **download files** from the BBS (catalogue / files
  menu) over XMODEM. *(Done 01/09/2026 — `pcterm/internal/xfer` reuses shared `internal/xmodem`:
  on `1F FE`, oterm takes over the connection, reads the download header (blocks + 12-byte
  Sedoric name + real size), receives the file, truncates to the exact size and saves it under
  its real name in `-dl` (default `.`). Server-initiated upload (`1F FD`) is cleanly cancelled
  (CAN). `ula.WriteScan` stops on a transfer trigger and hands the remaining bytes back; oterm
  prefixes them to the connection (`prefixConn`). Tests: Sedoric-name parsing + net.Pipe loopback
  download; `TestWriteScanDetecteTransfert`. Real smoke test against `bbsd -files` (downloaded
  file byte-identical, 90 o).)*
  - Not done (documented limit): pixel-accurate double-height (a text terminal has no half-height);
    client-initiated upload (needs local file selection UX).

## Definition of Done (DoD)
- Versioned code, `CHANGELOG.md` and `ROADMAP.md` updated.
- Tests passing for the delivered feature.
- Documentation up to date (`docs/`).
- Validated in `Oric1/oric1-emu` (Phosphoric) when applicable.
