# Forge Studio — content editor for the BBS Oric

**Local** web tool to edit the BBS `site*.json`(s) (pages `menu` / `page` /
`applet`, auth gate), with **coloured OASCII preview** and **validation by the same package
as the server** (`internal/content`) — so zero format divergence. Later:
deployment by profiles (dev / int / prod).

## Run

```bash
make studio                  # go run ./studio/cmd/forge -addr 127.0.0.1:8080
# or
go run ./studio/cmd/forge -addr 127.0.0.1:8080 -content-dir content
```

Then open <http://127.0.0.1:8080>. **Development** tool: it listens on `127.0.0.1`
only (not exposed, no authentication).

## Features

- Load a site, list/add/rename/delete pages.
- Edit by form according to type: `menu` (key/label/target entries),
  `page` (text lines + ink), `applet` (applet name + `next` page + intro).
  A "▶ applet" menu entry is wired via a dropdown of known applets
  (`login`, `register`, `guest`, `download`, `upload`, `who`, `chat`),
  with a tooltip describing each one. To keep aligned with the applets
  registered on the server side (`bbs.Register`).
- **"ULA simulator" preview** (240×224 canvas): renders the page's OASCII stream like the
  Oric video chip (embedded Oric font, attributes, inverse, double height, blink;
  approximated semi-graphics). No ROM or emulator. The rendering comes from `internal/render`
  (same byte stream as the server).
- **Validate** (rejects inconsistent JSON) and **Save** (atomic write).
- **Deploy** to an environment via a **profile** (Simulate / Deploy).

## Deployment by profiles (dev / int / prod)

Profiles are **specific to each site**: `deploy/profiles/<site>/<env>.conf` where `<site>`
is the file name without `.json`. Each site has its trio `dev` / `int` / `prod`.
`KEY=VALUE` format. A `.conf.example` serves as a **default**; copy it to `.conf` for real
infrastructure (the `.conf` is gitignored and **takes precedence** over the example):

```bash
# profiles for the site "site.json"
cp deploy/profiles/site/prod.conf.example deploy/profiles/site/prod.conf   # then fill in
```

The studio (source of truth) **validates → backs up (timestamped) → overwrites → reloads**. The
**Simulate** button (dry-run) shows the actions without executing anything; **Deploy** asks for
confirmation. `dev` = **local** (file copy, bbsd hot-reloads); `int`/`prod`
= **ssh/scp**. Fields: `LOCAL HOST USER PORT CONTENT_PATH SERVICE RELOAD`
(`RELOAD` = `none|reload|restart`).

API: `GET /api/profiles?site=`, `POST /api/deploy?site=&profile=&dryRun=`.

## Architecture

```
studio/
  cmd/forge/         web server (net/http, embedded assets) + API handlers
  internal/store/    lists / loads / saves the site*.json (validates before writing)
  internal/preview/  renders a page as coloured HTML (reuses oascii + content.Ink)
  web/               index.html, app.js, style.css (embed)
```

API: `GET /api/sites`, `GET /api/site?name=`, `POST /api/validate`,
`POST /api/save?name=`, `POST /api/preview?page=`.

Stdlib only, no external dependency.
