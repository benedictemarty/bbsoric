package ula

// bbsRunes mappe les codes de la police BBS alternative (charset redéfinissable
// de l'Oric) vers leur équivalent Unicode. La police BBS EST composée de symboles
// Unicode (filets, blocs, symboles) : source unique = tools/genfont (le commentaire
// de chaque glyphe donne le symbole visé). Une cellule en charset alternatif
// (attribut texte bit 0) est donc rendue fidèlement par la rune correspondante,
// sans rasterisation pixel. Un code absent retombe sur son caractère ASCII de base.
var bbsRunes = map[byte]rune{
	// Filets simples.
	'-': '─', '|': '│',
	'a': '┌', 'b': '┐', 'c': '└', 'd': '┘',
	'e': '├', 'f': '┤', 'g': '┬', 'h': '┴', 'i': '┼',
	// Filets doubles.
	'A': '═', 'B': '║', 'C': '╔', 'D': '╗', 'E': '╚', 'F': '╝',
	'G': '╠', 'H': '╣', 'I': '╦', 'J': '╩', 'K': '╬',
	// Blocs & quarts.
	'0': '█', '1': '▌', '2': '▐', '3': '▀', '4': '▄',
	'L': '▘', 'M': '▝', 'N': '▖', 'O': '▗', 'P': '▞', 'Q': '▚',
	// Trames.
	'5': '░', '6': '▒', '7': '▓',
	// Symboles BBS.
	'.': '•', '>': '►', '<': '◄', '^': '▲', 'v': '▼', '*': '★',
	'+': '+', 'x': '✕', 'y': '✓', 'o': '○', '8': '◆', '9': '●',
}
