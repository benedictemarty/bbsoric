package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validSite = `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"Q","label":"Quitter","target":"__quit__"}]}}}`

func TestLoadProfilesExampleAndOverride(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "dev.conf.example"), []byte("LOCAL=1\nCONTENT_PATH=content/site.json\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "prod.conf.example"), []byte("HOST=ex\nUSER=root\nCONTENT_PATH=/etc/x\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "prod.conf"), []byte("HOST=reel\nUSER=root\nPORT=2222\nCONTENT_PATH=/etc/bbsoric/site.json\nSERVICE=bbsoric\nRELOAD=restart\n"), 0o644)

	profs, err := LoadProfiles(dir)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	if got := Names(profs); len(got) != 2 || got[0] != "dev" || got[1] != "prod" {
		t.Fatalf("Names = %v, attendu [dev prod]", got)
	}
	if !profs["dev"].Local || profs["dev"].ContentPath != "content/site.json" {
		t.Errorf("profil dev mal lu : %+v", profs["dev"])
	}
	// le .conf réel doit l'emporter sur le .example
	if profs["prod"].Host != "reel" || profs["prod"].Port != "2222" || profs["prod"].Reload != "restart" {
		t.Errorf("prod.conf doit primer sur l'exemple : %+v", profs["prod"])
	}
}

func TestLoadSiteProfiles(t *testing.T) {
	base := t.TempDir()
	// profils du site "site" (site.json)
	siteDir := filepath.Join(base, "site")
	os.MkdirAll(siteDir, 0o755)
	os.WriteFile(filepath.Join(siteDir, "dev.conf.example"), []byte("LOCAL=1\nCONTENT_PATH=content/site.json\n"), 0o644)
	os.WriteFile(filepath.Join(siteDir, "prod.conf.example"), []byte("HOST=h\nUSER=root\nCONTENT_PATH=/etc/bbsoric/site.json\n"), 0o644)

	profs, err := LoadSiteProfiles(base, "site.json")
	if err != nil {
		t.Fatalf("LoadSiteProfiles: %v", err)
	}
	if got := Names(profs); len(got) != 2 || got[0] != "dev" || got[1] != "prod" {
		t.Fatalf("Names = %v, attendu [dev prod]", got)
	}
	// un site sans répertoire de profils -> map vide (pas d'erreur)
	empty, err := LoadSiteProfiles(base, "autre.json")
	if err != nil || len(empty) != 0 {
		t.Fatalf("site sans profils : %v / %d", err, len(empty))
	}
	// nom de site invalide -> erreur
	if _, err := LoadSiteProfiles(base, "../evil.json"); err == nil {
		t.Errorf("un site avec traversée doit être refusé")
	}
}

func TestSaveProfileRoundTrip(t *testing.T) {
	base := t.TempDir()
	p := &Profile{Local: false, Host: "h1", User: "root", Port: "2222",
		ContentPath: "/etc/bbsoric/site.json", Service: "bbsoric", Reload: "restart"}
	if err := SaveProfile(base, "site.json", "prod", p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	// relecture via LoadSiteProfiles
	profs, err := LoadSiteProfiles(base, "site.json")
	if err != nil {
		t.Fatalf("LoadSiteProfiles: %v", err)
	}
	got := profs["prod"]
	if got == nil || got.Host != "h1" || got.Port != "2222" || got.Reload != "restart" || got.Service != "bbsoric" {
		t.Fatalf("profil relu incorrect : %+v", got)
	}
	// site/env invalides -> erreur
	if err := SaveProfile(base, "../x.json", "prod", p); err == nil {
		t.Errorf("site avec traversée doit être refusé")
	}
	if err := SaveProfile(base, "site.json", "../evil", p); err == nil {
		t.Errorf("env avec traversée doit être refusé")
	}
}

func TestSiteKey(t *testing.T) {
	for in, want := range map[string]string{"site.json": "site", "bbs2.json": "bbs2", "x": "x"} {
		if got := SiteKey(in); got != want {
			t.Errorf("SiteKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDeployLocalBackupAndOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "site.json")
	os.WriteFile(target, []byte(`{"old":true}`), 0o644)
	p := &Profile{Name: "dev", Local: true, ContentPath: target, Reload: "none"}

	res, err := Deploy(p, []byte(validSite), false, "STAMP")
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !res.OK && len(res.Log) == 0 {
		t.Errorf("log attendu")
	}
	// la cible doit contenir le nouveau contenu
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), `"start":"main"`) {
		t.Errorf("la cible n'a pas été écrasée : %s", got)
	}
	// une sauvegarde horodatée doit exister avec l'ancien contenu
	bak, err := os.ReadFile(target + ".bak.STAMP")
	if err != nil {
		t.Fatalf("sauvegarde absente : %v", err)
	}
	if !strings.Contains(string(bak), "old") {
		t.Errorf("la sauvegarde doit contenir l'ancien contenu : %s", bak)
	}
}

func TestDeployRefusesInvalidSite(t *testing.T) {
	p := &Profile{Name: "dev", Local: true, ContentPath: filepath.Join(t.TempDir(), "s.json")}
	if _, err := Deploy(p, []byte(`{"pages":{}}`), false, "S"); err == nil {
		t.Errorf("un contenu invalide doit être refusé")
	}
}

// TestDeployRejectsShellInjection : un champ de profil interpolé dans une commande
// shell distante (CONTENT_PATH, SERVICE, HOST…) contenant un métacaractère doit
// faire échouer le déploiement AVANT toute exécution (régression S11.3).
func TestDeployRejectsShellInjection(t *testing.T) {
	cases := []*Profile{
		{Name: "p", Host: "h", User: "root", Port: "22", ContentPath: "/etc/x.json; rm -rf /", Service: "bbsoric", Reload: "reload"},
		{Name: "p", Host: "h", User: "root", Port: "22", ContentPath: "/etc/x.json", Service: "bbsoric; reboot", Reload: "reload"},
		{Name: "p", Host: "h;evil", User: "root", Port: "22", ContentPath: "/etc/x.json", Service: "bbsoric", Reload: "reload"},
		{Name: "p", Host: "h", User: "root", Port: "22", ContentPath: "/etc/$(id).json", Service: "bbsoric", Reload: "reload"},
	}
	for i, p := range cases {
		if _, err := Deploy(p, []byte(validSite), false, "S"); err == nil {
			t.Errorf("cas %d : une injection shell doit être refusée (%+v)", i, p)
		}
		// dry-run aussi : la validation précède le dispatch
		if _, err := Deploy(p, []byte(validSite), true, "S"); err == nil {
			t.Errorf("cas %d : injection refusée même en dry-run", i)
		}
	}
	// SaveProfile doit aussi refuser de persister un profil dangereux.
	bad := &Profile{ContentPath: "/etc/x.json; rm -rf /", Host: "h", User: "root", Port: "22"}
	if err := SaveProfile(t.TempDir(), "site.json", "prod", bad); err == nil {
		t.Errorf("SaveProfile doit refuser un CONTENT_PATH avec métacaractère")
	}
}

func TestDeployDryRunDoesNothing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "site.json")
	p := &Profile{Name: "prod", Host: "h", User: "root", Port: "22", ContentPath: target, Service: "bbsoric", Reload: "restart"}

	res, err := Deploy(p, []byte(validSite), true, "S")
	if err != nil {
		t.Fatalf("Deploy dry-run: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("le dry-run ne doit rien écrire")
	}
	joined := strings.Join(res.Log, "\n")
	if !strings.Contains(joined, "dry-run") || !strings.Contains(joined, "scp") {
		t.Errorf("le journal dry-run doit décrire les actions : %s", joined)
	}
}
