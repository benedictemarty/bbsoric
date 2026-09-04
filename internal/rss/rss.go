// Package rss lit un flux d'actualités RSS 2.0 ou Atom et en extrait des entrées
// simples, prêtes à devenir des annonces du BBS (server/internal/news).
//
// Lecture défensive (le flux vient d'Internet, en clair) : parseur stdlib
// (encoding/xml, pas de dépendance), taille d'entrée bornée par l'appelant via
// io.LimitReader, HTML retiré du corps, entités décodées, dates tolérantes.
// Le package ne fait AUCun accès réseau : il parse un io.Reader. La récupération
// HTTP (timeout, plafond de taille) est du ressort de l'appelant (cmd rssnews).
package rss

import (
	"encoding/xml"
	"html"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Item est une entrée de flux, normalisée et indépendante du dialecte (RSS/Atom).
type Item struct {
	Title string    // titre nettoyé (HTML retiré)
	Body  string    // description / résumé, HTML retiré
	Link  string    // URL de l'article (informative)
	At    time.Time // date de publication (zéro si absente/illisible)
}

// Feed est un flux parsé : titre de la source + entrées, de la plus récente à
// la plus ancienne.
type Feed struct {
	Title string
	Items []Item
}

// Structures XML couvrant à la fois RSS 2.0 (<rss><channel><item>) et Atom
// (<feed><entry>). encoding/xml ignore les champs absents : un même document
// remplit soit la branche RSS, soit la branche Atom.
type xmlDoc struct {
	XMLName xml.Name
	// RSS 2.0
	Channel struct {
		Title string    `xml:"title"`
		Items []xmlItem `xml:"item"`
	} `xml:"channel"`
	// Atom
	Title   string    `xml:"title"`
	Entries []xmlItem `xml:"entry"`
}

// xmlItem regroupe les champs d'un <item> RSS et d'une <entry> Atom : les noms
// se chevauchent sans conflit (title/link diffèrent, on gère link à part).
type xmlItem struct {
	Title       string     `xml:"title"`
	Description string     `xml:"description"` // RSS
	Summary     string     `xml:"summary"`     // Atom
	Content     string     `xml:"content"`     // Atom (souvent le corps complet)
	Encoded     string     `xml:"encoded"`     // RSS <content:encoded>
	PubDate     string `xml:"pubDate"`   // RSS
	Updated     string `xml:"updated"`   // Atom
	Published   string `xml:"published"` // Atom
	// <link> : RSS met l'URL en texte, Atom en attribut href — on capte les deux.
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Text string `xml:",chardata"`
	} `xml:"link"`
}

var (
	tagRE   = regexp.MustCompile(`(?s)<[^>]*>`)  // balises HTML/XML
	spaceRE = regexp.MustCompile(`\s+`)          // espaces multiples / retours ligne
	punctRE = regexp.MustCompile(` +([.,;:!?])`) // espace parasite avant ponctuation
)

// Formats de date acceptés, du plus courant au plus rare (RSS puis Atom puis
// variantes tolérées vues dans la nature).
var dateLayouts = []string{
	time.RFC1123Z,                     // RSS standard : Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC1123,                      // Mon, 02 Jan 2006 15:04:05 MST
	time.RFC3339,                      // Atom : 2006-01-02T15:04:05Z07:00
	"2006-01-02T15:04:05Z",            // Atom sans fuseau explicite
	"Mon, 2 Jan 2006 15:04:05 -0700",  // jour non zéro-paddé
	"2006-01-02 15:04:05",             // variante lâche
	"2006-01-02",                      // date seule
}

// Parse lit un flux RSS 2.0 ou Atom depuis r et renvoie le Feed normalisé.
// Les entrées sont triées de la plus récente à la plus ancienne (les entrées
// sans date valable sont conservées et rejetées en fin de liste).
func Parse(r io.Reader) (Feed, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false             // flux réels : entités/encodages parfois laxistes
	dec.CharsetReader = charsetPassthrough

	var doc xmlDoc
	if err := dec.Decode(&doc); err != nil {
		return Feed{}, err
	}

	f := Feed{Title: cleanText(firstNonEmpty(doc.Channel.Title, doc.Title))}

	raw := doc.Channel.Items
	if len(raw) == 0 {
		raw = doc.Entries // document Atom
	}
	for _, x := range raw {
		it := Item{
			Title: cleanText(x.Title),
			Body:  cleanText(firstNonEmpty(x.Encoded, x.Content, x.Description, x.Summary)),
			Link:  itemLink(x),
			At:    parseDate(firstNonEmpty(x.PubDate, x.Published, x.Updated)),
		}
		if it.Title == "" && it.Body == "" {
			continue // entrée vide → ignorée
		}
		f.Items = append(f.Items, it)
	}

	// Antéchronologique ; date zéro (inconnue) reléguée en fin.
	sort.SliceStable(f.Items, func(i, j int) bool {
		a, b := f.Items[i].At, f.Items[j].At
		if a.IsZero() != b.IsZero() {
			return !a.IsZero() // les datées d'abord
		}
		return a.After(b)
	})
	return f, nil
}

// itemLink choisit l'URL : texte de <link> (RSS) sinon href (Atom, rel="alternate"
// prioritaire, à défaut le premier href rencontré).
func itemLink(x xmlItem) string {
	var first string
	for _, l := range x.Links {
		if s := strings.TrimSpace(l.Text); s != "" {
			return s // RSS : URL en contenu textuel
		}
		if first == "" && l.Href != "" {
			first = l.Href
		}
		if l.Href != "" && (l.Rel == "" || l.Rel == "alternate") {
			return l.Href
		}
	}
	return first
}

// cleanText retire le HTML, décode les entités et compacte les espaces.
func cleanText(s string) string {
	s = tagRE.ReplaceAllString(s, " ") // espace (jamais coller deux mots)
	s = html.UnescapeString(s)
	s = spaceRE.ReplaceAllString(s, " ")
	s = punctRE.ReplaceAllString(s, "$1") // retire l'espace laissé devant « . , ; »
	return strings.TrimSpace(s)
}

// parseDate tente les formats connus ; renvoie le zéro time si aucun ne colle.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// charsetPassthrough laisse passer les flux non-UTF-8 (latin-1 fréquent) sans
// convertir : encoding/xml, en mode non strict, lit alors les octets bruts —
// la sanitisation OASCII en aval ne garde de toute façon que l'ASCII imprimable.
func charsetPassthrough(_ string, input io.Reader) (io.Reader, error) {
	return input, nil
}
