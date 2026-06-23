package oascii

import "bytes"

// Screen est un buffer écran 40×28 « intelligent » : il maintient l'état COMPOSÉ
// (ce qu'on veut afficher) et l'état AFFICHÉ (ce que le terminal montre déjà).
// Render() ne produit QUE le flux nécessaire pour passer de l'affiché au composé :
// les cellules inchangées ne sont pas réémises.
//
// Ce rendu différentiel est exact sur Oric car l'écran EST la VRAM : chaque
// cellule (caractère ou attribut sériel) est indépendante, et l'ULA recompose la
// ligne à chaque balayage en relisant la mémoire. On peut donc réécrire une
// cellule isolée — son rendu reprend les attributs déjà posés en amont sur la
// ligne. Le repositionnement utilise la commande plot X,Y (cf. PlotByte).
//
// Intérêt : sur la liaison série (9600 bauds), réémettre tout un écran coûte
// ~1,2 s ; un diff de quelques cellules est quasi instantané — idéal pour les
// écrans dynamiques (jeux, valeurs rafraîchies, animations).
type Screen struct {
	buf   []byte // composition courante (ce qu'on veut)
	shown []byte // dernier état émis (ce que le terminal affiche)
}

// NewScreen crée un buffer rempli d'espaces. Le premier Render() émet tout le
// contenu non vide (l'écran du terminal est supposé inconnu au départ).
func NewScreen() *Screen {
	n := Cols * Rows
	s := &Screen{buf: make([]byte, n), shown: make([]byte, n)}
	for i := range s.buf {
		s.buf[i] = ' '
	}
	// shown laissé à 0 (≠ espace) : le premier Render() émet donc l'écran complet.
	return s
}

// index renvoie l'offset (col,row) dans le buffer, ou -1 hors écran.
func index(col, row int) int {
	if col < 0 || col >= Cols || row < 0 || row >= Rows {
		return -1
	}
	return row*Cols + col
}

// Clear remplit la composition d'espaces (n'émet rien tant que Render n'est pas
// appelé).
func (s *Screen) Clear() {
	for i := range s.buf {
		s.buf[i] = ' '
	}
}

// Put pose un octet (caractère imprimable ou attribut sériel) en (col,row).
// Hors écran : ignoré.
func (s *Screen) Put(col, row int, b byte) {
	if i := index(col, row); i >= 0 {
		s.buf[i] = b
	}
}

// PutText pose une suite d'octets à partir de (col,row), bornée à la fin de la
// ligne (pas de passage à la ligne suivante). Renvoie la colonne après le dernier
// octet posé.
func (s *Screen) PutText(col, row int, text string) int {
	if row < 0 || row >= Rows {
		return col
	}
	for i := 0; i < len(text) && col < Cols; i++ {
		if col >= 0 {
			s.buf[row*Cols+col] = text[i]
		}
		col++
	}
	return col
}

// At renvoie l'octet composé en (col,row) (0 hors écran).
func (s *Screen) At(col, row int) byte {
	if i := index(col, row); i >= 0 {
		return s.buf[i]
	}
	return 0
}

// Reset oublie l'état affiché : le prochain Render() réémettra tout l'écran.
// Utile après une (re)connexion ou un changement d'écran complet.
func (s *Screen) Reset() {
	for i := range s.shown {
		s.shown[i] = 0
	}
}

// Render produit le flux minimal pour mettre l'écran du terminal à jour, puis
// mémorise le nouvel état affiché. Renvoie un flux vide si rien n'a changé.
//
// Les cellules modifiées consécutives sur une même ligne sont regroupées en un
// seul segment : un positionnement plot X,Y suivi des octets du segment. Les
// segments ne franchissent jamais une fin de ligne (le terminal borne à 40
// colonnes).
func (s *Screen) Render() []byte {
	var out bytes.Buffer
	for row := 0; row < Rows; row++ {
		col := 0
		for col < Cols {
			i := row*Cols + col
			if s.buf[i] == s.shown[i] {
				col++
				continue
			}
			start := col
			for col < Cols && s.buf[row*Cols+col] != s.shown[row*Cols+col] {
				col++
			}
			out.WriteString(Plot(start, row))
			out.Write(s.buf[row*Cols+start : row*Cols+col])
		}
	}
	copy(s.shown, s.buf)
	return out.Bytes()
}

// Buffer renvoie la composition courante (40×28 octets). À ne pas modifier
// directement (utiliser Put/PutText).
func (s *Screen) Buffer() []byte { return s.buf }
