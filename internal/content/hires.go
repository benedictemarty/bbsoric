package content

import "fmt"

// --- Modèle HIRES (pages graphiques haute résolution) ---
//
// L'Oric dispose d'un mode HIRES 240×200 (VRAM $A000, 8000 octets) sur le haut de
// l'écran ; les 3 dernières lignes restent en TEXT ($BB80) — idéal pour garder un
// menu/texte sous le décor graphique (cf. pattern « menu sur fond d'écran »).
//
// Une page HIRES combine, dans cet ordre, DEUX modèles non exclusifs :
//   - « bitmap » : un fond `Background` = buffer VRAM complet (8000 octets), posé
//     d'un bloc (compressé RLE sur la liaison série) — pour un logo / splash ;
//   - « primitives » : une liste `Draw` de commandes de tracé (curset/line/box/
//     circle/…) appliquées par-dessus — compact et animable.
//
// Les deux sont sérialisés vers le terminal par un MÊME flux de commandes
// (`render.Hires` → sous-commande série `1F FC`, cf. paquet oascii).

// Dimensions du mode HIRES Oric et taille du buffer VRAM correspondant.
const (
	HiresW          = 240                      // pixels en largeur
	HiresH          = 200                      // pixels en hauteur (lignes HIRES)
	HiresBytesPerRow = HiresW / 6              // 6 pixels par octet → 40 octets/ligne
	HiresBitmapSize = HiresBytesPerRow * HiresH // 40 × 200 = 8000 octets VRAM
)

// Hires décrit une page graphique. `Background`, s'il est présent, est le buffer
// VRAM HIRES complet (exactement `HiresBitmapSize` octets, base64 en JSON) ;
// `Draw` est la liste des primitives appliquées ensuite.
type Hires struct {
	Background []byte    `json:"background,omitempty"` // fond bitmap (modèle « bitmap »)
	Draw       []HiresOp `json:"draw,omitempty"`       // primitives (modèle « vectoriel »)
}

// HiresOp est une primitive de tracé. Le terminal maintient un « crayon » (pen) :
//
//	ink / paper   → C : couleur courante (0-7)
//	curset        → X,Y : déplace le crayon (sans tracer)
//	point         → X,Y : allume le pixel (crayon ← X,Y)
//	line          → X,Y : trace du crayon vers (X,Y) (crayon ← X,Y)
//	box / fillbox → X,Y : rectangle (vide / plein) du crayon à (X,Y)
//	circle        → R   : cercle de rayon R autour du crayon
//	char          → X,Y + Ch : caractère ASCII tracé en (X,Y)
type HiresOp struct {
	Op string `json:"op"`
	X  int    `json:"x,omitempty"`
	Y  int    `json:"y,omitempty"`
	R  int    `json:"r,omitempty"`
	C  int    `json:"c,omitempty"`  // couleur (ink/paper)
	Ch string `json:"ch,omitempty"` // caractère imprimé (op « char »)
}

// hiresOpsArgs : pour chaque primitive, le « profil » d'arguments attendu, utilisé
// par la validation. true = la primitive utilise des coordonnées X,Y.
var hiresOps = map[string]struct{ coords, color, radius, char bool }{
	"ink":     {color: true},
	"paper":   {color: true},
	"curset":  {coords: true},
	"point":   {coords: true},
	"line":    {coords: true},
	"box":     {coords: true},
	"fillbox": {coords: true},
	"circle":  {radius: true},
	"char":    {coords: true, char: true},
}

// validate vérifie une page HIRES : taille du fond, présence d'au moins un modèle,
// primitives connues et arguments dans les bornes (240×200, couleurs 0-7).
func (h *Hires) validate(pageID string) error {
	if len(h.Background) != 0 && len(h.Background) != HiresBitmapSize {
		return fmt.Errorf("page %q : hires.background doit faire %d octets (%d)", pageID, HiresBitmapSize, len(h.Background))
	}
	if len(h.Background) == 0 && len(h.Draw) == 0 {
		return fmt.Errorf("page %q : hires sans 'background' ni 'draw'", pageID)
	}
	for i, op := range h.Draw {
		prof, ok := hiresOps[op.Op]
		if !ok {
			return fmt.Errorf("page %q : primitive hires #%d inconnue %q", pageID, i, op.Op)
		}
		if prof.color && (op.C < 0 || op.C > 7) {
			return fmt.Errorf("page %q : primitive hires #%d (%s) couleur %d hors 0-7", pageID, i, op.Op, op.C)
		}
		if prof.radius && op.R < 0 {
			return fmt.Errorf("page %q : primitive hires #%d (circle) rayon négatif %d", pageID, i, op.R)
		}
		if prof.coords && (op.X < 0 || op.X >= HiresW || op.Y < 0 || op.Y >= HiresH) {
			return fmt.Errorf("page %q : primitive hires #%d (%s) point (%d,%d) hors %d×%d", pageID, i, op.Op, op.X, op.Y, HiresW, HiresH)
		}
		if prof.char && op.Ch == "" {
			return fmt.Errorf("page %q : primitive hires #%d (char) sans 'ch'", pageID, i)
		}
	}
	return nil
}
