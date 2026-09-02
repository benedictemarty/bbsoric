#!/bin/bash
# Snapshot LOCAL de la source catalogue tachibana.eu, pour alimenter
# scripts/gen-catalogue-tachibana.py sans dépendre en direct de l'hôte tachibana.
#
# Récupère (via ssh + Proxmox `pct exec`) :
#   1. la base   /var/lib/tachibana/tachibana.sqlite   -> $OUT/tachibana.sqlite
#   2. un MIROIR partiel du volume /srv/oriclib limité aux seuls fichiers
#      téléchargeables (.tap/.dsk/.rom/.ort) référencés par un logiciel publié et
#      ≤ plafond (défaut 65535 o) -> $OUT/oriclib/ (où /lib/X == $OUT/oriclib/X).
#
# Le miroir ne contient QUE l'utile (~20 Mo) au lieu des ~290 Mo du volume complet.
#
# Config (env) :
#   TACHI_SSH     hôte ssh Proxmox         (défaut kiranerys)
#   TACHI_CT      CT LXC tachibana         (défaut 716)
#   TACHI_DB      base sur le CT           (défaut /var/lib/tachibana/tachibana.sqlite)
#   TACHI_ORICLIB volume médias sur le CT  (défaut /srv/oriclib)
#   OUT           répertoire local         (défaut /tmp/tachibana)
#   MAX           plafond octets           (défaut 65535)
#
# Usage : scripts/pull-tachibana.sh
set -euo pipefail

SSH_HOST="${TACHI_SSH:-kiranerys}"
CT="${TACHI_CT:-716}"
REMOTE_DB="${TACHI_DB:-/var/lib/tachibana/tachibana.sqlite}"
REMOTE_LIB="${TACHI_ORICLIB:-/srv/oriclib}"
OUT="${OUT:-/tmp/tachibana}"
MAX="${MAX:-65535}"

CTX="ssh $SSH_HOST pct exec $CT --"
mkdir -p "$OUT/oriclib"

echo "=== 1. Base tachibana.sqlite (+ WAL) -> $OUT ==="
for f in tachibana.sqlite tachibana.sqlite-wal tachibana.sqlite-shm; do
    $CTX cat "$(dirname "$REMOTE_DB")/$f" > "$OUT/$f" 2>/dev/null || echo "  (pas de $f)"
done
[ -s "$OUT/tachibana.sqlite" ] || { echo "échec : base vide" >&2; exit 1; }

echo "=== 2. Inventaire des fichiers téléchargeables (tailles) ==="
# %P = chemin relatif à $REMOTE_LIB ; l'URL /lib/<P> == miroir $OUT/oriclib/<P>.
$CTX bash -lc "cd '$REMOTE_LIB' && find . -iregex '.*\.\(tap\|dsk\|rom\|ort\)' -type f -printf '%s\t%P\n'" \
    > "$OUT/inventory.tsv"
echo "  fichiers inventoriés : $(wc -l < "$OUT/inventory.tsv")"

echo "=== 3. Sélection des fichiers réellement utiles (référencés, ≤ $MAX o) ==="
OUT="$OUT" MAX="$MAX" python3 - <<'PY'
import os, re, sqlite3
out = os.environ["OUT"]; MAX = int(os.environ["MAX"])
sz = {}
for line in open(os.path.join(out, "inventory.tsv")):
    line = line.rstrip("\n")
    if "\t" not in line:
        continue
    s, rel = line.split("\t", 1)
    try:
        sz["/lib/" + rel] = int(s)
    except ValueError:
        pass
con = sqlite3.connect(os.path.join(out, "tachibana.sqlite"))
href = re.compile(r'href="(/lib/[^"?]+\.(?:tap|dsk|rom|ort))"', re.I)
EXT = [".tap", ".ort", ".rom", ".dsk"]
need = set()
for (body,) in con.execute("SELECT body FROM items WHERE category='Logiciel' AND published=1"):
    cands = []
    for u in href.findall(body or ""):
        ext = "." + u.rsplit(".", 1)[1].lower()
        if u in sz and sz[u] > 0 and ext in EXT:
            cands.append((EXT.index(ext), sz[u], u))
    cands.sort()
    chosen = next((u for _, s, u in cands if s <= MAX), None)
    if chosen:
        need.add(chosen[len("/lib/"):])   # relatif à $REMOTE_LIB / miroir
with open(os.path.join(out, "pull-rel.txt"), "w") as f:
    f.write("\n".join(sorted(need)) + "\n")
print("  fichiers à rapatrier : %d (%.1f Mo)"
      % (len(need), sum(sz["/lib/" + r] for r in need) / 1e6))
PY

echo "=== 4. Rapatriement (tar over ssh) -> $OUT/oriclib ==="
$CTX bash -lc "cat > /tmp/pull-rel.txt" < "$OUT/pull-rel.txt"
$CTX bash -lc "cd '$REMOTE_LIB' && tar czf - -T /tmp/pull-rel.txt 2>/dev/null" \
    | tar xzf - -C "$OUT/oriclib"
echo "  miroir : $(find "$OUT/oriclib" -type f | wc -l) fichiers, $(du -sh "$OUT/oriclib" | cut -f1)"

echo "=== Snapshot prêt ==="
echo "  base   : $OUT/tachibana.sqlite"
echo "  miroir : $OUT/oriclib   (utiliser --oriclib $OUT/oriclib)"
