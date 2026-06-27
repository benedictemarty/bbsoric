package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
)

const validSite = `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"Q","label":"Quitter","target":"__quit__"}]}}}`

func TestListLoadSave(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "site.json"), []byte(validSite), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	s := New(dir)

	names, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 1 || names[0] != "site.json" {
		t.Fatalf("List = %v, attendu [site.json]", names)
	}

	data, err := s.Load("site.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := content.Parse(data); err != nil {
		t.Fatalf("le site chargé doit être valide : %v", err)
	}
}

// dwSite : un site DataWindow (source + page grille) doit passer la validation
// du studio et round-tripper sans perte (régression : ne pas droper sources/datawindow).
const dwSite = `{"start":"g","sources_donnees":{"rep":{"table":"rep","colonnes":{"id":{"type":"INTEGER","cle_primaire":true,"auto_increment":true},"nom":{"type":"TEXT","requis":true}}}},"pages":{"g":{"title":"GRILLE","datawindow":{"source":"rep","colonnes_affichees":["nom"],"largeurs":[20]}}}}`

func TestSaveLoadDataWindowRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save("dw.json", []byte(dwSite)); err != nil {
		t.Fatalf("Save site DataWindow refusé : %v", err)
	}
	data, err := s.Load("dw.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	site, err := content.Parse(data)
	if err != nil {
		t.Fatalf("le site DataWindow rechargé doit être valide : %v", err)
	}
	if _, ok := site.SourcesDonnees["rep"]; !ok {
		t.Error("la source 'rep' a été perdue au round-trip")
	}
	if p := site.Pages["g"]; p == nil || p.DataWindow == nil || p.DataWindow.Source != "rep" {
		t.Errorf("le descripteur datawindow de la page 'g' a été perdu : %+v", site.Pages["g"])
	}
}

// richDwSite : un site DataWindow « complet » tel que produit par l'éditeur du
// studio (onglet Données + descripteur grille). Tous les champs doivent survivre
// au round-trip JSON : colonnes typées (pattern/valeur_defaut/auto_date/longueur_max),
// seed donnees, source API (url/racine/ttl_sec) et couleurs/lignes_max/editable.
const richDwSite = `{"start":"g","sources_donnees":{
  "rep":{"table":"rep","tri_defaut":"nom ASC","lignes_par_page":12,"colonnes":{
    "id":{"type":"INTEGER","libelle":"ID","cle_primaire":true,"auto_increment":true},
    "nom":{"type":"TEXT","libelle":"Nom","requis":true,"longueur_max":16,"pattern":"^[A-Z]"},
    "note":{"type":"INTEGER","libelle":"Note","valeur_defaut":0},
    "cree":{"type":"DATETIME","libelle":"Cree","auto_date":true}},
    "donnees":[{"nom":"Alice","note":5},{"nom":"Bob","note":3}]},
  "meteo":{"type_source":"api","tri_defaut":"ville ASC","lignes_par_page":10,
    "api":{"url":"https://x/meteo.json","racine":"results","ttl_sec":300},
    "colonnes":{"ville":{"type":"TEXT","libelle":"Ville"},"temp":{"type":"INTEGER","libelle":"Temp"}}}},
  "pages":{"g":{"title":"GRILLE","datawindow":{"source":"rep",
    "colonnes_affichees":["nom","note"],"largeurs":[16,4],
    "couleur_entete":"yellow","couleur_lignes":"cyan","couleur_selection":"green",
    "lignes_max":15,"editable":true}}}}`

func TestSaveLoadRichDataWindowRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save("rich.json", []byte(richDwSite)); err != nil {
		t.Fatalf("Save site DataWindow complet refusé : %v", err)
	}
	data, err := s.Load("rich.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	site, err := content.Parse(data)
	if err != nil {
		t.Fatalf("le site rechargé doit être valide : %v", err)
	}

	// Source SQLite : colonnes typées + seed préservés.
	rep, ok := site.SourcesDonnees["rep"]
	if !ok {
		t.Fatal("source 'rep' perdue")
	}
	if rep.TriDefaut != "nom ASC" || rep.LignesParPage != 12 {
		t.Errorf("paramètres source perdus : %+v", rep)
	}
	if c := rep.Colonnes["nom"]; c.Pattern != "^[A-Z]" || c.LongueurMax != 16 || !c.Requis {
		t.Errorf("champs colonne 'nom' perdus : %+v", c)
	}
	if c := rep.Colonnes["cree"]; !c.AutoDate {
		t.Errorf("auto_date de 'cree' perdu : %+v", c)
	}
	if len(rep.Donnees) != 2 {
		t.Errorf("seed perdu : %d lignes (attendu 2)", len(rep.Donnees))
	}

	// Source API : url/racine/ttl_sec préservés.
	meteo, ok := site.SourcesDonnees["meteo"]
	if !ok || !meteo.EstAPI() {
		t.Fatalf("source API 'meteo' perdue : %+v", meteo)
	}
	if meteo.API == nil || meteo.API.URL == "" || meteo.API.Racine != "results" || meteo.API.TTL != 300 {
		t.Errorf("config API perdue : %+v", meteo.API)
	}

	// Descripteur grille : couleurs, lignes_max et editable préservés.
	dw := site.Pages["g"].DataWindow
	if dw == nil || dw.CouleurEntete != "yellow" || dw.CouleurLignes != "cyan" || dw.CouleurSelection != "green" {
		t.Errorf("couleurs grille perdues : %+v", dw)
	}
	if dw.LignesMax != 15 || !dw.Editable {
		t.Errorf("lignes_max/editable perdus : %+v", dw)
	}
}

func TestSaveValidatesBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// JSON valide -> écrit et relisible.
	if err := s.Save("ok.json", []byte(validSite)); err != nil {
		t.Fatalf("Save valide: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ok.json")); err != nil {
		t.Errorf("le fichier doit exister: %v", err)
	}

	// JSON invalide (cible inexistante) -> refusé, pas de fichier.
	bad := `{"start":"main","pages":{"main":{"title":"M","type":"menu","entries":[{"key":"1","label":"x","target":"absent"}]}}}`
	if err := s.Save("bad.json", []byte(bad)); err == nil {
		t.Errorf("un site invalide doit être refusé")
	}
	if _, err := os.Stat(filepath.Join(dir, "bad.json")); !os.IsNotExist(err) {
		t.Errorf("aucun fichier ne doit être écrit pour un site invalide")
	}
}

func TestSafePathRejectsTraversal(t *testing.T) {
	s := New(t.TempDir())
	for _, bad := range []string{"../x.json", "a/b.json", "..", "site.txt", ""} {
		if _, err := s.Load(bad); err == nil {
			t.Errorf("Load(%q) devrait échouer", bad)
		}
	}
}
