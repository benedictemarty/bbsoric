# Conteneurisation Docker — BBS Oric

> **Sprint 5 (optionnel).** Alternative au déploiement systemd (`deploy/`) :
> exécuter le BBS dans un conteneur. Le déploiement de production reste sur
> **systemd** (`make deploy`) ; Docker sert aux déploiements portables, aux
> environnements de test, ou à un hébergeur orienté conteneurs.

## Image

`Dockerfile` multi-stage :

1. **build** (`golang:1.26-alpine`) : compile un binaire **statique**
   (`CGO_ENABLED=0`, `-trimpath -ldflags='-s -w'`). Aucune dépendance externe
   (stdlib uniquement, pas de `go.sum`).
2. **runtime** (`alpine:3.20`) : binaire + `site.json` par défaut, utilisateur
   non-root `bbsoric` (uid 10001), `wget`/`ca-certificates` pour le healthcheck
   et le TLS optionnel.

Résultat : image **~18 Mo**.

### Healthcheck

L'image embarque un `HEALTHCHECK` qui interroge l'endpoint local `/healthz`
(`-metrics-addr 127.0.0.1:6510`, cf. `monitoring.md`). `docker ps` affiche alors
l'état `healthy`/`unhealthy`.

## Démarrage rapide

```console
# build + run (docker compose)
make docker-up            # = docker compose up -d --build
docker compose logs -f    # journaux
make docker-down          # arrêt

# ou directement
make docker-build
docker run -d --name bbsoric -p 6502:6502 -v bbsoric-state:/var/lib/bbsoric bbsoric:latest
```

Le BBS écoute alors sur le port **6502** de l'hôte. Test :

```console
nc 127.0.0.1 6502
```

## Configuration

| Aspect | Détail |
|--------|--------|
| **Port public** | `6502` (telnet). Mappé via `ports:` du compose. |
| **Comptes** | persistés dans le volume `bbsoric-state` (`/var/lib/bbsoric/users.json`). |
| **Contenu** | `site.json` par défaut intégré à l'image ; surchargeable en montant un fichier sur `/etc/bbsoric/site.json` (cf. ligne commentée du compose). |
| **Supervision** | `/healthz` + `/metrics` en **local au conteneur** (`127.0.0.1:6510`) — non publiés (pas dans `EXPOSE`/`ports`). |
| **Redémarrage** | `restart: unless-stopped`. |

### Activer TLS (port 6992)

Ajouter `-tls-addr 0.0.0.0:6992` à la commande et publier le port :

```yaml
    command: ["-addr","0.0.0.0:6502","-tls-addr","0.0.0.0:6992",
              "-content","/etc/bbsoric/site.json","-users","/var/lib/bbsoric/users.json",
              "-metrics-addr","127.0.0.1:6510"]
    ports:
      - "6502:6502"
      - "6992:6992"
```

Sans `-tls-cert`/`-tls-key`, un certificat auto-signé est généré au démarrage.

## Sécurité

- Le conteneur tourne en **non-root** (`USER bbsoric`).
- Seul le port **6502** (et éventuellement 6992) est exposé ; la supervision
  reste interne.
- Les garde-fous Internet du serveur (limite de connexions globale/par IP,
  timeout d'inactivité) s'appliquent comme en natif.

## Voir aussi

- `deploy/` + `vps-deploy.sh` — déploiement systemd de production.
- `docs/monitoring.md` — endpoint de supervision et sonde.
- `docs/architecture.md` §5 — exposition Internet.
