# ADR-0004 — DataWindow: typed data sources backed by pure-Go SQLite

- **Status**: Accepted
- **Date**: 2026-06-27
- **Deciders**: bmarty
- **Related to**: ADR-0001/0002 (JSON-driven content, `applet` page type), ADR-0003 (shared `internal/`)

## Context

The BBS so far only served **static** content (menus, text pages, forms). The
sibling project **telenet** (Minitel server) has a **DataWindow** subsystem —
typed *data sources* (tables) presented as a paginated, navigable grid with full
**CRUD** and validation, backed by **SQLite**. The owner wants the **same notion**
in bbsoric, with **SQLite parity** and **full CRUD** from the first increment.

bbsoric differs: **OASCII** rendering (Oric 40×28, not Videotex), a **content/applet**
engine, JSON-only persistence so far, one goroutine per session.

## Decision

1. **Pure-Go SQLite** (`modernc.org/sqlite`, no CGO) is the data backend — the only
   new third-party dependency. It keeps cross-compiling the Linux server trivial and
   matches telenet's proven engine, which we port near-verbatim.

2. **Model types live in `internal/content`** (`SourceDonnees`, `ColonneDef`, the page
   `DataWindow` descriptor) so `Site.Validate()` checks them at load (and the studio
   can edit them later) — same rationale as ADR-0003 (shared, server-agnostic model).
   The **engine** (`server/internal/datawindow`) is server-side (it owns the DB).

3. **The grid is an applet** (`datawindow`), launched by a `DataWindow` page (a new
   case in the engine `switch`) or a menu entry. It renders through the **`oascii.Screen`
   differential buffer**: moving the selection re-emits only the two changed rows over
   the slow serial link. Selection/header highlight uses Oric **per-character inverse**
   (bit 7), not a serial attribute.

4. **SQL-injection guards are ported byte-for-byte** (`ValiderNomSQL`/`ValiderTypeSQL`,
   identifier whitelist regex, `?` placeholders for all values). They live in `content`
   so both the loader and the engine share one source of truth. Dedicated security tests.

5. **40-column budget** is enforced in `Site.Validate()`: a too-wide grid fails at load,
   not at render. Validation also rejects unknown sources/columns and mismatched widths.

## Consequences

- Sources are **initialized at startup** (`-data <dir>` flag → one SQLite file,
  `CREATE TABLE IF NOT EXISTS` + seed if empty + auto-migration of new columns).
  Idempotent, so it doubles as a lazy re-init.
- `modernc` returns TEXT as `[]byte`; a central `cellString` normalizes engine output
  to `map[string]string` rows so the renderer never shows numeric garbage.
- Oric has no arrow keys: navigation uses `+`/`-` (selection), `S`/`R` (pages),
  `V` (detail), `N`/`E`/`D` (CRUD, if editable + logged in), `F`/`C` (filter), `Q`.
- Studio editing of sources/data is a **later increment** (not in this one).
