package rss

import (
	"strings"
	"testing"
	"time"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Tachibana</title>
    <link>https://tachibana.eu</link>
    <item>
      <title>Sortie du jeu &amp; du manuel</title>
      <link>https://tachibana.eu/a/2</link>
      <description>&lt;p&gt;Un &lt;b&gt;nouveau&lt;/b&gt; jeu   Oric.&lt;/p&gt;</description>
      <pubDate>Wed, 03 Sep 2026 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Ancienne annonce</title>
      <link>https://tachibana.eu/a/1</link>
      <description>Texte simple.</description>
      <pubDate>Mon, 01 Sep 2026 08:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

const sampleAtom = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Blog Oric</title>
  <entry>
    <title>Billet recent</title>
    <link rel="alternate" href="https://ex.org/recent"/>
    <summary>Resume &lt;i&gt;court&lt;/i&gt;.</summary>
    <published>2026-09-02T12:00:00Z</published>
  </entry>
  <entry>
    <title>Billet ancien</title>
    <link href="https://ex.org/vieux"/>
    <content>Contenu complet.</content>
    <updated>2026-08-15T09:30:00Z</updated>
  </entry>
</feed>`

func TestParseRSS(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleRSS))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Title != "Tachibana" {
		t.Errorf("titre flux = %q, veut Tachibana", f.Title)
	}
	if len(f.Items) != 2 {
		t.Fatalf("nb items = %d, veut 2", len(f.Items))
	}
	// Antéchronologique : la plus récente en tête.
	if f.Items[0].Title != "Sortie du jeu & du manuel" {
		t.Errorf("item[0].Title = %q", f.Items[0].Title)
	}
	// HTML retiré, entités décodées, espaces compactés.
	if got := f.Items[0].Body; got != "Un nouveau jeu Oric." {
		t.Errorf("body nettoyé = %q, veut %q", got, "Un nouveau jeu Oric.")
	}
	if f.Items[0].Link != "https://tachibana.eu/a/2" {
		t.Errorf("link = %q", f.Items[0].Link)
	}
	want := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	if !f.Items[0].At.Equal(want) {
		t.Errorf("date = %v, veut %v", f.Items[0].At, want)
	}
}

func TestParseAtom(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleAtom))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Title != "Blog Oric" {
		t.Errorf("titre flux = %q", f.Title)
	}
	if len(f.Items) != 2 {
		t.Fatalf("nb items = %d, veut 2", len(f.Items))
	}
	if f.Items[0].Title != "Billet recent" {
		t.Errorf("item[0].Title = %q", f.Items[0].Title)
	}
	if got := f.Items[0].Body; got != "Resume court." {
		t.Errorf("body = %q, veut %q", got, "Resume court.")
	}
	if f.Items[0].Link != "https://ex.org/recent" {
		t.Errorf("link = %q (doit préférer rel=alternate)", f.Items[0].Link)
	}
	if f.Items[1].Body != "Contenu complet." {
		t.Errorf("item[1] body (content Atom) = %q", f.Items[1].Body)
	}
}

func TestParseOrderingUndatedLast(t *testing.T) {
	const feed = `<rss version="2.0"><channel><title>T</title>
	  <item><title>Sans date</title><description>x</description></item>
	  <item><title>Datee</title><description>y</description><pubDate>Mon, 01 Sep 2026 08:00:00 +0000</pubDate></item>
	</channel></rss>`
	f, err := Parse(strings.NewReader(feed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Items[0].Title != "Datee" || f.Items[1].Title != "Sans date" {
		t.Errorf("ordre = [%q, %q], veut [Datee, Sans date]", f.Items[0].Title, f.Items[1].Title)
	}
}

func TestParseSkipsEmptyEntries(t *testing.T) {
	const feed = `<rss version="2.0"><channel><title>T</title>
	  <item></item>
	  <item><title>Vrai</title><description>ok</description></item>
	</channel></rss>`
	f, err := Parse(strings.NewReader(feed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(f.Items) != 1 || f.Items[0].Title != "Vrai" {
		t.Fatalf("items = %+v, veut une seule entrée Vrai", f.Items)
	}
}

func TestParseDateLayouts(t *testing.T) {
	cases := map[string]bool{
		"Wed, 03 Sep 2026 10:00:00 +0000": true,
		"2026-09-03T10:00:00Z":            true,
		"2026-09-03":                      true,
		"pas une date":                    false,
		"":                                false,
	}
	for in, ok := range cases {
		got := parseDate(in)
		if got.IsZero() == ok {
			t.Errorf("parseDate(%q) zéro=%v, veut valide=%v", in, got.IsZero(), ok)
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse(strings.NewReader("ceci n'est pas du xml <<<")); err == nil {
		t.Error("un flux non-XML devrait renvoyer une erreur")
	}
}
