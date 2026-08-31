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
// attributs sériels des trois familles sont gérés (encre/fond, texte : charset
// alternatif « police BBS » rendu en Unicode + clignotement, mode vidéo :
// inverse ON/OFF). Le mode HIRES (1F FC) est rasterisé et rendu en demi-blocs
// (paquet internal/hires) ; 1F FB rend la main au mode texte. Les transferts
// XMODEM (1F FE/FD) ne sont pas pris en charge (message de statut).
package ula

import (
	"strconv"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/pcterm/internal/hires"
)

// Cols/Rows reprennent la géométrie de l'écran Oric (40×28).
const (
	Cols = oascii.Cols
	Rows = oascii.Rows
)

// États de la machine de décodage (miroir de PLOTST dans client/term.s).
const (
	stNormal  = iota // flux normal (caractères / attributs / contrôle)
	stAfter1F        // octet 0x1F reçu : attendre la sous-commande ou la colonne
	stPlotRow        // colonne du plot mémorisée : attendre la ligne
	stHires          // page HIRES en cours : octets avalés jusqu'à 1F FB
)

// Modes d'affichage : écran texte 40×28 ou écran graphique HIRES 240×200.
const (
	modeText = iota
	modeHires
)

// Types de transfert détectés dans le flux (le client reprend alors la main sur
// la connexion pour dérouler XMODEM).
const (
	XferNone     = iota
	XferDownload // 1F FE : le serveur va ENVOYER un fichier (download côté client)
	XferUpload   // 1F FD : le serveur veut RECEVOIR un fichier (upload côté client)
)

// Terminal porte l'état d'écran décodé. Non concurrent : sérialiser les appels
// (le client protège Write/RenderANSI par un verrou).
type Terminal struct {
	grid   [Rows * Cols]byte
	col    int
	row    int
	st     int
	plotX  int
	mode    int           // modeText ou modeHires
	hr      *hires.Raster // rasteriseur du flux HIRES courant (si mode HIRES)
	status  string        // dernier message de statut
	pending int           // transfert détecté en attente (XferNone/Download/Upload)
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

// InHires indique si l'affichage est en mode graphique HIRES.
func (t *Terminal) InHires() bool { return t.mode == modeHires }

// Write injecte des octets reçus du BBS dans le décodeur. Ne faillit jamais
// (implémente io.Writer pour un usage direct avec io.Copy si besoin). Ignore la
// détection de transfert (voir WriteScan pour la gérer).
func (t *Terminal) Write(p []byte) (int, error) {
	for _, b := range p {
		t.feed(b)
	}
	t.pending = XferNone
	return len(p), nil
}

// WriteScan décode p octet par octet et S'ARRÊTE dès qu'un déclencheur de
// transfert (1F FE/FD) est rencontré : il renvoie le nombre d'octets consommés
// (jusqu'au déclencheur inclus) et le type de transfert. Les octets suivants de p
// (en-tête + flux XMODEM) sont alors à traiter par l'appelant sur la connexion.
// Sans transfert, consomme tout p et renvoie (len(p), XferNone).
func (t *Terminal) WriteScan(p []byte) (consumed, xfer int) {
	for i, b := range p {
		t.feed(b)
		if t.pending != XferNone {
			x := t.pending
			t.pending = XferNone
			return i + 1, x
		}
	}
	return len(p), XferNone
}

func (t *Terminal) feed(b byte) {
	switch t.st {
	case stAfter1F:
		switch b {
		case 0xFE: // 1F FE : le serveur va envoyer un fichier (download)
			t.pending = XferDownload
			t.st = stNormal
		case 0xFD: // 1F FD : le serveur veut recevoir un fichier (upload)
			t.pending = XferUpload
			t.st = stNormal
		case 0xFC: // 1F FC : ouvre un flux de commandes HIRES
			t.hr = hires.New()
			t.mode = modeHires
			t.status = ""
			t.st = stHires
		case 0xFB: // 1F FB : retour TEXT (efface l'écran, comme le firmware)
			t.clear()
			t.mode = modeText
			t.hr = nil
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
		// Rasterise le flux HIRES ; à HiEnd on repasse au traitement normal du flux
		// tout en CONTINUANT d'afficher l'image (jusqu'à 1F FB qui rend la main au
		// mode texte). Les octets texte éventuels reçus entre HiEnd et 1F FB sont
		// alors posés dans la grille (mais masqués tant que mode == HIRES).
		if t.hr.Feed(b) {
			t.st = stNormal
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
	ch     rune
}

// rowCells résout une ligne selon la sémantique sérielle de l'ULA (miroir du
// simulateur ULA du studio, renderScreenBuf) : encre 7 / fond 0 en début de ligne,
// les attributs modifient l'état courant et s'affichent comme un bloc de fond, les
// caractères utilisent (encre, fond). Gère les trois familles d'attributs :
//   - encre (0x00–0x07), texte (0x08–0x0F : bit0 charset alt, bit2 clignotement),
//     fond (0x10–0x17) ;
//   - mode vidéo (0x18–0x1F) : inverse OFF (28) / ON (29) en état sériel de ligne.
//
// Inverse combiné : bit 0x80 du caractère XOR inverse sériel courant. En charset
// alternatif (police BBS), le glyphe est rendu par sa rune Unicode (cf. bbsRunes).
func (t *Terminal) rowCells(row int) [Cols]cell {
	var out [Cols]cell
	ink, paper, attr := byte(7), byte(0), byte(0)
	serialInv := false
	for col := 0; col < Cols; col++ {
		b := t.grid[row*Cols+col]
		if b&0x60 == 0 { // attribut (0x00–0x1F ou 0x80–0x9F)
			v := b & 0x1F
			switch v & 0x18 {
			case 0x00:
				ink = v & 7
			case 0x08:
				attr = v & 7
			case 0x10:
				paper = v & 7
			case 0x18: // mode vidéo : 28 = inverse off, 29 = inverse on
				switch v {
				case 28:
					serialInv = false
				case 29:
					serialInv = true
				}
			}
			// La case d'attribut s'affiche comme un bloc de la couleur de fond.
			out[col] = cell{fg: ink, bg: paper, ch: ' '}
			continue
		}
		code := b & 0x7F
		var ru rune
		if attr&1 != 0 { // charset alternatif (police BBS) -> rune Unicode
			if r, ok := bbsRunes[code]; ok {
				ru = r
			} else if code >= 0x20 && code != 0x7F {
				ru = rune(code)
			} else {
				ru = ' '
			}
		} else if code < 0x20 || code == 0x7F {
			ru = ' '
		} else {
			ru = rune(code)
		}
		inv := b&0x80 != 0
		if serialInv {
			inv = !inv
		}
		fg, bg := ink, paper
		if inv { // inverse : encre/fond échangés
			fg, bg = paper, ink
		}
		out[col] = cell{fg: fg, bg: bg, blink: attr&4 != 0, ch: ru}
	}
	return out
}

// RenderANSI produit une trame ANSI complète repeignant l'écran 40×28 : curseur
// masqué, retour en haut à gauche, 28 lignes colorées, puis le curseur matériel
// repositionné sur le curseur Oric et réaffiché. Les 8 couleurs Oric coïncident
// avec l'ordre ANSI (0=noir … 7=blanc).
func (t *Terminal) RenderANSI() string {
	if t.mode == modeHires && t.hr != nil {
		return t.hr.RenderANSI() // écran graphique 240×200 en demi-blocs
	}
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
			sb.WriteRune(c.ch)
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
