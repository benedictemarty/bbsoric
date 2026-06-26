# ADR-0003 — "Forge" studio: Go web app, shared internal/, deployment by profiles

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: bmarty
- **Related to**: ADR-0001/0002 (JSON-driven content, `applet` page type)

## Context

The BBS content (`content/site.json`: `menu`/`page`/`applet` pages, auth gate) was
edited **by hand**. The user wants a **3rd sub-project** dedicated to it — a **"forge"
studio** — to generate/edit the site(s) **and deploy them** to several environments
(**prod / int / dev**) via **profiles**.

## Decision

1. **Three sub-projects**: `server/` (Go server), `client/` (Oric terminal), `studio/`
   (the forge). The **shared** packages `content` and `oascii` stay in the **root**
   `internal/`: the Go visibility rule would forbid `studio/` from importing
   `server/internal/...`, so we place the reused code at the root so the studio
   uses **exactly** the same validation and the same palette as the server (zero
   divergence). The server-specific code (`bbs`, `server`, `user`) is under `server/internal/`.

2. **Studio = Go web app** (`studio/cmd/forge`), **stdlib only**, **embedded** assets
   (`embed`). **Local development** tool: bind `127.0.0.1`, **no authentication**.
   - `studio/internal/store`: lists/loads/**saves after validation** (`content.Parse`),
     atomic write, anti path-traversal.
   - `studio/internal/preview`: renders a page as **40-column coloured HTML** faithful to
     the engine (reuses `oascii` + `content.Ink`).
   - `studio/internal/deploy`: deployment by **profiles**.

3. **Studio = source of truth; deployment OVERWRITES** the target (abandoning the
   "sow only once" rule), **after validation** and **timestamped backup**
   (`<target>.bak.<timestamp>`). **Dry-run** by default in the UI; **confirmation** before
   a real deployment.

4. **Profiles PER SITE**: each site has its trio `dev`/`int`/`prod` in
   `deploy/profiles/<site>/<env>.conf` (where `<site>` = file name without `.json`, e.g.
   `deploy/profiles/site/dev.conf`). `KEY=VALUE` format. An `<env>.conf.example` serves as
   **default**; the real `<env>.conf` (gitignored) **takes precedence**. `dev` = **local** (file
   copy, bbsd hot-reloads); `int`/`prod` = **ssh/scp** (reuses the mechanism of
   `deploy/vps-deploy.sh`, no dependency). Fields: `LOCAL HOST USER PORT CONTENT_PATH
   SERVICE RELOAD` (`RELOAD` = `none|reload|restart`).

5. **Indented save**: the studio re-indents the JSON on write (`json.Indent`,
   2 spaces) — readable files, stable git diffs, all keys preserved (`_comment`).

## Consequences

**Positive**
- Assisted editing + colour preview, **validation identical** to the server (reuses `content`).
- Multi-environment deployment **traceable** (validation, backup, dry-run, log).
- Zero external dependency; three clear sub-projects.

**Negative / to watch**
- The studio has **no authentication**: it must stay **local** (`127.0.0.1`).
- Deployment **overwrites** the target: hot edits made directly on a server
  are no longer the source of truth (but backed up before overwriting).
- The preview is **faithful but approximate** (the exact semantics of Teletext
  attribute cells can be refined).

## Rejected alternatives
- **Python/Flask studio** (like the telenet studios): duplicates the validation outside the
  Go package and adds a Python dependency.
- **Everything under `server/internal/`**: would prevent the studio from reusing `content`/`oascii`
  (Go visibility) → duplication.
- **"Sow only once" deployment**: would defeat the purpose of a deployment studio.
