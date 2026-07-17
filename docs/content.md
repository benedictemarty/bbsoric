# Dynamic content — page flow in JSON

The sequence of BBS screens (menus, content pages, navigation) is
**driven by a hot-reloaded JSON file**: editing it updates the BBS
without recompiling or restarting (applied within ~2 s, at the next navigation
of sessions in progress).

- Versioned reference file: [`../content/site.json`](../content/site.json)
- In production: `/etc/bbsoric/site.json` (edited directly on the server;
  deployment never overwrites it — it only seeds it at initialization).
- Launch: `bbsd -content /etc/bbsoric/site.json` (without `-content`, default
  built-in content).

## Format

```json
{
  "start": "main",
  "pages": {
    "main": { "title": "MENU PRINCIPAL", "entries": [ ... ] },
    "info": { "title": "INFOS", "lines": [ ... ] }
  }
}
```

- **`start`**: identifier of the starting page.
- **`pages`**: dictionary `identifier → page`.

### The page (single type)
A page has a **title** and, **optionally**, **text** (`lines`) and/or
**choices** (`entries`).

> A JSON `"type"` field (e.g. `"menu"`/`"page"`/`"applet"`) is often present in
> content and fixtures: it is a **human-readable hint only**. There is no Go field
> for it — the engine infers the page kind from which fields are set (`entries` →
> menu, `applet`, `form`, `datawindow`, `hires`, otherwise content). It is tolerated
> and preserved on studio round-trip, but never interpreted.


- **with `entries`** → **interactive** screen: a key routes to the chosen entry;
  the optional text (`lines`) is displayed **above** the choices;
- **without `entries`** → **content** screen: a key goes back
  (character mode, see ADR-0002).

**Text lines** (`lines`). A line carries an Oric **style** (all optional):
```json
{ "text": " ALERTE ", "ink": "white", "paper": "red", "blink": true,
  "doubleHeight": false, "altCharset": false, "inverse": false }
```
- `ink`: text color — `black red green yellow blue magenta cyan white` (default white).
- `paper`: **background** color (same names; default black).
- `blink`: **blinking** · `doubleHeight`: **double height** ·
  `altCharset`: **BBS font** (rules/frames, blocks, halftones, symbols — redefined
  Oric alternate charset, see `tools/genfont`) · `inverse`: **inverse video**.

> With `altCharset`, the characters are no longer standard ASCII but the **BBS font**
> (e.g. `a`/`b`/`c`/`d` = corners, `-`/`|` = rules, `0` = solid block, `5`/`6`/`7` = halftones).
> The studio provides a **palette** (Edit tab) to insert these glyphs; the preview and
> the Oric terminal use the same font.

**Multicolor on one line** — split into `segments`, each with its own style:
```json
{ "segments": [
  { "text": "Score ", "ink": "white" },
  { "text": "42", "ink": "yellow", "blink": true },
  { "text": " GAME OVER ", "ink": "white", "paper": "red", "inverse": true }
]}
```
In a line with segments, an unset attribute reverts to the **default value**
(white ink, black background, no effect); the engine only emits attribute **changes**.

> Oric reminder: each attribute byte **occupies a screen cell** (a change "eats" a
> column) and the ULA resets attributes at the start of each line. For **dynamic** content
> (animation, computed values, interaction, positioning, elaborate semi-graphical art),
> write an **applet** (see `studio/README.md` / ADR-0001).

**Choices** (`entries`) — an entry **navigates** (`target`) **or launches an applet** (`applet`
+ `next`). A menu can therefore offer several applets to choose from.

Navigation entry:
```json
{ "key": "1", "label": "Informations", "target": "info" }
```
- `key`: key (case-insensitive).
- `target`: page identifier **or** special target:
  - `__quit__`: ends the session,
  - `__back__`: previous page (stack),
  - `__home__`: starting page.

Applet entry:
```json
{ "key": "1", "label": "Se connecter", "applet": "login", "next": "main" }
```
- `applet`: name of the applet to launch when the entry is chosen. Registered applets:
  `login`, `register`, `guest`, `download`, `upload`, `who`, `chat`, `wall`, `datawindow`.
  **Adding an applet** = writing a small Go function and registering it.
- `next` (optional): page to go to **after the applet succeeds** (empty = stay).

> **`wall`** (mur de messages) — persisted one-liner wall (the historic "guestbook"):
> reads the latest messages then lets the caller post one. Backed by `server/internal/wall`
> (atomic JSON store), enabled for persistence by the server flag `-wall <file.json>`
> (without it, the wall lives in memory only). Messages are bounded (≤ 78 chars, ≤ 200
> kept) and ASCII-sanitised server-side.

> Compat: a page can also carry `applet` (+ `next`) at the **page** level (applet
> auto-launched on arrival). A historical mechanism kept for hand-written JSON;
> prefer an **applet entry**.

## Rendering (OASCII reminder)

Titles in yellow, 40-column rules, keys in cyan, labels in white, prompts
in green. A color attribute byte occupies a screen cell (see `oascii.md`) — avoid
labels that are too long to stay within 40 columns.

## Validation

An invalid JSON (syntax, `start` not found, nonexistent target, unknown type)
is **rejected**: the previous version stays in service and the error is logged.
The `internal/content` test also verifies that the repository's `content/site.json` is valid.
