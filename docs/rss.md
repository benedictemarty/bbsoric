# Actualités du BBS depuis un flux RSS / Atom

Les **actualités** du BBS (applet `news`, flag serveur `-news`) peuvent être alimentées depuis
n'importe quel **flux RSS 2.0 ou Atom** externe, en plus (ou à la place) du pont tachibana.eu
(cf. `docs/news-tachibana.md`). C'est le besoin **Sprint 7 #5 « RSS→OASCII news »**.

Deux briques, sans nouvelle dépendance (stdlib `encoding/xml` uniquement) :

- **`internal/rss`** — parseur défensif RSS 2.0 / Atom (`rss.Parse(io.Reader) (Feed, error)`).
  Aucun accès réseau : il parse un flux déjà récupéré. HTML retiré, entités décodées, dates
  tolérantes (RFC1123/RFC3339 et variantes), entrées triées de la plus récente à la plus ancienne.
- **`server/cmd/rssnews`** — outil CLI : récupère le flux (HTTP borné + timeout, ou fichier local),
  le convertit en annonces `news.Item`, et **fusionne** dans un `news.json` existant.

> Le daemon `bbsd` ne fait **aucun** accès réseau sortant (serveur public en clair, cf. `CLAUDE.md`) :
> c'est `rssnews`, déployable via un timer systemd (`deploy/rssnews/`), qui va chercher le flux —
> hors du daemon, comme la synchro tachibana.

## Mapping

Le store news persiste un tableau JSON d'`Item` `{title, body, author, at}` (ordre chronologique,
le plus récent en dernier). Chaque entrée du flux devient une annonce :

| BBS `Item` | Entrée du flux | Traitement |
|---|---|---|
| `title` | `<title>` | balises retirées, **accents translittérés** (é→e, œ→oe), ≤ 38 car. (défaut « Annonce ») |
| `body` | `content:encoded` / `content` / `description` / `summary` (1er non vide) | HTML retiré, translittéré, ≤ 400 car. |
| `author` | — | `-author`, sinon titre du flux, sinon `rss` (≤ 16 car.) |
| `at` | `pubDate` / `published` / `updated` | normalisé UTC ; **entrée ignorée si date inexploitable ou corps vide** (on n'invente pas) |

L'affichage Oric étant en **ASCII pur** (`oascii.SanitizeText` écarte le non-ASCII), `rssnews`
**translittère** les accents et la ponctuation typographique (« » … ' ") avant de nettoyer, pour
ne pas *perdre* les lettres accentuées.

Seules les `-max` entrées les plus récentes sont importées (défaut 20). La fusion `--merge-into`
préserve les annonces d'un **autre** `author` (sysop, tachibana.eu…) et remplace celles du même
`author` par l'import frais — même règle que le pont tachibana.

## Générer / tester

```bash
make rssnews                                   # compile ./rssnews

# aperçu depuis un flux téléchargé (aucune écriture) :
./rssnews -file flux.xml -author exemple.org -out -

# récupération réseau + fusion réelle dans un news.json :
./rssnews -feed https://exemple.org/rss.xml -author exemple.org \
          --merge-into news.json --out news.json
```

Options : `-feed` (URL http/https) **ou** `-file` (fichier local), `-author`, `-max` (défaut 20),
`-timeout` (défaut 15 s), `-max-bytes` (défaut 4 Mio), `-merge-into`, `-out` (`-` = stdout).

## Déployer (synchro périodique)

Scaffolding systemd dans **`deploy/rssnews/`** (script + `.service` + `.timer` + README), calqué
sur `deploy/newssync`. **Non installé** : aucune URL de flux n'est configurée par défaut (on
n'invente pas de source) — renseigner `FEED_URL` dans l'unité avant d'activer. Voir
`deploy/rssnews/README.md`.

## Tests

- `internal/rss/rss_test.go` — RSS + Atom, ordre, entrées vides, formats de date, rejet du non-XML.
- `server/cmd/rssnews/main_test.go` — conversion (ignore non datées/vides), bornes, garde des N plus
  récents, fusion préservant les autres auteurs, déaccentuage.
