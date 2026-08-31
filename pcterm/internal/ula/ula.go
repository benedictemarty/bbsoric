// Package ula émule l'écran texte de l'Oric côté client : il décode le flux
// OASCII reçu du BBS en une grille 40×28 d'octets bruts + un curseur, puis le
// restitue en séquences ANSI colorées (pour un terminal moderne portable).
//
// Le modèle est fidèle à l'ULA de l'Oric (cf. internal/oascii et docs/oascii.md) :
//   - un octet d'attribut (encre/fond/texte) OCCUPE une case et s'applique en
//     SÉRIE jusqu'à la fin de la ligne (l'ULA réinitialise les attributs à chaque
//     début de ligne) ;
//   - le rendu couleur n'est donc PAS mémorisé dans la grille : il est recalculé
//     ligne par ligne à l'affichage (RenderANSI), exactement comme la puce vidéo.
//
// Le décodage du flux reproduit le firmware terminal (client/term.s, handle_rx +
// putbyte) : caractères, CR/LF avec défilement, backspace destructif, clamp à 40
// colonnes (sans passage à la ligne), et la commande plot « 1F col row ». Les
// commandes XMODEM (1F FE/FD) et HIRES (1F FC/FB) ne sont pas rendues par ce
// client texte : elles positionnent seulement un message de statut.
package ula

import (
	"strconv"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
)

// Cols/Rows reprennent la géométrie de l'écran Oric (40×28).
const (
	Cols = oascii.Cols
	Rows = oascii.Rows
)

// États de la machine de décodage (miroir de PLOTST dans client/term.s).
const (
	stNormal = iota // flux normal (caractères / attributs / contrôle)
	stAfter1F       // octet 0x1F reçu : attendre la sous-commande ou la colonne
	stPlotRow       // colonne du plot mémorisée : attendre la ligne
	stHires         // page HIRES en cours : octets avalés jusqu'à 1F FB
)

// Terminal porte l'état d'écran décodé. Non concurrent : sérialiser les appels
// (le client protège Write/RenderANSI par un verrou).
type Terminal struct {
	grid    [Rows * Cols]byte
	col     int
	row     int
	st      int
	plotX   int
	hprev1F bool   // en mode HIRES : dernier octet vu = 0x1F (repère 1F FB)
	status  string // dernier message de statut (XMODEM/HIRES non rendus)
}

// New crée un terminal effacé (grille d'espaces, curseur en haut à gauche).
func New() *Terminal {
	t := &Terminal{}
	t.clear()
	return t
}

func (t *Terminal) clear() {
	for i := range t.grid {
		t.grid[i] = ' '
	}
	t.col, t.row = 0, 0
}

// Status renvoie le dernier message de statut (transfert/HIRES non rendus), ""
// s'il n'y a rien à signaler.
func (t *Terminal) Status() string { return t.status }

// Cursor renvoie la position courante du curseur d'écriture (col, row).
func (t *Terminal) Cursor() (int, int) { return t.col, t.row }

// Write injecte des octets reçus du BBS dans le décodeur. Ne faillit jamais
// (implémente io.Writer pour un usage direct avec io.Copy si besoin).
func (t *Terminal) Write(p []byte) (int, error) {
	for _, b := range p {
		t.feed(b)
	}
	return len(p), nil
}

func (t *Terminal) feed(b byte) {
	switch t.st {
	case stAfter1F:
		switch b {
		case 0xFE:
			t.status = "XMODEM download reçu — non pris en charge par ce client texte"
			t.st = stNormal
		case 0xFD:
			t.status = "XMODEM upload demandé — non pris en charge par ce client texte"
			t.st = stNormal
		case 0xFC:
			t.status = "page HIRES (graphique) — non rendue par ce client texte"
			t.hprev1F = false
			t.st = stHires
		case 0xFB:
			t.clear() // 1F FB = retour TEXT : le firmware efface l'écran
			t.status = ""
			t.st = stNormal
		default:
			t.plotX = int(b) // 1F <col> <row> : premier octet = colonne
			t.st = stPlotRow
		}
	case stPlotRow:
		t.setCursor(t.plotX, int(b))
		t.st = stNormal
	case stHires:
		// Avale le flux HIRES jusqu'au terminateur 1F FB (best-effort : un 1F FB
		// présent dans les données binaires sortirait prématurément — acceptable,
		// ce client ne rend pas le HIRES).
		if t.hprev1F {
			t.hprev1F = false
			if b == 0xFB {
				t.clear()
				t.status = ""
				t.st = stNormal
			}
		} else if b == oascii.PlotByte {
			t.hprev1F = true
		}
	default: // stNormal
		if b == oascii.PlotByte {
			t.st = stAfter1F
			return
		}
		t.putByte(b)
	}
}

// putByte reproduit client/term.s putbyte : CR, LF+défilement, backspace, et pose
// de tout autre octet (caractère OU attribut sériel) avec clamp à 40 colonnes.
func (t *Terminal) putByte(b byte) {
	switch b {
	case 0x0D: // CR : retour colonne 0
		t.col = 0
	case 0x0A: // LF : colonne 0 + ligne suivante (défilement en bas)
		t.col = 0
		t.row++
		if t.row >= Rows {
			t.scrollUp()
			t.row = Rows - 1
		}
	case 0x08: // backspace destructif
		if t.col > 0 {
			t.col--
			t.grid[t.row*Cols+t.col] = ' '
		}
	default:
		if t.col < Cols { // clamp 40 col : au-delà, l'octet est ignoré (pas de wrap)
			t.grid[t.row*Cols+t.col] = b
			t.col++
		}
	}
}

func (t *Terminal) scrollUp() {
	copy(t.grid[0:(Rows-1)*Cols], t.grid[Cols:Rows*Cols])
	last := (Rows - 1) * Cols
	for i := last; i < Rows*Cols; i++ {
		t.grid[i] = ' '
	}
}

// setCursor positionne le curseur en clampant col à 0..39 et row à 0..27 (cf.
// set_cursor_xy dans client/term.s).
func (t *Terminal) setCursor(col, row int) {
	if col < 0 {
		col = 0
	} else if col > Cols-1 {
		col = Cols - 1
	}
	if row < 0 {
		row = 0
	} else if row > Rows-1 {
		row = Rows - 1
	}
	t.col, t.row = col, row
}

// Dump renvoie le contenu texte de la grille (Rows lignes de Cols caractères),
// attributs et non-imprimables réduits à un espace. Pratique pour les tests et
// pour un aperçu sans couleur.
func (t *Terminal) Dump() string {
	var sb strings.Builder
	sb.Grow(Rows * (Cols + 1))
	for row := 0; row < Rows; row++ {
		for col := 0; col < Cols; col++ {
			b := t.grid[row*Cols+col]
			sb.WriteByte(printable(b))
		}
		if row < Rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// printable réduit un octet de grille à un caractère affichable : les attributs
// (groupe 0x00–0x1F / 0x80–0x9F) et non-imprimables deviennent un espace ; les
// caractères inverses (0xA0–0xFF) reviennent à leur glyphe de base.
func printable(b byte) byte {
	if b&0x60 == 0 { // octet d'attribut : case « fond » -> espace
		return ' '
	}
	c := b & 0x7F
	if c < 0x20 || c == 0x7F {
		return ' '
	}
	return c
}

// cell est l'état de rendu d'une case (couleurs déjà résolues, glyphe).
type cell struct {
	fg, bg byte
	blink  bool
	ch     byte
}

// rowCells résout une ligne selon la sémantique sérielle de l'ULA (miroir du
// simulateur ULA du studio, renderScreenBuf) : encre 7 / fond 0 en début de ligne,
// les attributs modifient l'état courant et s'affichent comme un bloc de fond, les
// caractères utilisent (encre, fond) avec inverse pour le bit 0x80.
func (t *Terminal) rowCells(row int) [Cols]cell {
	var out [Cols]cell
	ink, paper, attr := byte(7), byte(0), byte(0)
	for col := 0; col < Cols; col++ {
		b := t.grid[row*Cols+col]
		if b&0x60 == 0 { // attribut
			v := b & 0x1F
			switch v & 0x18 {
			case 0x00:
				ink = v & 7
			case 0x08:
				attr = v & 7
			case 0x10:
				paper = v & 7
			}
			// La case d'attribut s'affiche comme un bloc de la couleur de fond.
			out[col] = cell{fg: ink, bg: paper, ch: ' '}
			continue
		}
		ch := b & 0x7F
		if ch < 0x20 || ch == 0x7F {
			ch = ' '
		}
		fg, bg := ink, paper
		if b&0x80 != 0 { // caractère inverse : encre/fond échangés
			fg, bg = paper, ink
		}
		out[col] = cell{fg: fg, bg: bg, blink: attr&4 != 0, ch: ch}
	}
	return out
}

// RenderANSI produit une trame ANSI complète repeignant l'écran 40×28 : curseur
// masqué, retour en haut à gauche, 28 lignes colorées, puis le curseur matériel
// repositionné sur le curseur Oric et réaffiché. Les 8 couleurs Oric coïncident
// avec l'ordre ANSI (0=noir … 7=blanc).
func (t *Terminal) RenderANSI() string {
	var sb strings.Builder
	sb.WriteString("\x1b[?25l\x1b[H") // masque curseur + home
	lastFg, lastBg, lastBlink := byte(255), byte(255), false
	for row := 0; row < Rows; row++ {
		cells := t.rowCells(row)
		for col := 0; col < Cols; col++ {
			c := cells[col]
			if c.fg != lastFg || c.bg != lastBg || c.blink != lastBlink {
				sb.WriteString("\x1b[0;3")
				sb.WriteByte('0' + c.fg)
				sb.WriteString(";4")
				sb.WriteByte('0' + c.bg)
				if c.blink {
					sb.WriteString(";5")
				}
				sb.WriteByte('m')
				lastFg, lastBg, lastBlink = c.fg, c.bg, c.blink
			}
			sb.WriteByte(c.ch)
		}
		sb.WriteString("\x1b[0m")
		lastFg, lastBg, lastBlink = 255, 255, false
		if row < Rows-1 {
			sb.WriteString("\r\n")
		}
	}
	// Repositionne le curseur matériel sur le curseur Oric (1-indexé) et le réaffiche.
	sb.WriteString("\x1b[")
	sb.WriteString(strconv.Itoa(t.row + 1))
	sb.WriteByte(';')
	sb.WriteString(strconv.Itoa(t.col + 1))
	sb.WriteString("H\x1b[?25h")
	return sb.String()
}
