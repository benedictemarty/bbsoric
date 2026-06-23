package bbs

import (
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Version applicative affichée par la bannière (suit le sprint courant).
const bbsVersion = "Sprint 4"

// firstKey renvoie le premier caractère significatif d'une ligne, en majuscule
// (0 si la ligne est vide). Sert à router les choix de menu.
func firstKey(line string) byte {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}
	c := line[0]
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return c
}

// upperKey met une touche ASCII en majuscule (pour router les choix de menu de
// façon insensible à la casse).
func upperKey(c byte) byte {
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return c
}

// rule trace une règle pleine largeur (40 col) en couleur par défaut.
func rule() string { return strings.Repeat("=", oascii.Cols) }

// makeInk renvoie l'octet d'attribut d'encre sous forme de chaîne (1 caractère).
func makeInk(c oascii.Color) string { return string([]byte{oascii.InkAttr(c)}) }
