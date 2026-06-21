# Terminaison TLS du BBS par Caddy (Let's Encrypt)

Le TLS du BBS (`pavi.3617.fr:6992`) est terminé par **Caddy** (cert Let's Encrypt
auto-renouvelé), pas par `bbsd`. Caddy déchiffre et relaie le flux telnet en clair
vers `bbsd` sur le réseau interne.

## Chaîne réseau

```
Oric + Pico W ──TLS──► MikroTik (.4:6992)  ──►  Caddy CT130 (.3:6992)
  (ATDT#pavi.3617.fr:6992)   dst-nat              │ termine TLS (cert Let's Encrypt
                                                  │ pour pavi.3617.fr, module layer4)
                                                  ▼
                                          bbsd telnet (.2:6502)  ── en clair (LAN interne)
```

## Côté Caddy (CT 130 « caddy-meteolib », atlas)

Caddy standard est HTTP only ; la terminaison TLS d'un port TCP brut nécessite le
module **`caddy-l4`** (layer4). Binaire reconstruit avec ce module
(`https://caddyserver.com/api/download?...&p=github.com/mholt/caddy-l4`, v2.11.4).

Ajouts au `Caddyfile` (backup : `/root/Caddyfile.bak-pre-l4`, binaire :
`/root/caddy.bak-pre-l4`) :

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

# cert Let's Encrypt pour pavi.3617.fr (+ page info HTTPS)
pavi.3617.fr {
	encode zstd gzip
	respond "BBS Oric - connexion telnet TLS sur le port 6992 (ATDT#pavi.3617.fr:6992)" 200
}
```

Le bloc `layer4` écoute en `:6992`, matche les connexions TLS (`@bbs tls`), les
termine (`tls`, cert géré automatiquement par SNI) et proxifie vers `bbsd` en clair.
Le site `pavi.3617.fr` déclenche l'obtention du certificat (challenge ACME sur `:443`,
déjà routé vers Caddy).

## Côté MikroTik

Règle dst-nat (numéro 64) : `:6992` redirigé vers Caddy au lieu de bbsd.
```
chain=dstnat dst-address=203.0.113.4 in-interface=ether1 protocol=tcp
dst-port=6992 action=dst-nat to-addresses=192.168.1.3 to-ports=6992
;;; telenet - PAVI Oric TLS via Caddy (.4:6992 -> .3:6992)
```

## Notes

- **Cert vérifiable** : le Pico W peut activer `AT$CV1` avec le CA racine Let's Encrypt
  (ISRG Root X1) chargé par `AT$CA` — le cert présenté est désormais de confiance.
- **bbsd `-tls-addr 6992`** devient redondant (Caddy gère le TLS public). Le listener
  TLS interne de bbsd sur `.2:6992` n'est plus dans le chemin ; il peut être retiré de
  l'unité systemd (`deploy/bbsoric.service`) lors d'un prochain déploiement. `bbsd` ne
  sert plus que le **telnet en clair sur `.2:6502`** vers Caddy.
- Hop interne Caddy→bbsd en **clair** sur le LAN (réseau de confiance) ; le chiffrement
  bout-en-bout va du Pico W à Caddy.
```
