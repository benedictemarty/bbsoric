package dwgrid

import (
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
	RenderGrid(scr, dw, src, rows, -1, 2, 10, 23, "", "", false, false)
	c1 := scr.At(contentX, dataTop) & 0x7F   // 1er caractère du « No »
	c2 := scr.At(contentX+1, dataTop) & 0x7F // 2e (espace si « 1  », chiffre si « 11 »)
	if c1 != '1' || c2 != ' ' {
		t.Errorf("numérotation par page attendue (« 1  »), lu %q%q (index absolu ?)", string(rune(c1)), string(rune(c2)))
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
