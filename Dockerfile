# syntax=docker/dockerfile:1
# ---------------------------------------------------------------------------
# BBS Oric — image conteneurisée du serveur (bbsd).
# Build multi-stage : compilation Go statique → image alpine minimale.
# Le serveur n'a aucune dépendance externe (stdlib uniquement).
# ---------------------------------------------------------------------------

# --- Étape 1 : compilation ---
FROM golang:1.26-alpine AS build
WORKDIR /src
# go.mod seul d'abord (cache de couche ; aucun module externe → pas de go.sum).
COPY go.mod ./
COPY . .
# Binaire statique, sans table de symboles ni chemins absolus (reproductible).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' \
    -o /out/bbsoric ./server/cmd/bbsd

# --- Étape 2 : exécution ---
FROM alpine:3.20
# wget pour le HEALTHCHECK, ca-certificates pour l'écoute TLS optionnelle.
RUN apk add --no-cache wget ca-certificates \
    && adduser -D -H -u 10001 bbsoric \
    && mkdir -p /var/lib/bbsoric /etc/bbsoric \
    && chown -R bbsoric /var/lib/bbsoric

COPY --from=build /out/bbsoric /usr/local/bin/bbsoric
# Contenu par défaut (surchargeable par un volume sur /etc/bbsoric/site.json).
COPY content/site.json /etc/bbsoric/site.json

USER bbsoric
# 6502 = telnet (public) ; 6992 = TLS (si -tls-addr activé).
EXPOSE 6502 6992

# Sonde de vivacité via l'endpoint local /healthz (cf. -metrics-addr ci-dessous).
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:6510/healthz | grep -q ok || exit 1

ENTRYPOINT ["/usr/local/bin/bbsoric"]
# Le BBS écoute en clair sur 6502 ; la supervision reste LOCALE au conteneur.
CMD ["-addr", "0.0.0.0:6502", \
     "-content", "/etc/bbsoric/site.json", \
     "-users", "/var/lib/bbsoric/users.json", \
     "-max-conns", "50", "-max-conns-per-ip", "3", "-idle", "5m", \
     "-metrics-addr", "127.0.0.1:6510"]
