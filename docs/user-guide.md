# User guide — connecting to the BBS Oric

Welcome to the **BBS Oric**! This retro messaging server is reachable
**24/7** over the Internet. This guide explains how to connect to it and browse it,
whether you have a **real Oric** or just a modern computer.

## Contact details

| | Address | Port | Protocol |
|--|---------|------|----------|
| **Telnet (cleartext)** | `pavi.3617.fr` | `6502` | telnet / raw |
| **TLS (encrypted)** | `pavi.3617.fr` | `6992` | TLS (terminated by the modem) |

> Port **6502** is a nod to the Oric's microprocessor.

---

## A. From a real Oric (Oric-1 / Atmos)

This is the intended use: an Oric equipped with a **serial interface** (ACIA card or
**LOCI**) and a **WiFi modem**.

1. Plug in the serial interface and the WiFi modem (modem associated with your WiFi).
2. Load the terminal: `CLOAD"TERM"` (the `term.tap` program starts on its own).
3. **Modem menu**: type `1` (ACIA `$031C`) or `2` (LOCI `$0380`) depending on the card.
4. **Directory**: type `1` for *BBS Oric (prod)*, or `M` to enter an
   address by hand.
5. The terminal dials the call by itself and displays the BBS's **color banner**.

Wiring details, AT commands, troubleshooting: see **`hardware-connection.md`**.

For the curious, the call dialed manually from any Hayes modem:

```
ATD pavi.3617.fr:6502
```

---

## B. From a modern computer (to test)

No Oric hardware is needed to **try** the BBS — any telnet
client will do. The Oric colors (serial Teletext attributes)
will not appear correctly on a PC terminal, but navigation works.

### Linux / macOS

```console
# with netcat (recommended: no spurious telnet negotiation)
nc pavi.3617.fr 6502

# or with telnet
telnet pavi.3617.fr 6502
```

### Windows

- Enable the Telnet client ("Windows Features" → *Telnet Client*) then:
  `telnet pavi.3617.fr 6502`
- Or use **PuTTY**: connection type *Telnet*? no — choose *Raw*,
  host `pavi.3617.fr`, port `6502`.

### Encrypted connection (TLS) to test

```console
openssl s_client -connect pavi.3617.fr:6992 -quiet
```

---

## C. Navigating the BBS

On connection, the BBS displays a **banner** then the **main menu**. The
navigation is designed for an Oric keyboard:

- **Menus**: a single key is enough (no need to press Enter).
  Example: `1` opens "System information".
- **Input fields** (login, etc.): type your text then **Enter** (RETURN).
- **Go back / continue**: press a key when the "press a key" prompt
  appears.
- **Quit**: `Q` at the main menu (the BBS replies "A bientot").

### User accounts

The BBS offers (depending on the online content):

- **Guest**: immediate access without an account.
- **Login**: username + password for personalized access.
- **Sign up**: create an account (password stored hashed, never in cleartext).

---

## D. Common problems

| Symptom | Solution |
|---------|----------|
| "Serveur sature, reessayez plus tard" | connection limit reached; try again in a moment. |
| "Trop de connexions depuis votre adresse" | you already have several sessions open from the same IP; close some. |
| Disconnection after a few minutes of inactivity | normal: 5-min inactivity timeout. Reconnect. |
| Colored text unreadable on PC | expected: the colors are Oric attributes, rendered only by an Oric. |
| Cannot connect | check the address/port; the server may be under maintenance. |

---

## See also

- `hardware-connection.md` — wiring and configuration from a real Oric.
- `oascii.md` — how the Oric displays colors (Teletext attributes).
- `README.md` (root) — general overview of the project.
