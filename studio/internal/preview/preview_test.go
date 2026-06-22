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
			"main": {Title: "MENU", Type: "menu", Entries: []content.Entry{
				{Key: "1", Label: "Infos", Target: "info"},
			}},
			"info": {Title: "INFOS", Type: "page", Lines: []content.Line{
				{Text: "Bonjour", Ink: "cyan"},
			}},
			"login": {Title: "AUTH", Type: "applet", Applet: "login", Next: "main"},
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

func TestRenderMissingPage(t *testing.T) {
	if _, err := RenderHTML(testSite(), "absent"); err == nil {
		t.Errorf("une page absente doit renvoyer une erreur")
	}
}
