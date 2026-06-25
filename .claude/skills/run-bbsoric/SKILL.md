---
name: run-bbsoric
description: Build, run, test and drive the BBS Oric server (bbsd) — a TCP/telnet BBS for Oric computers — plus the Forge studio web app and the Oric terminal .tap. Use when asked to run, start, launch, build, test, smoke-test, screenshot, or drive the bbsoric server / studio / terminal.
---

# Run BBS Oric

`bbsoric` is a **TCP/telnet BBS server** (Go) for Oric micro-computers. It emits
**OASCII** (ASCII text interleaved with Téletexte attribute bytes `$00-$1F`).
The repo also ships the **Forge studio** (Go web app) and the **Oric terminal**
(6502 firmware → `.tap`).

The server has no GUI — you drive it over a socket. The agent harness is
**`.claude/skills/run-bbsoric/driver.py`**: it connects, sends single-key menu
choices, reads each screen, and renders the OASCII into readable text.

All paths below are relative to the repo root (`<unit>/`).

## Prerequisites

```bash
# Ubuntu/Debian. Go 1.26+ ; xa65 = 6502 assembler (only for `make client`).
sudo apt-get update && sudo apt-get install -y golang xa65 python3
```
Verified host had: `go1.26.4`, `xa` (pkg `xa65`), `python3` (stdlib only — the
driver needs no pip packages).

## Build

```bash
make build      # -> ./bbsd        (server binary)
make client     # -> client/term.tap  (Oric terminal; needs xa)
```

## Run + drive the server (agent path) — START HERE

Launch the server (bind localhost; enable the local /healthz,/metrics endpoint):

```bash
./bbsd -addr 127.0.0.1:6502 -metrics-addr 127.0.0.1:6510 -idle 30s >/tmp/bbsd.log 2>&1 &
sleep 1
curl -s http://127.0.0.1:6510/healthz        # -> ok
```

Drive the telnet side with the smoke driver (connects, navigates the menu from
fresh connections, asserts the expected screens, writes text captures):

```bash
python3 .claude/skills/run-bbsoric/driver.py            # host/port default 127.0.0.1 6502
# -> prints each screen, [OK]/[FAIL] per check, exit 0 if all pass.
# captures: /tmp/bbs-01-banner.txt .. /tmp/bbs-04-guestbook.txt
```

Expected tail:
```
=== Résultat ===
  [OK] banniere+menu
  [OK] touche 1 -> INFORMATIONS SYSTEME
  [OK] touche 2 -> PROPOS
  [OK] touche 3 -> LIVRE
```

Drive it manually (the `BBS` class is importable):

```bash
python3 - <<'PY'
import sys; sys.path.insert(0, ".claude/skills/run-bbsoric")
from driver import BBS
b = BBS("127.0.0.1", 6502); b.connect()
print(b.render(b.read_screen()))      # banner + main menu
b.send("1"); print(b.render(b.read_screen()))   # -> INFORMATIONS SYSTEME
b.send("Q"); b.close()
PY
```

Serve the production content (adds the **Fichiers** download/upload menu) with
`-content content/site.json -files /tmp/bbsfiles`.

Stop the server:
```bash
pkill -x bbsd
```

## Run the Forge studio (web app)

```bash
make studio        # go run ./studio/cmd/forge -addr 127.0.0.1:8080
# in another shell:
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/    # -> 200
pkill -f forge
```
For a screenshot, drive `http://127.0.0.1:8080/` with `chromium-cli`.

## Test

```bash
make test     # go test ./... — all packages OK (server, studio, internal)
```

## Run the human path

`make run` builds and runs the server on `0.0.0.0:6502`, then blocks — connect
with a real telnet/`nc` client, Ctrl-C to stop. Useless headless; use the driver.

## Oric terminal in the emulator (optional, external dep)

`scripts/test-emulateur.sh` boots the `.tap` in `oric1-emu` (Phosphoric) wired to
the BBS over `--serial tcp:`. It needs the emulator at `$HOME/Oric1/oric1-emu`
(NOT in this repo — absent on a clean machine). Set `ORIC_EMU` to override. The
Sedoric disk build for the terminal is `client/build-disk.sh` (see
`docs/sedoric-api.md`); it also needs `$HOME/Oric1` assets.

## Gotchas

- **Emulator: never `--loci` together with `--acia-addr 03A0`.** Both map to
  `$03A0-$03BF` (LOCI MIA + ACIA); the MIA shadows the ACIA *and* drives the PSG
  that the keyboard scan reads → `get_key` spins → the terminal freezes on the
  directory. `--loci` is for SD/flash ops, not the modem. Correct wiring for the
  BBS terminal (menu option `2` = `$03A0`):
  `oric1-emu -t client/term.tap -f -r roms/basic11b.rom --serial picowifi --acia-addr 03A0 --serial-buffer 512`
  (no `--loci`). Phosphoric ≥ 1.27.2 warns on the overlap. See `phosphoric-findings.md` (F1).
- **OASCII, not plain text.** The wire bytes `$00-$1F` are colour/attribute
  codes, not text. `driver.render()` turns them into `·`; don't expect clean
  strings on a raw `nc`. Single key = one menu choice (no Enter for menus).
- **`ReadKey` ignores CR/LF/NUL** (`session.go` — they're line-ending residue
  from `nc`-style clients). So a "press any key" prompt (e.g. "Appuyez sur une
  touche") is NOT satisfied by `\r` — send a **real key like a space `" "`**.
  Multi-hop flows (accueil → guest → menu → submenu) only work with real keys;
  `\r` between screens silently does nothing. Verified guest→Fichiers→download
  with spaces.
- **Production-content flow** (`-content content/site.json`): accueil menu →
  `"3"` (guest) → `" "` (any key) → main menu (incl. `"5"` Fichiers) → `"5"` →
  Fichiers submenu → `"1"` download lists `-files` library. Use spaces, not CR.
- **Default vs production content.** Without `-content`, the server serves a
  built-in menu (1/2/3/Q). The **Fichiers** (download/upload) entry only exists
  in `content/site.json` — pass `-content content/site.json`.
- **`-idle` matters for scripts.** Default idle timeout is 5 min; the driver is
  fast, but set `-idle 30s` so stray test sessions don't linger.
- **`make client` needs `xa`.** Without the `xa65` package the `.tap` build fails
  with "xa: command not found". The Go server/studio don't need it.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `driver.py` prints "connexion … impossible" | Server not running — launch `./bbsd -addr 127.0.0.1:6502 …` first. |
| `curl /healthz` refused | You didn't pass `-metrics-addr 127.0.0.1:6510`. |
| `xa: command not found` (make client) | `apt-get install -y xa65`. |
| Driver screens look truncated/empty | Bump `read_screen(max_wait=...)`; the server is bursty over loopback. |
