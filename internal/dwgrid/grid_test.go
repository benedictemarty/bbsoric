package dwgrid

import (
	"strings"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// TestRenderGridPerPageNumbering : la colonne « No » numérote PAR PAGE (1..parPage),
// pas en absolu — sinon, sur une grande table, un index ≥ 100 collerait le titre
// (colonne de 3) et ≥ 1000 serait tronqué.
func TestRenderGridPerPageNumbering(t *testing.T) {
	scr := oascii.NewScreen()
	dw := &content.DataWindow{ColonnesAffichees: []string{"nom"}, Largeurs: []int{20}}
	src := content.SourceDonnees{Colonnes: map[string]content.ColonneDef{"nom": {Type: "TEXT"}}}
	rows := []map[string]string{{"nom": "X"}, {"nom": "Y"}, {"nom": "Z"}}
	// Page 2, 10 par page : un index absolu vaudrait 11,12,13 ; par page = 1,2,3.
	RenderGrid(scr, dw, src, rows, -1, 2, 10, 23, "", "", false, false, 0)
	c1 := scr.At(contentX, dataTop) & 0x7F   // 1er caractère du « No »
	c2 := scr.At(contentX+1, dataTop) & 0x7F // 2e (espace si « 1  », chiffre si « 11 »)
	if c1 != '1' || c2 != ' ' {
		t.Errorf("numérotation par page attendue (« 1  »), lu %q%q (index absolu ?)", string(rune(c1)), string(rune(c2)))
	}
}

// TestRenderGridScroll : la flèche droite (scroll>0) décale le contenu COMPLET
// (non tronqué) de la ligne sélectionnée, révélant le texte coupé par les largeurs.
func TestRenderGridScroll(t *testing.T) {
	dw := &content.DataWindow{ColonnesAffichees: []string{"titre"}, Largeurs: []int{6}}
	src := content.SourceDonnees{Colonnes: map[string]content.ColonneDef{"titre": {Type: "TEXT"}}}
	long := "ABCDEFGHIJKLMNOP" // 16 > largeur 6 (tronqué normalement)
	rows := []map[string]string{{"titre": long}}

	// Sans scroll : la ligne sélectionnée montre le début tronqué.
	s0 := oascii.NewScreen()
	RenderGrid(s0, dw, src, rows, 0, 1, 10, 1, "", "", false, false, 0)
	// Avec scroll 8 : la ligne montre le contenu décalé de 8 -> "IJKLMNOP".
	s8 := oascii.NewScreen()
	RenderGrid(s8, dw, src, rows, 0, 1, 10, 1, "", "", false, false, 8)

	rowText := func(scr *oascii.Screen) string {
		var b []byte
		for c := contentX; c < oascii.Cols; c++ {
			b = append(b, scr.At(c, dataTop)&0x7F)
		}
		return string(b)
	}
	t0, t8 := rowText(s0), rowText(s8)
	if !strings.Contains(t0, "ABCDEF") {
		t.Errorf("sans scroll : début attendu (ABCDEF) ; vu %q", t0)
	}
	if !strings.Contains(t8, "IJKLMNOP") {
		t.Errorf("scroll 8 : suite attendue (IJKLMNOP) ; vu %q", t8)
	}
}

// TestGridLignesMax : le nombre de lignes affichables est borné par la hauteur écran.
func TestGridLignesMax(t *testing.T) {
	if n := GridLignesMax(&content.DataWindow{}); n <= 0 || n > 20 {
		t.Errorf("GridLignesMax défaut incohérent : %d", n)
	}
	if n := GridLignesMax(&content.DataWindow{LignesMax: 5}); n != 5 {
		t.Errorf("GridLignesMax(5) = %d, attendu 5", n)
	}
}
