package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benedictemarty/bbsoric/internal/rss"
	"github.com/benedictemarty/bbsoric/server/internal/news"
)

func d(y, m, day int) time.Time {
	return time.Date(y, time.Month(m), day, 12, 0, 0, 0, time.UTC)
}

func TestToItemsSkipsUndatedAndEmpty(t *testing.T) {
	entries := []rss.Item{
		{Title: "Recent", Body: "corps ok", At: d(2026, 9, 3)},
		{Title: "Sans date", Body: "corps", At: time.Time{}},   // ignoré : pas de date
		{Title: "Vide", Body: "", At: d(2026, 9, 2)},           // ignoré : corps vide
		{Title: "Ancien", Body: "autre corps", At: d(2026, 9, 1)},
	}
	items, skipped := toItems(entries, "src", 0)
	if skipped != 2 {
		t.Errorf("skipped = %d, veut 2", skipped)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, veut 2", len(items))
	}
	// Ordre chronologique croissant (le plus récent en dernier).
	if !items[0].At.Before(items[1].At) {
		t.Errorf("ordre non chronologique : %v puis %v", items[0].At, items[1].At)
	}
	if items[1].Title != "Recent" || items[1].Author != "src" {
		t.Errorf("dernier item = %+v", items[1])
	}
}

func TestToItemsBoundsAndDefaultTitle(t *testing.T) {
	long := ""
	for i := 0; i < 100; i++ {
		long += "x"
	}
	entries := []rss.Item{{Title: "", Body: long, At: d(2026, 9, 3)}}
	items, _ := toItems(entries, "src", 0)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Title != "Annonce" {
		t.Errorf("titre vide doit devenir Annonce, a %q", items[0].Title)
	}
	if len(items[0].Body) > news.MaxBody {
		t.Errorf("corps non borné : %d > %d", len(items[0].Body), news.MaxBody)
	}
}

func TestToItemsKeepsMostRecentN(t *testing.T) {
	entries := []rss.Item{ // antéchronologique en entrée
		{Title: "A", Body: "x", At: d(2026, 9, 5)},
		{Title: "B", Body: "x", At: d(2026, 9, 4)},
		{Title: "C", Body: "x", At: d(2026, 9, 3)},
	}
	items, _ := toItems(entries, "src", 2)
	if len(items) != 2 {
		t.Fatalf("items = %d, veut 2", len(items))
	}
	// Doit garder A et B (les plus récents), en ordre croissant B puis A.
	if items[0].Title != "B" || items[1].Title != "A" {
		t.Errorf("garde = [%q,%q], veut [B,A]", items[0].Title, items[1].Title)
	}
}

func TestMergeFilePreservesOtherAuthors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "news.json")
	existing := []news.Item{
		{Title: "Sysop", Body: "manuel", Author: "sysop", At: d(2026, 9, 1)},
		{Title: "Vieux RSS", Body: "ancien", Author: "src", At: d(2026, 8, 1)},
	}
	data, _ := json.Marshal(existing)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	fresh := []news.Item{{Title: "Neuf", Body: "frais", Author: "src", At: d(2026, 9, 3)}}

	merged := mergeFile(path, "src", fresh)
	if len(merged) != 2 {
		t.Fatalf("merged = %d, veut 2 (sysop conservé + neuf, vieux RSS remplacé)", len(merged))
	}
	var authors []string
	for _, it := range merged {
		authors = append(authors, it.Author+":"+it.Title)
	}
	// Ordre chronologique : sysop (09-01) puis Neuf (09-03) ; vieux RSS évincé.
	if merged[0].Title != "Sysop" || merged[1].Title != "Neuf" {
		t.Errorf("merge = %v", authors)
	}
}

func TestSanitizeDeaccents(t *testing.T) {
	cases := map[string]string{
		"Démo & prod":     "Demo & prod",
		"cœur naïf où":    "coeur naif ou",
		"Château fort":    "Chateau fort",
		"“guillemets”":    "\"guillemets\"",
		"c’est l’été":     "c'est l'ete",
	}
	for in, want := range cases {
		if got := sanitize(in, 400); got != want {
			t.Errorf("sanitize(%q) = %q, veut %q", in, got, want)
		}
	}
}

func TestMergeFileMissingFile(t *testing.T) {
	fresh := []news.Item{{Title: "x", Body: "y", Author: "src", At: d(2026, 9, 3)}}
	got := mergeFile(filepath.Join(t.TempDir(), "absent.json"), "src", fresh)
	if len(got) != 1 {
		t.Errorf("fichier absent : doit renvoyer l'import seul, a %d", len(got))
	}
}
