# Actualités du BBS depuis les articles de tachibana.eu

Les **actualités** du BBS (applet `news`, `docs`… flag serveur `-news`) peuvent être alimentées
depuis les **articles** du site **tachibana.eu** (`tachibana.sqlite`, table `articles`).

> État au 03/09/2026 : tachibana.eu **n'a encore aucun article publié** — le pont est en place et
> l'« Actualites » du BBS s'affiche « Aucune annonce pour le moment » ; elle se remplira dès qu'un
> article `fr` publié apparaîtra sur tachibana.eu, au prochain `deploy-news.sh`.

## Mapping

Le store news du BBS persiste un tableau JSON d'`Item` `{title, body, author, at}` (ordre
chronologique, le plus récent en dernier). Chaque article `published=1` de la langue choisie
(défaut `fr`) devient une annonce :

| BBS `Item` | Article tachibana | Traitement |
|---|---|---|
| `title` | `title` | balises retirées, **accents translittérés** (é→e), ≤ 38 car. |
| `body` | `summary` sinon texte du `body` HTML | HTML retiré, translittéré, ≤ 400 car. |
| `author` | — | `tachibana.eu` |
| `at` | `created_at` | normalisé RFC3339 (article ignoré si date inexploitable) |

L'affichage Oric étant en **ASCII pur** (`oascii.SanitizeText` écarte le non-ASCII), le générateur
**translittère** les accents au lieu de les perdre.

## Générer / déployer

```bash
# génération locale (lecture seule de la base) :
python3 scripts/gen-news-tachibana.py --db /tmp/tachibana/tachibana.sqlite [--lang fr] --out news.json

# déploiement en prod (récupère le news.json distant, fusionne, dépose, redémarre) :
scripts/deploy-news.sh [--dry-run] [--pull]
```

`deploy-news.sh` **préserve les annonces d'autres auteurs** (celles postées par un sysop dans le
BBS) et ne remplace que les entrées `tachibana.eu`. Le store news étant chargé **au démarrage** (pas
relu à chaud comme le `site.json`), le dépôt s'accompagne d'un **redémarrage** du service.

Le service de production active `-news /var/lib/bbsoric/news.json` (cf. `deploy/bbsoric.service`).
L'entrée « Actualites » existe déjà dans le menu **Services** du contenu.

## Tests

`scripts/test-gen-news-tachibana.py` (17 cas) : translittération, retrait HTML, bornes titre/corps,
filtre `published`/langue, date RFC3339, articles ignorés (date KO / corps vide), fusion préservant
les autres auteurs.
