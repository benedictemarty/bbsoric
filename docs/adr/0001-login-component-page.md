# ADR-0001 — Login: interactive component called by a page, hashed persistence

- **Status**: Accepted
- **Date**: 2026-06-22
- **Sprint / Backlog**: Sprint 2 — item **C4** ("I want to identify myself and retrieve my profile")
- **Deciders**: bmarty
- **Replaces / complements**: none (first formalised ADR of the repository)

## Context

The BBS engine (`internal/bbs/engine.go`) runs a **JSON-driven page flow**
(`internal/content`): a `Site` contains `Page`s of type `menu` or `page`, and
navigation is done by `Entry`s whose `Target` is either a page identifier, or a
**special target** (`__quit__`, `__back__`, `__home__`). The content is editable and
**hot-reloaded**; no logic is coded in the JSON.

User identification must be introduced without breaking these properties. Three
constraints structure the decision:

1. **Local echo of the Oric terminal** (`oric-client/term.s`): the typed character is shown
   by the Oric before sending → server-side password masking with `*` is ineffective.
2. **Line-by-line input** (`server.Session.ReadLine`): no character-by-character reading
   nor remote cursor control.
3. **"Zero dependency" philosophy**: `go.mod` declares no external dependency;
   we want to keep it that way.

## Decision

### 1. Login is an *applet* launched by a *page of type `applet`*, as a gate at CONNECT

**Revision 2 (2026-06-22)**: in line with historical BBSes, identification happens
**at connection time, before the main menu**. We generalise the pattern "pure page →
Go behaviour" with a **3rd page type: `applet`**. The page stays **JSON/text**,
it just declares *which applet* to run; the **applet** is a small specific Go unit
(login, registration, guest… then games, polls), registered by its name in a
**registry**. We therefore do **not** add special targets per function.

- New `applet` page type with two fields: `applet` (registered name) and `next`
  (page to go to **after success**, e.g. `main`). An `applet` page can also carry
  `lines` (intro text displayed before launching the applet).
- The **start page** (`site.Start`) is a pure-JSON auth menu whose entries
  point to `applet` pages (`login`/`register`/`guest`) like any other
  page. As long as the user is neither logged in nor a guest, they stay on this gate.
- **Applet registry** (package `bbs`): `Register(name, Applet)`. An applet has the
  signature `func(ctx, *server.Session, *AppContext) Outcome`; it does its own OASCII
  rendering and its own input, **does not know** the page flow → testable in isolation.
  `AppContext` injects the dependencies (`*user.Store`, session state); `Outcome` tells
  the engine what comes next (success → `next`, cancellation → back, quit).
- The **engine** (`engine.go`), upon reaching an `applet` page, resolves the applet by its
  name, runs it, then applies the `Outcome` (navigates to `next` on success).

**Adding an applet** = writing a small Go function + registering it; **placing** it in
the BBS = editing the JSON. No navigation change.

### 2. Session state

`runSite` receives a minimal session state carrying the current user
(`user *user.User`, `nil` if guest), so that the following screens personalise
the display ("Hello {handle}, call no.{n}") and, later, restrict access.

### 3. Persistence and hashing

- `user.User` model: `Handle`, `PassHash`, `Created`, `LastLogin`, `Calls`.
- File store **`users.json`** with a **lock** (concurrent writes) and **atomic
  write** (temporary file + `rename`). Symmetric to the JSON choice already adopted for
  the content, but for read **and** write.
- Passwords **never in cleartext**: **PBKDF2-HMAC-SHA256** hashing (`crypto/pbkdf2`,
  Go 1.24+ **stdlib**), random salt per account (`crypto/rand`). Self-describing encoded
  format: `pbkdf2$sha256$<iter>$<salt_b64>$<hash_b64>`.

### 4. Cleartext password on screen: accepted for now

Lacking controllable echo, the password input is **visible** on the Oric screen.
We accept it (warning displayed), the **transport** confidentiality being already
covered by **TLS on `:6992`**. Real masking (`IAC WILL ECHO` negotiation or
"no-echo" mode on the `term.s` side) is deferred to a later increment.

## Consequences

**Positive**
- The flow stays 100% JSON-driven and hot-reloadable; no frozen page.
- The login component is isolated, testable without network, and the "special target →
  component" pattern is reusable (future: posting a message, playing, etc.).
- No external dependency added.

**Negative / to watch**
- The password transits in cleartext **on screen** (not on the network if TLS) as long as
  no-echo is not done.
- Login at CONNECT removes the need for conditional menu entries at the start
  (the user passes the gate before reaching the main menu). Dynamically hiding
  entries by role remains a later increment (JSON visibility rules).
- Concurrent writing of `users.json` imposes a lock + atomic write (accounted for).
- Input (single key for menus, line+RETURN for text fields) is the subject of
  a dedicated ADR: see **ADR-0002**.

## Rejected alternatives

1. **Login page hard-coded (in Go) before the flow**: it does ensure login at CONNECT
   but breaks the "everything is JSON-driven" uniformity and is not reusable. We
   prefer making the **JSON start page** the auth gate (same result, without freezing
   the screen in code).
2. **Login as a page `type` in JSON** (e.g. `"type":"login"`): mixes data and
   interactive behaviour in the content; the special target is simpler and consistent
   with `__quit__`/`__back__`/`__home__`.
3. **bcrypt/argon2 via `golang.org/x/crypto`**: better state of the art, but adds an
   external dependency; PBKDF2 stdlib is sufficient for the use case and preserves "zero
   dependency".

## Increment plan (Sprint 2 / C4)

1. **`internal/user`**: model + atomic hashed store + unit tests (without network). ← *this increment*
2. Component `RunLogin` / `RunRegister` / guest access inserted via special targets (emulator test).
3. Session state + personalised welcome.
4. (Later) Conditional entries, password no-echo.
