# ROADMAP — BBS Oric

**Agile** approach, incremental deliveries. Each sprint produces a testable increment.

> **Cross-cutting constraint: public Internet server.** The BBS is exposed on the Internet (listens
> on `0.0.0.0`, reachable by any Oric via WiFiModem). Security, exposure and hosting are concerns of
> **every** sprint, not just Sprint 5. See `docs/architecture.md` §5.

## Sprint 0 — Scoping & foundation ⏳ (in progress)
- [x] State of the art of retro BBS servers (`docs/state-of-the-art.md`)
- [x] Target scoping: Oric-1/Atmos + LOCI + WiFiModem; test emulator = `Oric1/oric1-emu` (Phosphoric)
- [x] Git repository initialisation, agile documentation, CHANGELOG, ROADMAP
- [x] **DECISION**: server language = **Go** (1.26)
- [x] **DECISION**: hosting = **cloud VPS (fixed IP)**; public port = **6502** (a nod to the CPU)
- [x] "Hello world" telnet server listening on `0.0.0.0:6502`, tested via `nc` ✅
- [x] Minimal Internet exposure: global and per-IP connection limit, idle timeout, connection logs
- [x] Emulator test pipeline confirmed (oric1-emu/Phosphoric `--serial tcp:`) — see `docs/emulator-testing.md`

## Sprint 1 — Oric terminal layer ("OASCII") 🎯 heart of the project — ⏳ in progress
- [x] Encoding of the **Teletext serial attributes**: ink (8), paper (8), blink, double height, alt charset
  — table extracted from the ULA decoder of `oric1-emu` (`src/video/video.c`), unit tests green
- [x] `internal/oascii`: `Builder` (`Ink/Paper/Blink/DoubleHeight/AltCharset/Text/Newline`), `Sticky` mode
- [x] Coloured welcome banner (handler) — byte stream verified by hexdump
- [x] Documented spec: `docs/oascii.md`
- [x] **Oric terminal** (`oric-client/term.s`, 6502/xa): ACIA `$031C` → direct VRAM write `$BB80`
  (CR/LF/scroll, 40-col clamp), autorun `.tap` build via `bin2tap`
- [x] **Visual validation in `oric1-emu`**: coloured banner rendered correctly (yellow/cyan/green/white)
  — capture `docs/img/sprint1-banner.png`, automated test `scripts/test-emulateur.sh`
- [ ] Cursor positioning / direct `cls` (optional — VRAM writing already handles rendering; to be defined if needed)

## Sprint 2 — BBS engine — ⏳ in progress
- [x] Multi-client session loop (1 connection = 1 goroutine) — `server` layer
- [x] Menu / navigation system (`internal/bbs/menu.go`: main menu + Information / About / Guestbook
  screens, coloured OASCII rendering) — validated on screen (`docs/img/sprint2-menu.png`)
- [x] Idle timeout, clean disconnect — `server` layer
- [x] **Keyboard transmission (TX) on the Oric terminal side** — full AY-via-VIA matrix scan
  (`oric-client/term.s`), local echo, line termination on CR. **Interactive navigation
  validated on screen** (`docs/img/sprint2-keyboard-nav.png`, test via `--type-keys`).
- [~] Login / user profiles (persistence) — **increments 1–3 done** (ADR-0001/0002):
  - `internal/user`: model + atomic hashed store (PBKDF2 stdlib), `-race` tests.
  - **Single-key** input (menus) + **line/RETURN** (text fields): `server.ReadKey`.
  - **Applet engine**: `applet` page type (JSON) → Go applet registered by name.
  - **`login`/`register`/`guest`** applets, auth gate at CONNECT, personalised welcome.
  - `cmd/bbsd -users` wiring + deployment (`StateDirectory`). Validated end-to-end (`nc`).
  - **Oric terminal**: verified — `term.s` **already** emits each keystroke immediately (character
    mode), no change required (cf. ADR-0002). The emulator confirms the
    keyboard→dial→CONNECT→RX pipeline.
  - **Remaining**: emulator capture of the new login screen (limitation of the emulated modem backend
    that dials the real hosts — plan a local picowifi entry / hardware test);
    password no-echo (optional).

## Sprint 9 — DataWindow (typed data grids) — ✅ increment 1 (27/06/2026)
> Notion ported from the telenet server: typed data sources + paginated CRUD grid.
- [x] **SQLite engine** (`server/internal/datawindow`, `modernc.org/sqlite` pure-Go):
  CRUD, validation, pagination, sort, LIKE filter, seed + auto-migration. Ported from
  telenet `datawindow.go`; SQL-injection guards copied verbatim.
- [x] **Model in `internal/content`** (`SourceDonnees`/`ColonneDef` + page `DataWindow`)
  with `Site.Validate()` checks (names, columns, 40-col budget).
- [x] **Grid applet** on the `oascii.Screen` diff buffer (inverse selection, `+/- S/R
  V N/E/D F/C Q`); `-data` flag; threaded through `AppContext`/`SessionState`.
- [x] **Tested** (engine + content + TCP-driver integration) + driver smoke +
  **oric1-emu visual** (`docs/img/datawindow-grid.png` : 40-col, inverse selection).
  Docs `docs/datawindow.md`, ADR-0004, demo `docs/examples/datawindow-demo.json`.
- [x] **Interactive column sort** (key `T`, cycles col/dir, footer label) — done 27/06.
- [x] **REST API sources** (`type_source:"api"`, read-only, server-side filter/sort/
  pagination, TTL cache) — done 27/06 (`server/internal/datawindow/api.go`).
- [x] **Studio editor for sources/data** (27/06): « Données » tab (sources: SQLite/API,
  typed columns, seed) + grid descriptor editor (columns/order/widths/colours/editable,
  live /40 budget) in « Édition ». Round-trip test extended. Docs updated.
- [ ] **Increment 2** (later): prefix search, input masks, API auth/headers.

## Sprint 10 — HIRES pages (240×200 graphics) — ⏳ slice 1 (27/06/2026)
> Graphics pages over the serial link: **both** a bitmap model (logo/splash) and a
> primitives model (line/box/circle/…). Design: `docs/adr/0005-hires-pages.md`.
- [x] **Server foundation** (27/06): content model (`Hires`/`HiresOp`, page field
  `hires`) + `Site.Validate()` (bitmap 8000 B, bounds 240×200, colours, known ops);
  unified wire stream `render.Hires` over sub-command `1F FC` (opcodes + RLE bitmap
  blit, `internal/oascii/hires.go`); engine wiring (menu/one-key); unit + TCP-driver
  tests. Docs `docs/hires.md`, ADR-0005.
- [x] **Terminal firmware** (`client/hires.s`, 27/06): HIRES interpreter — `1F FC`
  handler, mode switch (attr `0x1E` → `$A000` + 3 text lines), self-contained 6502
  primitives (setpixel, Bresenham x/y-major, box/fillbox, midpoint circle, `char` via
  charset saved to `$9800`), RLE blit decoder. **Validated in `oric1-emu`** — both
  models render (`docs/img/hires-demo-emu.png` primitives + `…-bitmap-emu.png` blit).
- [x] **Studio Forge** (27/06): « Édition » tab HIRES editor — `+ page graphique`,
  primitive table (op + X/Y/R/colour/char, reorder/remove), **image import** → bitmap
  background, and a **live 240×200 preview rasterized in JS** (mirror of `hires.s`).
  Page map labels it `graphique` (`◨ hires`). Store round-trip test. Docs updated.
- [x] **Clean TEXT-return** (27/06): `1F FB` command — server emits it (session flag)
  before a text page following a HIRES one; terminal restores charset (`$9800`→`$B400`),
  TEXT video attribute and clears the screen. Validated in `oric1-emu` + integration test.
- [x] **HIRES ink colour** (27/06): `ink` op now coloured — the terminal places a
  per-line ink attribute at column 0 (authentic Oric attribute clash); monochrome
  unchanged when no `ink` op. Studio preview colourised. Validated in `oric1-emu`.
- [ ] **Later**: HIRES `paper` colour, flow-controlled bitmap transfer (vs raw blit),
  differential HIRES buffer for animation.

## Sprint 11 — Code quality & hardening — ✅ done (16/07/2026), sauf I2b (firmware, différé)
> Decomposition of **Epic I** (`docs/agile/backlog.md`), issued from the full source
> analysis of 16/07/2026. Ordered by value: real bugs first, then security, robustness,
> hygiene. Each task cites the offending `file:line` and its acceptance test. DoD applies
> per task (CHANGELOG + tests green + doc if behaviour changes).

### Slice 1 — Real bugs (I1, I2) — ✅ done (16/07/2026)
- [x] **S11.1 — Presence via `form` applet** (I1): `applyFormAction` (`form.go`) now calls
  `setPresenceHandle(ac.State, u.Handle)` in the login and register cases, as `login.go`
  does. Regression test `TestFormLoginSetsPresence` — **verified failing without the fix**
  (handle stayed `connexion...`).
- [x] **S11.2 — Guard oversized download** (I2): `downloadApplet` refuses a file larger than
  `maxDownloadSize` (0xFFFF — the 16-bit size header's limit) with an explicit message,
  before any transfer, instead of silently truncating. Test `TestDownloadTooLarge`.
- [ ] **S11.2b — Widen download header for files > 64 KB** (I2b): 3-byte size field + matching
  `client/xmodem.s` change; needs emulator validation (DoD). Deferred.

### Slice 2 — Security (I3, I4, I5) — ✅ done (16/07/2026)
- [x] **S11.3 — Injection-safe remote deploy** (I3): `validateProfileFields` restricts
  `HOST/USER/PORT/CONTENT_PATH/SERVICE` to a safe charset (`[A-Za-z0-9._@/-]`), enforced in
  both `Deploy` (before execution) and `SaveProfile` (before persisting). A profile with a
  shell metacharacter is refused. Test `TestDeployRejectsShellInjection`.
- [x] **S11.4 — Auth rate-limiting over time** (I4): new `server/internal/throttle` limiter
  (sliding window per key, concurrency-safe, injectable clock); wired on the client IP —
  5 fails/IP/5 min, in addition to the per-pass cap. Applied to `login` and the `form` login
  action. Unit tests + integration `TestLoginRateLimited`.
- [x] **S11.5 — Admin role for DataWindow CRUD** (I5): `User.Admin` (first registered account
  = sysop; flag editable in users.json). `editable = dw.Editable && IsAdmin()`; read stays
  open to all, write requires admin, and the grid legend hides `N/E/D` from non-admins.
  Tests `TestFirstAccountIsAdmin`, `TestAdminFlagPersists`, `TestDataWindowGuestCannotCreate`.

### Slice 3 — Robustness & tests (I6, I7, I8) — ✅ done (16/07/2026)
- [x] **S11.6 — XMODEM hardened + checksum tests** (I6): distinguish timeout from real I/O
  error (surface immediately, no spin to `ErrTooManyNAK`); emit `CAN` on abort; bound
  `Receive` (`maxReceiveBytes`, `ErrTooLarge`). Checksum branch now exercised
  (`TestSendChecksumMode`, `TestReadBlockChecksum`) + `TestSendSurfacesIOError`,
  `TestReceiveRejectsOversize`.
- [x] **S11.7 — Stronger validation** (I7): column `Pattern` regexes compiled in `content`
  validation; `bbs.ValidateSiteApplets` flags an unknown referenced applet at startup (wired
  in `main`); default column width deduped via `content.DefaultColWidth` (validation ⇄ render).
  Tests `TestValidateColumnPattern`, `TestValidateSiteApplets`.
- [x] **S11.8 — Forge HTTP hardening** (I8): mutating endpoints require POST (405 + `Allow`);
  `readBody` errors → 400; invalid save → 400 (detail in body). Tests
  `TestMutatingEndpointsRequirePOST`, `TestHandleSaveInvalidReturns400`.

### Slice 4 — Hygiene (I9, I10, I11) — ✅ done (16/07/2026)
- [x] **S11.10 — Clarify phantom `"type"` field** (I9): documented in `content.Page` +
  `docs/content.md` as a tolerated, preserved, never-interpreted descriptive hint (page kind
  inferred from populated fields). Kept, not removed (used pervasively, preserved by the studio).
- [x] **S11.11 — Dedupe re-emit invariant + rune-safe centering** (I10): shared
  `oascii.Builder.ReemitAttrs` used by `Newline` (sticky) and `render.reemitState`;
  `render.center` counts runes. Test `TestReemitAttrs`.
- [x] **S11.12 — Firmware minor cleanups** (I11): fixed the `term.s` comment (byte sent is
  `:`/`$3A`), removed the dead `hires_fillbox` block in `hires.s`; reassembled via `make client`.
  The tight XMODEM `$4000`↔`$B800` buffer margin is left as a documented latent observation.

## Sprint 8 — Close out file transfer + news — 🎯 in progress (27/06/2026)
> Wraps up Epic G (transfer) and starts Epic D (content/news).
- [~] **S1 — User-editable filename at reception** (terminal): before saving, the
  received file's proposed Sedoric name can be edited at the keyboard, then it is
  saved (Sedoric/LOCI) under the edited name. Reuses `input_line`; validated in
  `oric1-emu` via `--type-keys`.
- [ ] **S2 — News / announcements page** (server): not started. Dated, persisted
  news applet (store pattern à la `internal/user`), Go-tested + driver. (D2.)
- [x] **S3 — Cassette (.TAP) save spike** ✅ (27/06): **feasible** (`oric1-emu`
  captures ROM CSAVE to a host `.tap`; Atmos hooks `$E607/$E75A/$E65E/$E93C`,
  buffers `$027F`/`$02A8`) but **deferred** — ML CSAVE recipe ≈ the Sedoric R&E
  effort for low value (Sedoric/LOCI already cover storage). See `docs/transfer.md`.

## Sprint 3 — Content modules
- [x] Messaging / forum (read, post) — done 17/07/2026 (applet `forum`, cf. Sprint 7 #1)
- [ ] News / announcements page
- [ ] Interactive mini-game (e.g. Connect Four / tic-tac-toe) to validate interactivity
- [~] **File transfer (XMODEM)**: **server side done** (`internal/xmodem`,
  `server/internal/files` library, `download`/`upload` applets, `-files`/
  `-max-upload` flags, studio, doc `docs/transfer.md`); **remaining is the Oric terminal**
  (6502 XMODEM + SD/floppy/cassette storage). See backlog G1.
  - **Sedoric floppy write PROVEN** (24/06) in the emulator: `SAVE`
    persists to the `.dsk` with the `--disk-writeback` flag (root cause of the fake
    block — it was not the API addresses).
  - **Full reverse of the SAVE dispatch** (24/06): map of routines/variables
    established (`docs/sedoric-api.md`).
  - **✅ Sedoric save VALIDATED end-to-end on SEDORIC V3.0**: a file is
    written and persisted to the `.dsk` from machine language. Recipe (disassembled
    V3.0 manual): `JSR $04F2` (V3.0 overlay switch) → system variables →
    `JSR $DE9C` (XSAVEB) → `JSR $04F2`. Public vectors confirmed identical
    V1.0/V3.0. `client/sedoric.s` finalised (assembles). Triggering by `term.s`
    already wired.
  - **✅ Bootable terminal floppy**: `client/build-disk.sh` produces
    `term-boot.dsk` (Sedoric master + TERM.COM); the terminal **runs** from the
    floppy (`LOAD"TERM":CALL#1000`, modem menu displayed). ACIA `$0380` (LOCI
    modem) at runtime to coexist with the Microdisc. **Remaining**: hands-free
    auto-start (replace the master's boot program) + **test on a real Oric**.

## Sprint 4 — Real hardware connection — ⏳ in progress
- [x] **WiFiModem + LOCI connection doc** (`docs/hardware-connection.md`): chain
  Oric→ACIA→modem→TCP, ACIA addressing `$031C` / LOCI modem `$0380` (MIA `$03A0-$03BF`), 6551 registers,
  AT commands (`ATD`/`ATDT#`/`AT$CA`/`AT$CV1`), 9600 8N1 settings, troubleshooting.
- [x] **Oric client/terminal program** (`client/term.s`) — standalone
  6502 terminal (modem menu, directory, manual entry, Hayes dialling, RX/TX
  terminal mode), validated end-to-end in the emulator. (done in Sprints 1–2)
- [x] **Oric ASCII-art welcome screen**: server banner enriched with an "ORIC"
  art over 5 lines (5×5 glyphs), centred and OASCII-compliant (≤ 40 columns), yellow/cyan colours.
- [ ] **Test on a real Oric** — *awaiting hardware*. Hardware acceptance protocol
  (T1–T9) ready: `docs/hardware-connection.md` §7.

## Sprint 5 — Deployment — ✅ done (IN PRODUCTION ✅)
- [x] **Deployed in production** on the pavi3617 LXC (systemd service `bbsoric`, `enabled`+`active`)
  via `make deploy` (mechanism reused from telenet). Static Go binary linux/amd64, `DynamicUser`.
- [x] **Public exposure validated**: `pavi.3617.fr:6502` (telnet) — banner + navigation OK
  from the public Internet.
- [x] **Dedicated monitoring / alerting**: local HTTP endpoint `/healthz` + `/metrics`
  (Prometheus format, `-metrics-addr` flag), probe `scripts/monitor.sh`
  (healthz/TCP + email alert) triggered by `bbsoric-monitor.timer` (5 min).
  Deployment integrated into `vps-deploy.sh`. Doc: `docs/monitoring.md`.
- [x] **Containerisation (Docker)**: multi-stage `Dockerfile` (static binary →
  alpine image ~18 MB, non-root, `/healthz` healthcheck), `docker-compose.yml`
  (accounts volume, restart), `make docker-build/up/down` targets. Build and run
  validated (BBS on 6502, healthcheck `ok`). Doc: `docs/docker.md`. (prod = systemd)
- [x] **User documentation**: `docs/user-guide.md` (connecting from
  a real Oric and from a PC for testing, navigation, accounts, troubleshooting).
- [x] **State backup & restore**: `scripts/backup.sh` (timestamped
  `tar.gz` archive of accounts+files+content, rotation, hot) +
  `scripts/restore.sh`, daily systemd timer (`bbsoric-backup.{service,timer}`),
  e2e test `scripts/test-backup.sh` (13 cases green), doc `docs/backup.md`.
  Deployment integrated into `vps-deploy.sh`.

## Community announcement (alpha) — ✅ published (25/06/2026)
- [x] **Alpha announcement published** on the **Defence Force** forum:
  <https://forum.defence-force.org/viewtopic.php?t=2897> (server + Oric terminal +
  "Forge" studio). Video: <https://youtu.be/YRFBYkpsKMc>. Record: `docs/communication.md`.
- [x] **Public GitHub repository**: <https://github.com/benedictemarty/bbsoric>
  (history purged of internal IPs via `git filter-repo`).
- [ ] **Test feedback on real hardware** (call for contributions from the announcement):
  terminal rendering on iron, real serial XMODEM timing, Sedoric write on a physical
  drive (Microdisc/LOCI). To be recorded in `docs/communication.md`.

## "Forge" studio — content tooling ⏳ (in progress)
`studio/` sub-project: local Go web app to edit the `site.json`(s) and deploy by
profiles. See `docs/adr/0003-studio-forge.md`.
- [x] **Restructured** the repository into 3 sub-projects `server/` `client/` `studio/`
  (`internal/{content,oascii}` shared at the root).
- [x] **Forge**: web editor (menu/page/applet pages), coloured OASCII preview, validation
  by `internal/content`, atomic save.
- [x] **Applet parity**: the "▶ applet" list covers all server applets
  (`login`/`register`/`guest`/`download`/`upload`/`who`/`chat`) with tooltips —
  editing/preview of the **Community** menu (Sprint 7) operational.
- [x] **Deployment by profiles** (dev/int/prod): validate → backup → overwrite → reload,
  dry-run; `dev` local (hot-reload), `int`/`prod` ssh/scp. Validated end-to-end.
- [x] **Menu over a background screen**: a `raw` page (composed 40×28 buffer) combines
  with `entries` — background décor + key navigation (page or applet),
  navigation editor integrated into the "Screen" tab.
- [x] **Declarative input pages** (`content.Form`): generic `form` applet
  (login/registration) driven by declared fields (key/label/secret) + action;
  form editor in the studio. Sensitive logic (hashing) stays server-side.
  **In-place retry** (`Retries`) + **failure page** (`Fail`, also for ▶ applet
  entries) in addition to the success `Next`.
- [x] **Cursor positioning (plot X,Y)**: terminal sequence `1F col row`
  (`oascii.Plot`/`Builder.At`), positionable form fields (`Field.At`),
  X/Y columns in the studio. Décor + prompts placed at absolute coordinates.
- [x] **Differential screen buffer** (`oascii.Screen`): "dirty cells" rendering —
  emits only the modified cells (segments + plot X,Y). Basis for dynamic screens
  (games, animations) over a slow serial link.
- [ ] Advanced multi-site (creating new files from the UI), backup management.
- [ ] Authentication if the studio were to be exposed (today local-only).

## Sprint 7 — Communication between callers (state-of-the-art parity) — ⏳ in progress
> Historic heart of a BBS, today absent (the "Guestbook" is static).
> Gap analysis: `docs/state-of-the-art.md` §6. Each feature = one applet
> (`bbs.Register`) + a persisted store modelled on `internal/user`.
- [x] **Who's online + chat / paging** (#3) — presence registry
  (`server/internal/presence`) + `who` and `chat` applets (real-time room,
  non-blocking broadcast, **Community** menu). Unit tests + two-client integration,
  `-race` clean. *(leverages the multi-session engine)*
- [x] **Writable one-liner wall** (#2, 17/07/2026) — the static Guestbook becomes a
  persisted message wall (applet `wall`, package `server/internal/wall`: atomic JSON store
  modelled on `internal/user`, bounded ≤ 78 chars / ≤ 200 messages, ASCII-sanitised).
  Server flag `-wall`; menu entry « Mur de messages ». **Establishes the "persisted-write
  applet" pattern** reused by the forum (#1). Tests: store unit + TCP integration
  (`TestWallPostAndPersist`/`TestWallEmptyReturns`). Studio applet list updated.
- [x] **Message base / forums** (#1, 17/07/2026) — read + post in threads with pagination.
  Applet `forum` + package `server/internal/forum` (atomic JSON store, thread→posts,
  persistent ID counter, bounded ≤ 38-char titles / ≤ 200-char posts / ≤ 100 threads /
  ≤ 500 posts-per-thread, ASCII-sanitised). List paged (`S/P`, `1-8` to open, `N` new),
  thread view paged (`R` reply). Server flag `-forum`; « Forum » entry in the Community
  menu. Reuses the wall's persisted-write pattern; sanitisation shared via
  `oascii.SanitizeText`. Store unit tests + TCP integration (`TestForumCreateReadReply`).
  *Full-redraw pagination (not the differential buffer), consistent with the wall/menu
  applets.* *The* feature that moves from "menus" to "BBS".
- [ ] **Private messaging** (#4), **RSS→OASCII news** (#5), **door game** (#6).

---

## Decisions made
- **Server language**: Go 1.26 (`cmd/bbsd`, `internal/server`, `internal/bbs`).
- **Hosting**: cloud VPS with fixed IP (public Internet exposure 24/7).
- **Public port**: `6502`.
- **Testing**: **single** emulator `Oric1/oric1-emu` (Phosphoric) via TCP socket (`--serial tcp:`).

## Client review (Oric terminal) — 26/06/2026
Engineering review of the 6502 client (`docs/client-review.md`). **Fixed**: LOCI base
`$03A0`→**`$0380`** (the `$03A0` was the MIA space, hence the frozen keyboard), plot
clamp (anti out-of-VRAM), XMODEM receive bound (anti overflow), **uppercase**
(SHIFT) and **backspace** (DEL, client + server). **Documented deferrals**:
RX flow control (#1), modem/DCD codes (#6), telnet IAC (#7), client tests (#12).

## Open decisions (ADRs to formalise)
1. **ACIA addressing** — ✅ settled: `$031C` (standard ACIA) and **`$0380`** (LOCI
   WiFi modem; `$03A0-$03BF` = MIA space, not the modem). To confirm on real iron.
2. **Telnet protocol** — full IAC negotiation vs minimal filtering (current). To settle in Sprint 1.
3. **OASCII rendering** — exact Oric Teletext attribute table to validate on the emulator (Sprint 1).
