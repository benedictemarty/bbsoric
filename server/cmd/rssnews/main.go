// Commande rssnews : importe un flux RSS 2.0 / Atom dans les ACTUALITÉS du BBS.
//
// Pendant Go du pont tachibana (scripts/gen-news-tachibana.py) mais pour un flux
// RSS quelconque : récupère le flux (HTTP borné + timeout, ou fichier local),
// le convertit en annonces news.Item (titre/corps nettoyés ASCII et bornés) et
// FUSIONNE dans un news.json existant en préservant les annonces d'autres auteurs
// (mêmes règles que le pont tachibana : les entrées de --author sont remplacées
// par l'import frais, les autres conservées, tri chronologique croissant).
//
// Serveur public : lecture défensive. Aucun accès réseau côté bbsd — c'est cet
// outil (déployable via un timer systemd, cf. deploy/newssync) qui va chercher le
// flux, hors du daemon.
//
//	rssnews -feed https://exemple.org/rss.xml -author exemple.org \
//	        -merge-into /var/lib/bbsoric/news.json -out /var/lib/bbsoric/news.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/rss"
	"github.com/benedictemarty/bbsoric/server/internal/news"
)

func main() {
	feed := flag.String("feed", "", "URL du flux RSS/Atom (http/https)")
	file := flag.String("file", "", "lire le flux depuis un fichier local (au lieu de -feed)")
	author := flag.String("author", "", "auteur des annonces importées (défaut : titre du flux)")
	out := flag.String("out", "-", "fichier de sortie (- = stdout)")
	mergeInto := flag.String("merge-into", "", "news.json existant : préserve les autres auteurs, remplace ceux de -author")
	maxItems := flag.Int("max", 20, "nombre maximum d'articles importés (les plus récents)")
	timeout := flag.Duration("timeout", 15*time.Second, "délai maximum de récupération HTTP")
	maxBytes := flag.Int64("max-bytes", 4<<20, "taille maximale lue du flux (octets)")
	flag.Parse()

	if (*feed == "") == (*file == "") {
		fmt.Fprintln(os.Stderr, "erreur : fournir exactement l'un de -feed ou -file")
		os.Exit(2)
	}

	r, closeFn, err := openFeed(*feed, *file, *timeout, *maxBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur récupération flux :", err)
		os.Exit(1)
	}
	defer closeFn()

	parsed, err := rss.Parse(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur parsing flux :", err)
		os.Exit(1)
	}

	who := sanitize(*author, 16)
	if who == "" {
		who = sanitize(parsed.Title, 16)
	}
	if who == "" {
		who = "rss"
	}

	items, skipped := toItems(parsed.Items, who, *maxItems)

	if *mergeInto != "" {
		items = mergeFile(*mergeInto, who, items)
	}
	if len(items) > news.MaxItems {
		items = items[len(items)-news.MaxItems:]
	}

	if err := writeItems(*out, items); err != nil {
		fmt.Fprintln(os.Stderr, "erreur écriture :", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Flux RSS (%s) : %d annonce(s) importee(s), %d ignoree(s) -> %s\n",
		who, len(parsed.Items)-skipped, skipped, *out)
}

// openFeed ouvre le flux depuis une URL (HTTP borné + timeout) ou un fichier.
// Renvoie un io.Reader et une fonction de fermeture.
func openFeed(feedURL, path string, timeout time.Duration, maxBytes int64) (io.Reader, func(), error) {
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, func() {}, err
		}
		return io.LimitReader(f, maxBytes), func() { f.Close() }, nil
	}
	if !strings.HasPrefix(feedURL, "http://") && !strings.HasPrefix(feedURL, "https://") {
		return nil, func() {}, fmt.Errorf("URL non http(s) : %q", feedURL)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(feedURL)
	if err != nil {
		return nil, func() {}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, func() {}, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.LimitReader(resp.Body, maxBytes), func() { resp.Body.Close() }, nil
}

// toItems convertit les entrées de flux en annonces, nettoyées et bornées, en
// ne gardant que les n plus récentes datées (ordre chronologique croissant, le
// plus récent en dernier — comme le store). Les entrées sans date ou au corps
// vide sont ignorées (on n'invente pas de date).
func toItems(entries []rss.Item, author string, n int) (items []news.Item, skipped int) {
	for _, e := range entries {
		body := sanitize(e.Body, news.MaxBody)
		if e.At.IsZero() || body == "" {
			skipped++
			continue
		}
		title := sanitize(e.Title, news.MaxTitle)
		if title == "" {
			title = "Annonce"
		}
		items = append(items, news.Item{Title: title, Body: body, Author: author, At: e.At.UTC()})
	}
	// entries est antéchronologique ; on garde les n premières (plus récentes)…
	if n > 0 && len(items) > n {
		items = items[:n]
	}
	// …puis on repasse en chronologique croissant pour le store.
	sort.SliceStable(items, func(i, j int) bool { return items[i].At.Before(items[j].At) })
	return items, skipped
}

// mergeFile fusionne les annonces importées avec un news.json existant : les
// annonces d'un auteur différent sont conservées, celles de author remplacées.
func mergeFile(path, author string, fresh []news.Item) []news.Item {
	data, err := os.ReadFile(path)
	if err != nil {
		return fresh // fichier absent/illisible : on repart de l'import seul
	}
	var existing []news.Item
	if err := json.Unmarshal(data, &existing); err != nil {
		return fresh
	}
	merged := fresh
	for _, it := range existing {
		if it.Author != author {
			merged = append(merged, it)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].At.Before(merged[j].At) })
	return merged
}

func writeItems(out string, items []news.Item) error {
	if items == nil {
		items = []news.Item{}
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if out == "-" {
		_, err := os.Stdout.Write(append(data, '\n'))
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o644)
}

// sanitize : déaccentue (é→e, œ→oe…) puis nettoie en ASCII imprimable (oascii)
// et borne — comme le store news, mais en translittérant d'abord pour ne pas
// PERDRE les lettres accentuées (SanitizeText, seul, supprimerait « é »).
func sanitize(s string, max int) string {
	s = oascii.SanitizeText(deaccent(s))
	if len(s) > max {
		s = strings.TrimSpace(s[:max])
	}
	return strings.TrimSpace(s)
}

// deaccent translittère les lettres latines accentuées et ligatures courantes
// vers l'ASCII (le terminal Oric n'a pas d'accents). Table volontairement
// restreinte aux caractères réellement rencontrés dans les flux fr/en/de/es.
func deaccent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if repl, ok := deaccentMap[r]; ok {
			b.WriteString(repl)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var deaccentMap = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a",
	'À': "A", 'Á': "A", 'Â': "A", 'Ä': "A", 'Ã': "A", 'Å': "A",
	'ç': "c", 'Ç': "C",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'È': "E", 'É': "E", 'Ê': "E", 'Ë': "E",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i",
	'Ì': "I", 'Í': "I", 'Î': "I", 'Ï': "I",
	'ñ': "n", 'Ñ': "N",
	'ò': "o", 'ó': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o",
	'Ò': "O", 'Ó': "O", 'Ô': "O", 'Ö': "O", 'Õ': "O", 'Ø': "O",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u",
	'Ù': "U", 'Ú': "U", 'Û': "U", 'Ü': "U",
	'ý': "y", 'ÿ': "y", 'Ý': "Y",
	'œ': "oe", 'Œ': "OE", 'æ': "ae", 'Æ': "AE", 'ß': "ss",
	'’': "'", '‘': "'", '“': "\"", '”': "\"", '–': "-", '—': "-",
	'…': "...", '«': "\"", '»': "\"", ' ': " ",
}
