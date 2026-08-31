package ula

import (
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/oascii"
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

func TestHiresAvaleJusquA1FFB(t *testing.T) {
	term := New()
	term.Write([]byte("AVANT\r\n"))
	// 1F FC = début HIRES ; on envoie des octets quelconques (dont un 0x1F isolé)
	// puis 1F FB = retour TEXT (efface l'écran).
	term.Write([]byte{oascii.PlotByte, 0xFC})
	term.Write([]byte{0x10, 0x1F, 0x20, 0x00, 0xFF})
	if !strings.Contains(term.Status(), "HIRES") {
		t.Errorf("statut HIRES attendu, got %q", term.Status())
	}
	term.Write([]byte{oascii.PlotByte, 0xFB})
	if term.Status() != "" {
		t.Errorf("statut devrait être vidé après 1F FB, got %q", term.Status())
	}
	if got := ligne(term, 0); got != "" {
		t.Errorf("1F FB doit effacer l'écran, ligne 0 = %q", got)
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
