package ula

import (
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/internal/render"
)

// ligne renvoie la ligne row du Dump (sans espaces de fin, pour comparer le texte).
func ligne(t *Terminal, row int) string {
	return strings.TrimRight(strings.Split(t.Dump(), "\n")[row], " ")
}

func TestTexteEtRetourLigne(t *testing.T) {
	term := New()
	term.Write([]byte("BONJOUR\r\nORIC"))
	if got := ligne(term, 0); got != "BONJOUR" {
		t.Errorf("ligne 0 = %q, veut \"BONJOUR\"", got)
	}
	if got := ligne(term, 1); got != "ORIC" {
		t.Errorf("ligne 1 = %q, veut \"ORIC\"", got)
	}
	if c, r := term.Cursor(); c != 4 || r != 1 {
		t.Errorf("curseur = (%d,%d), veut (4,1)", c, r)
	}
}

func TestClamp40Colonnes(t *testing.T) {
	term := New()
	// 45 caractères : seuls les 40 premiers tiennent, pas de passage à la ligne.
	term.Write([]byte(strings.Repeat("X", 45)))
	if got := ligne(term, 0); len(got) != 40 {
		t.Errorf("ligne 0 longueur = %d, veut 40", len(got))
	}
	if got := ligne(term, 1); got != "" {
		t.Errorf("ligne 1 devrait être vide (pas de wrap), got %q", got)
	}
	if c, _ := term.Cursor(); c != 40 {
		t.Errorf("colonne = %d, veut 40 (clamp)", c)
	}
}

func TestBackspaceDestructif(t *testing.T) {
	term := New()
	term.Write([]byte("ABC\bX"))
	if got := ligne(term, 0); got != "ABX" {
		t.Errorf("got %q, veut \"ABX\"", got)
	}
}

func TestPlotPositionne(t *testing.T) {
	term := New()
	// oascii.Plot(col,row) = 1F col row.
	term.Write([]byte(oascii.Plot(10, 5)))
	term.Write([]byte("ICI"))
	if got := ligne(term, 5); got != strings.Repeat(" ", 10)+"ICI" {
		t.Errorf("ligne 5 = %q", got)
	}
	// Plot hors bornes : col clampée à 39, row à 27.
	term.Write([]byte(oascii.Plot(200, 200)))
	if c, r := term.Cursor(); c != 39 || r != 27 {
		t.Errorf("curseur clampé = (%d,%d), veut (39,27)", c, r)
	}
}

func TestDefilement(t *testing.T) {
	term := New()
	// Écrit 30 lignes numérotées 0..29 : les 2 premières doivent défiler hors écran.
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString("L")
		sb.WriteByte(byte('0' + i%10))
		sb.WriteString("\r\n")
	}
	term.Write([]byte(sb.String()))
	// 30 lignes suivies chacune d'un LF : le LF final défile une fois de plus, donc
	// L0..L2 sortent et le haut affiche L3 (comportement d'un terminal défilant).
	if got := ligne(term, 0); got != "L3" {
		t.Errorf("après défilement ligne 0 = %q, veut \"L3\"", got)
	}
	if _, r := term.Cursor(); r != Rows-1 {
		t.Errorf("curseur ligne = %d, veut %d", r, Rows-1)
	}
}

func TestAttributsSerielsRenduANSI(t *testing.T) {
	term := New()
	b := oascii.New()
	b.Ink(oascii.Red).Text("R").Ink(oascii.Green).Text("G")
	term.Write([]byte(b.String()))

	ansi := term.RenderANSI()
	// Encre rouge = ANSI 31, verte = 32 ; le fond par défaut = 40.
	if !strings.Contains(ansi, "\x1b[0;31;40m") {
		t.Errorf("séquence encre rouge absente du rendu ANSI")
	}
	if !strings.Contains(ansi, "\x1b[0;32;40m") {
		t.Errorf("séquence encre verte absente du rendu ANSI")
	}
	// Le texte lui-même est présent.
	if !strings.Contains(ansi, "R") || !strings.Contains(ansi, "G") {
		t.Errorf("texte RG absent du rendu ANSI")
	}
}

func TestCaractereInverse(t *testing.T) {
	term := New()
	// Encre blanche (7) sur fond noir (0), puis un 'A' inverse (0x80|'A').
	b := oascii.New()
	term.Write([]byte(b.String()))
	term.Write([]byte{0x80 | 'A'})
	cells := term.rowCells(0)
	c := cells[0]
	// Inverse : encre/fond échangés -> fg=0 (noir), bg=7 (blanc).
	if c.fg != 0 || c.bg != 7 || c.ch != 'A' {
		t.Errorf("cellule inverse = %+v, veut fg=0 bg=7 ch='A'", c)
	}
}

func TestHiresRenduPuisRetourTexte(t *testing.T) {
	term := New()
	term.Write([]byte("AVANT\r\n"))
	// 1F FC ouvre le flux HIRES ; HiOn + un point + HiEnd terminent le flux.
	term.Write([]byte(oascii.HiresCmd()))
	term.Write([]byte{oascii.HiOn, oascii.HiPoint, 10, 10, oascii.HiEnd})
	if !term.InHires() {
		t.Fatal("après 1F FC + flux, l'affichage devrait être en mode HIRES")
	}
	if ansi := term.RenderANSI(); !strings.Contains(ansi, "▀") {
		t.Error("le rendu HIRES devrait contenir des demi-blocs ▀")
	}
	// 1F FB : retour TEXT, écran effacé.
	term.Write([]byte(oascii.HiresOff()))
	if term.InHires() {
		t.Error("1F FB devrait rendre la main au mode texte")
	}
	if got := ligne(term, 0); got != "" {
		t.Errorf("1F FB doit effacer l'écran, ligne 0 = %q", got)
	}
}

func TestCharsetAlternatifUnicode(t *testing.T) {
	term := New()
	// Attribut texte « charset alternatif » (0x08 | bit0) puis le code '0' (bloc plein).
	term.Write([]byte{0x09, '0', '1'})
	cells := term.rowCells(0)
	if cells[1].ch != '█' { // '0' en police BBS = bloc plein
		t.Errorf("charset alt : cellule = %q, veut '█'", cells[1].ch)
	}
	if cells[2].ch != '▌' { // '1' = moitié gauche
		t.Errorf("charset alt : cellule = %q, veut '▌'", cells[2].ch)
	}
}

func TestInverseSerielMode(t *testing.T) {
	term := New()
	// Encre blanche/fond noir (défaut), inverse ON (29), 'A', inverse OFF (28), 'B'.
	term.Write([]byte{29, 'A', 28, 'B'})
	cells := term.rowCells(0)
	// Après l'attribut 29 (col 0), 'A' en col 1 : inversé -> fg=0 bg=7.
	if cells[1].fg != 0 || cells[1].bg != 7 {
		t.Errorf("inverse ON : cellule 'A' = fg%d bg%d, veut fg0 bg7", cells[1].fg, cells[1].bg)
	}
	// Après 28 (col 2), 'B' en col 3 : normal -> fg=7 bg=0.
	if cells[3].fg != 7 || cells[3].bg != 0 {
		t.Errorf("inverse OFF : cellule 'B' = fg%d bg%d, veut fg7 bg0", cells[3].fg, cells[3].bg)
	}
}

// TestHiresFluxReelServeur : bout en bout — un flux HIRES produit par l'ENCODEUR
// serveur (render.Hires, source unique du rendu) est décodé et rendu par oterm.
func TestHiresFluxReelServeur(t *testing.T) {
	page := &content.Page{Hires: &content.Hires{
		Draw: []content.HiresOp{
			{Op: "ink", C: int(oascii.Cyan)},
			{Op: "curset", X: 40, Y: 40},
			{Op: "line", X: 200, Y: 40},
			{Op: "circle", R: 20},
		},
	}}
	stream := render.Hires(page) // 1F FC HiOn ... HiEnd
	term := New()
	term.Write(stream)
	if !term.InHires() {
		t.Fatal("le flux HIRES serveur devrait basculer oterm en mode graphique")
	}
	// La ligne horizontale y=40 est bien tracée (couleur cyan attendue, x>=6).
	// On vérifie via le rendu ANSI qu'il contient des demi-blocs et la couleur cyan (36).
	ansi := term.RenderANSI()
	if !strings.Contains(ansi, "▀") {
		t.Error("rendu HIRES sans demi-blocs")
	}
	if !strings.Contains(ansi, "\x1b[36") { // avant-plan cyan = 36
		t.Error("la couleur cyan de la ligne devrait apparaître dans le rendu")
	}
}

func TestXmodemStatut(t *testing.T) {
	term := New()
	term.Write([]byte{oascii.PlotByte, 0xFE}) // download
	if !strings.Contains(term.Status(), "XMODEM") {
		t.Errorf("statut XMODEM attendu, got %q", term.Status())
	}
	// Le flux reprend normalement juste après (pas d'octet avalé indûment).
	term.Write([]byte("OK"))
	if got := ligne(term, 0); got != "OK" {
		t.Errorf("reprise après 1F FE = %q, veut \"OK\"", got)
	}
}
