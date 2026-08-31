package hires

import (
	"os"
	"testing"

	"github.com/benedictemarty/bbsoric/internal/content"
	"github.com/benedictemarty/bbsoric/internal/render"
)

// TestPixelExactVsEmulateur compare la VRAM HIRES ($A000, 8000 octets) produite
// par un VRAI firmware Oric dans oric1-emu à celle de mon rasteriseur, sur le
// MÊME flux fil (render.Hires de la page). C'est la preuve ultime « identique au
// client Oric » — pilotée par scripts/test-emulateur-hires.sh qui capture la RAM.
//
// Gaté : ne s'exécute que si ORIC_RAM_DUMP pointe une capture 64 Ko et ORIC_HIRES_JSON
// le site JSON (défaut docs/examples/hires-demo.json, page « logo »). Absent en CI.
func TestPixelExactVsEmulateur(t *testing.T) {
	dumpPath := os.Getenv("ORIC_RAM_DUMP")
	if dumpPath == "" {
		t.Skip("ORIC_RAM_DUMP non défini — capture émulateur requise (voir scripts/test-emulateur-hires.sh)")
	}
	ram, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("lecture du dump : %v", err)
	}
	const vbase, vsize = 0xA000, H * rowByte // 200×40 = 8000
	if len(ram) < vbase+vsize {
		t.Fatalf("dump trop court (%d octets)", len(ram))
	}
	emuVRAM := ram[vbase : vbase+vsize]

	jsonPath := os.Getenv("ORIC_HIRES_JSON")
	if jsonPath == "" {
		jsonPath = "../../../docs/examples/hires-demo.json"
	}
	page := os.Getenv("ORIC_HIRES_PAGE")
	if page == "" {
		page = "logo"
	}
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("lecture du site : %v", err)
	}
	site, err := content.Parse(raw)
	if err != nil {
		t.Fatalf("parse du site : %v", err)
	}
	pg, ok := site.Pages[page]
	if !ok || pg.Hires == nil {
		t.Fatalf("page HIRES %q introuvable dans %s", page, jsonPath)
	}

	// Mon rasteriseur sur le flux fil exact que le serveur envoie à l'Oric.
	r := New()
	for _, b := range render.Hires(pg) {
		r.Feed(b)
	}
	mine := r.VRAM()

	// Exception connue : le firmware écrit l'attribut de bascule mode vidéo 0x1E à
	// $BB80 (offset 7040 depuis $A000) pour latcher le mode HIRES dans l'ULA. Ce
	// n'est PAS du contenu dessiné (il n'est pas dans le flux fil render.Hires) :
	// mon rasteriseur, qui ne modélise que l'image, ne l'a pas — à juste titre.
	const modeLatchOffset = 0xBB80 - vbase // 7040

	diff, first := 0, -1
	for i := 0; i < vsize; i++ {
		if i == modeLatchOffset {
			continue // artefact firmware (latch ULA), hors contenu dessiné
		}
		if mine[i] != emuVRAM[i] {
			diff++
			if first < 0 {
				first = i
			}
		}
	}
	if diff != 0 {
		row, col := first/rowByte, first%rowByte
		t.Errorf("VRAM diffère de l'émulateur : %d/%d octets (1er à l'offset %d = row %d col %d ; moi=%#02x emu=%#02x)",
			diff, vsize, first, row, col, mine[first], emuVRAM[first])
	} else {
		t.Logf("VRAM identique à l'émulateur : %d/%d octets exacts (page %q ; seul l'octet de bascule mode ULA à $BB80 diffère, artefact firmware)", vsize-1, vsize, page)
	}
}
