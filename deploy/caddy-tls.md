# BBS TLS termination by Caddy (Let's Encrypt)

The BBS TLS (`pavi.3617.fr:6992`) is terminated by **Caddy** (auto-renewed Let's Encrypt
cert), not by `bbsd`. Caddy decrypts and relays the telnet stream in cleartext
to `bbsd` on the internal network.

## Network chain

```
Oric + Pico W ──TLS──► MikroTik (.4:6992)  ──►  Caddy CT130 (.3:6992)
  (ATDT#pavi.3617.fr:6992)   dst-nat              │ terminates TLS (Let's Encrypt cert
                                                  │ for pavi.3617.fr, layer4 module)
                                                  ▼
                                          bbsd telnet (.2:6502)  ── cleartext (internal LAN)
```

## Caddy side (CT 130 "caddy-meteolib", atlas)

Standard Caddy is HTTP only; TLS termination of a raw TCP port requires the
**`caddy-l4`** (layer4) module. Binary rebuilt with this module
(`https://caddyserver.com/api/download?...&p=github.com/mholt/caddy-l4`, v2.11.4).

Additions to the `Caddyfile` (backup: `/root/Caddyfile.bak-pre-l4`, binary:
`/root/caddy.bak-pre-l4`):

```caddyfile
{
	email bmarty@mailo.com
	layer4 {
		:6992 {
			@bbs tls
			route @bbs {
				tls
				proxy {
					upstream 192.168.1.2:6502
				}
			}
		}
	}
}

# Let's Encrypt cert for pavi.3617.fr (+ HTTPS info page)
pavi.3617.fr {
	encode zstd gzip
	respond "BBS Oric - connexion telnet TLS sur le port 6992 (ATDT#pavi.3617.fr:6992)" 200
}
```

The `layer4` block listens on `:6992`, matches TLS connections (`@bbs tls`),
terminates them (`tls`, cert managed automatically by SNI) and proxies to `bbsd` in cleartext.
The `pavi.3617.fr` site triggers certificate issuance (ACME challenge on `:443`,
already routed to Caddy).

## MikroTik side

dst-nat rule (number 64): `:6992` redirected to Caddy instead of bbsd.
```
chain=dstnat dst-address=203.0.113.4 in-interface=ether1 protocol=tcp
dst-port=6992 action=dst-nat to-addresses=192.168.1.3 to-ports=6992
;;; telenet - PAVI Oric TLS via Caddy (.4:6992 -> .3:6992)
```

## Notes

- **Verifiable cert**: the Pico W can enable `AT$CV1` with the Let's Encrypt root CA
  (ISRG Root X1) loaded by `AT$CA` — the presented cert is now trusted.
- **bbsd `-tls-addr 6992`** becomes redundant (Caddy handles the public TLS). The internal
  TLS listener of bbsd on `.2:6992` is no longer in the path; it can be removed from
  the systemd unit (`deploy/bbsoric.service`) at a future deployment. `bbsd` now only
  serves **cleartext telnet on `.2:6502`** to Caddy.
- Internal Caddy→bbsd hop in **cleartext** on the LAN (trusted network); the encryption
  is end-to-end from the Pico W to Caddy.
```
