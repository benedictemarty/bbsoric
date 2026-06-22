// Commande genfont : génère la police « BBS Oric » (charset alternatif Oric,
// 6×8) à partir de glyphes décrits en ASCII-art ci-dessous — SOURCE UNIQUE.
//
// Sorties :
//   - studio/web/altcharset.js : Uint8Array de 128×8 octets (window.ORIC_ALTCHARSET)
//     pour le simulateur ULA du studio ;
//   - client/altcharset.s : bloc de données 1024 octets (label altcharset_data)
//     copié dans $B800 par le terminal Oric.
//
// Chaque glyphe : 8 lignes de 6 colonnes. '#'/'X'/'*' = pixel allumé, autre = éteint.
// L'octet écran lit les pixels de gauche (bit5) à droite (bit0).
//
//	go run ./tools/genfont
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// glyph associe un code de caractère (0x20–0x7F) à un dessin 6×8.
type glyph struct {
	code byte
	art  []string
}

// La police BBS. Les filets/coins se raccordent (horizontale ligne 3, verticale
// colonne 2) ; double trait sur lignes 2 et 4 / colonnes 1 et 3.
var glyphs = []glyph{
	// ── Filets & cadres (simple) ────────────────────────────────────────────
	{'-', []string{"......", "......", "......", "######", "......", "......", "......", "......"}}, // ─ horizontale
	{'|', []string{"..#...", "..#...", "..#...", "..#...", "..#...", "..#...", "..#...", "..#..."}}, // │ verticale
	{'a', []string{"......", "......", "......", "..####", "..#...", "..#...", "..#...", "..#..."}}, // ┌ coin haut-gauche
	{'b', []string{"......", "......", "......", "###...", "..#...", "..#...", "..#...", "..#..."}}, // ┐ coin haut-droit
	{'c', []string{"..#...", "..#...", "..#...", "..####", "......", "......", "......", "......"}}, // └ coin bas-gauche
	{'d', []string{"..#...", "..#...", "..#...", "###...", "......", "......", "......", "......"}}, // ┘ coin bas-droit
	{'e', []string{"..#...", "..#...", "..#...", "..####", "..#...", "..#...", "..#...", "..#..."}}, // ├
	{'f', []string{"..#...", "..#...", "..#...", "####..", "..#...", "..#...", "..#...", "..#..."}}, // ┤
	{'g', []string{"......", "......", "......", "######", "..#...", "..#...", "..#...", "..#..."}}, // ┬
	{'h', []string{"..#...", "..#...", "..#...", "######", "......", "......", "......", "......"}}, // ┴
	{'i', []string{"..#...", "..#...", "..#...", "######", "..#...", "..#...", "..#...", "..#..."}}, // ┼
	// ── Filets double ───────────────────────────────────────────────────────
	{'A', []string{"......", "......", "######", "......", "######", "......", "......", "......"}}, // ═
	{'B', []string{".#.#..", ".#.#..", ".#.#..", ".#.#..", ".#.#..", ".#.#..", ".#.#..", ".#.#.."}}, // ║
	{'C', []string{"......", "......", ".####.", ".#....", ".#.##.", ".#.#..", ".#.#..", ".#.#.."}}, // ╔
	{'D', []string{"......", "......", ".####.", "....#.", ".##.#.", "..#.#.", "..#.#.", "..#.#."}}, // ╗
	{'E', []string{".#.#..", ".#.#..", ".#.##.", ".#....", ".####.", "......", "......", "......"}}, // ╚
	{'F', []string{"..#.#.", "..#.#.", ".##.#.", "....#.", ".####.", "......", "......", "......"}}, // ╝
	{'G', []string{".#.#..", ".#.#..", ".#.###", ".#....", ".#.###", ".#.#..", ".#.#..", ".#.#.."}}, // ╠
	{'H', []string{"..#.#.", "..#.#.", "###.#.", "....#.", "###.#.", "..#.#.", "..#.#.", "..#.#."}}, // ╣
	{'I', []string{"......", "......", "######", "......", "##.###", ".#.#..", ".#.#..", ".#.#.."}}, // ╦
	{'J', []string{".#.#..", ".#.#..", "##.###", "......", "######", "......", "......", "......"}}, // ╩
	{'K', []string{".#.#..", ".#.#..", "######", ".#.#..", "######", ".#.#..", ".#.#..", ".#.#.."}}, // ╬
	// ── Blocs ───────────────────────────────────────────────────────────────
	{'0', []string{"######", "######", "######", "######", "######", "######", "######", "######"}}, // █ plein
	{'1', []string{"###...", "###...", "###...", "###...", "###...", "###...", "###...", "###..."}}, // ▌ moitié gauche
	{'2', []string{"...###", "...###", "...###", "...###", "...###", "...###", "...###", "...###"}}, // ▐ moitié droite
	{'3', []string{"######", "######", "######", "######", "......", "......", "......", "......"}}, // ▀ moitié haut
	{'4', []string{"......", "......", "......", "......", "######", "######", "######", "######"}}, // ▄ moitié bas
	{'L', []string{"###...", "###...", "###...", "###...", "......", "......", "......", "......"}}, // ▘ quart haut-gauche
	{'M', []string{"...###", "...###", "...###", "...###", "......", "......", "......", "......"}}, // ▝ quart haut-droit
	{'N', []string{"......", "......", "......", "......", "###...", "###...", "###...", "###..."}}, // ▖ quart bas-gauche
	{'O', []string{"......", "......", "......", "......", "...###", "...###", "...###", "...###"}}, // ▗ quart bas-droit
	{'P', []string{"...###", "...###", "...###", "...###", "###...", "###...", "###...", "###..."}}, // ▞ diagonale
	{'Q', []string{"###...", "###...", "###...", "###...", "...###", "...###", "...###", "...###"}}, // ▚ diagonale
	// ── Trames ──────────────────────────────────────────────────────────────
	{'5', []string{"#.....", "......", "..#...", "......", "....#.", "......", "#.....", "......"}}, // ░ légère
	{'6', []string{"#.#.#.", ".#.#.#", "#.#.#.", ".#.#.#", "#.#.#.", ".#.#.#", "#.#.#.", ".#.#.#"}}, // ▒ moyenne
	{'7', []string{"##.###", "#.####", "#####.", "####.#", "##.###", "#.####", "#####.", "####.#"}}, // ▓ dense
	// ── Symboles BBS ────────────────────────────────────────────────────────
	{'.', []string{"......", "......", "..##..", "..##..", "......", "......", "......", "......"}}, // • puce
	{'>', []string{"......", ".#....", ".###..", ".#####", ".###..", ".#....", "......", "......"}}, // ► curseur droite
	{'<', []string{"......", "....#.", "..###.", "#####.", "..###.", "....#.", "......", "......"}}, // ◄ curseur gauche
	{'^', []string{"......", "..#...", ".###..", "#####.", "......", "......", "......", "......"}}, // ▲ haut
	{'v', []string{"......", "......", "......", "#####.", ".###..", "..#...", "......", "......"}}, // ▼ bas
	{'*', []string{"......", "..#...", "#.#.#.", ".###..", "#####.", ".###..", "#.#.#.", "..#..."}}, // ★ étoile
	{'+', []string{"......", "..#...", "..#...", "#####.", "..#...", "..#...", "......", "......"}}, // croix/plus
	{'x', []string{"......", "#...#.", ".#.#..", "..#...", ".#.#..", "#...#.", "......", "......"}}, // ✕
	{'y', []string{"......", "....#.", "...##.", "#.##..", "###...", ".#....", "......", "......"}}, // ✓ coche
	{'o', []string{"......", ".###..", "#...#.", "#...#.", "#...#.", ".###..", "......", "......"}}, // ○ rond
	{'8', []string{"..#...", ".###..", "#####.", "######", "#####.", ".###..", "..#...", "......"}}, // ◆ losange
	{'9', []string{"......", ".###..", "#####.", "#####.", "#####.", ".###..", "......", "......"}}, // ● point plein
}

func main() {
	const n = 128
	font := make([]byte, n*8) // tout vide par défaut
	for _, g := range glyphs {
		if len(g.art) != 8 {
			fmt.Fprintf(os.Stderr, "glyphe 0x%02X : %d lignes (8 attendues)\n", g.code, len(g.art))
			os.Exit(1)
		}
		for row, line := range g.art {
			var bits byte
			for x := 0; x < 6 && x < len(line); x++ {
				if c := line[x]; c == '#' || c == 'X' || c == '*' {
					bits |= 1 << (5 - x)
				}
			}
			font[int(g.code)*8+row] = bits
		}
	}

	// studio/web/altcharset.js
	b64 := base64.StdEncoding.EncodeToString(font)
	js := "// Police BBS Oric (charset alternatif redefinissable, 128 x 8 octets).\n" +
		"// GENERE par tools/genfont — NE PAS EDITER A LA MAIN.\n" +
		"window.ORIC_ALTCHARSET = Uint8Array.from(atob('" + b64 + "'), c => c.charCodeAt(0));\n"
	write("studio/web/altcharset.js", js)

	// client/altcharset.s
	var s strings.Builder
	s.WriteString("; Police BBS Oric (charset alternatif) — GENERE par tools/genfont.\n")
	s.WriteString("; 128 caracteres x 8 octets = 1024 octets, a copier dans $B800.\n")
	s.WriteString("altcharset_data:\n")
	for i := 0; i < len(font); i += 8 {
		s.WriteString("        .byt ")
		parts := make([]string, 8)
		for j := 0; j < 8; j++ {
			parts[j] = fmt.Sprintf("$%02X", font[i+j])
		}
		s.WriteString(strings.Join(parts, ","))
		s.WriteString(fmt.Sprintf("   ; car 0x%02X\n", i/8))
	}
	write("client/altcharset.s", s.String())

	fmt.Printf("genfont : %d glyphes, %d octets -> studio/web/altcharset.js + client/altcharset.s\n", len(glyphs), len(font))
}

func write(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
