package preview

import (
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
)

func testSite() *content.Site {
	return &content.Site{
		Start: "main",
		Pages: map[string]*content.Page{
			"main": {Title: "MENU", Entries: []content.Entry{
				{Key: "1", Label: "Infos", Target: "info"},
			}},
			"info": {Title: "INFOS", Lines: []content.Line{
				{Text: "Bonjour", Style: content.Style{Ink: "cyan"}},
			}},
			"login": {Title: "AUTH", Applet: "login", Next: "main"},
		},
	}
}

func TestRenderMenu(t *testing.T) {
	out, err := RenderHTML(testSite(), "main")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, "MENU") || !strings.Contains(out, "Infos") || !strings.Contains(out, "Votre choix") {
		t.Errorf("rendu menu incomplet:\n%s", out)
	}
}

func TestRenderPageEscapesAndColors(t *testing.T) {
	site := testSite()
	site.Pages["info"].Lines[0].Text = "<b>x</b>"
	out, err := RenderHTML(site, "info")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if strings.Contains(out, "<b>x</b>") {
		t.Errorf("le texte doit être échappé HTML")
	}
	if !strings.Contains(out, "&lt;b&gt;") {
		t.Errorf("échappement attendu, reçu:\n%s", out)
	}
	if !strings.Contains(out, "Appuyez sur une touche") {
		t.Errorf("invite de page attendue")
	}
}

func TestRenderApplet(t *testing.T) {
	out, err := RenderHTML(testSite(), "login")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, "[applet: login]") {
		t.Errorf("marqueur applet attendu, reçu:\n%s", out)
	}
}

func TestRenderLineAttributes(t *testing.T) {
	site := &content.Site{Start: "p", Pages: map[string]*content.Page{
		"p": {Title: "P", Lines: []content.Line{
			{Text: "X", Style: content.Style{Ink: "white", Paper: "red", Blink: true, DoubleHeight: true}},
		}},
	}}
	out, err := RenderHTML(site, "p")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, "background:") {
		t.Errorf("fond (paper) manquant:\n%s", out)
	}
	if !strings.Contains(out, `class="blink"`) {
		t.Errorf("clignotement manquant:\n%s", out)
	}
	if !strings.Contains(out, "font-size") {
		t.Errorf("double hauteur manquante:\n%s", out)
	}
}

func TestRenderSegmentsInverseAlt(t *testing.T) {
	site := &content.Site{Start: "p", Pages: map[string]*content.Page{
		"p": {Title: "P", Lines: []content.Line{
			{Segments: []content.Span{
				{Text: "Score ", Style: content.Style{Ink: "white"}},
				{Text: "42", Style: content.Style{Ink: "yellow"}},
				{Text: " INV ", Style: content.Style{Ink: "white", Paper: "red", Inverse: true}},
				{Text: " gfx ", Style: content.Style{AltCharset: true}},
			}},
		}},
	}}
	out, err := RenderHTML(site, "p")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(out, ">Score <") || !strings.Contains(out, ">42<") {
		t.Errorf("segments non rendus:\n%s", out)
	}
	// inverse : l'encre devient le fond (rouge en color:)
	if !strings.Contains(out, "color:"+css(1 /*Red*/)) {
		t.Errorf("inverse (échange encre/fond) attendu:\n%s", out)
	}
	// charset alternatif : classe d'approximation
	if !strings.Contains(out, `class="alt"`) {
		t.Errorf("marqueur semi-graphiques attendu:\n%s", out)
	}
}

func TestRenderMissingPage(t *testing.T) {
	if _, err := RenderHTML(testSite(), "absent"); err == nil {
		t.Errorf("une page absente doit renvoyer une erreur")
	}
}
