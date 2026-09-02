package main

// Traduction des touches du terminal PC vers les codes attendus par le serveur BBS.
//
// Le serveur (server/internal/bbs/datawindow.go) attend les codes flèches de
// l'Oric — keyUp=0x0B, keyDown=0x0C, keyLeft=0x0E, keyRight=0x0F — que le clavier
// du terminal Oric produit nativement. Un terminal PC, lui, émet les flèches sous
// forme de séquences d'échappement ANSI : CSI « ESC [ A/B/C/D » ou SS3
// « ESC O A/B/C/D ». Sans traduction, ces octets partent tels quels et le serveur
// ne navigue pas (seuls '+'/'-' fonctionnaient). translateKeys fait la conversion.

// Codes flèches Oric attendus par le serveur (cf. datawindow.go keyUp/Down/Left/Right).
const (
	oricUp    = 0x0B
	oricDown  = 0x0C
	oricLeft  = 0x0E
	oricRight = 0x0F
)

// indexByte renvoie l'indice de la première occurrence de c dans b, ou -1.
func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

// arrowCode mappe l'octet final d'une séquence de flèche (A/B/C/D) vers le code Oric.
func arrowCode(final byte) (byte, bool) {
	switch final {
	case 'A':
		return oricUp, true
	case 'B':
		return oricDown, true
	case 'C':
		return oricRight, true
	case 'D':
		return oricLeft, true
	}
	return 0, false
}

// translateKeys convertit les séquences de flèches ANSI présentes dans `in` en
// codes flèches Oric ; les autres octets passent tels quels. Les séquences CSI non
// gérées (Home, Suppr, flèches avec paramètres…) sont ignorées plutôt que transmises
// en clair (elles injecteraient des « [ A » parasites côté serveur).
//
// Si `in` se termine par une séquence d'échappement INCOMPLÈTE (flèche à cheval sur
// deux lectures stdin), les octets correspondants sont renvoyés dans `rest` : l'appelant
// doit les recoller au début du prochain chunk. `out` est toujours une tranche neuve ;
// `rest` est une sous-tranche de `in` (à copier si `in` doit être réutilisé).
func translateKeys(in []byte) (out, rest []byte) {
	out = make([]byte, 0, len(in))
	for i := 0; i < len(in); {
		b := in[i]
		if b != 0x1B { // pas un ESC : octet ordinaire
			out = append(out, b)
			i++
			continue
		}
		// ESC : début possible d'une séquence de flèche.
		if i+1 >= len(in) {
			return out, in[i:] // ESC seul en fin de chunk -> attendre la suite
		}
		switch in[i+1] {
		case '[': // CSI : ESC [ <params> <final 0x40-0x7E>
			j := i + 2
			for j < len(in) && (in[j] < 0x40 || in[j] > 0x7E) {
				j++
			}
			if j >= len(in) {
				return out, in[i:] // final pas encore arrivé -> reporter
			}
			if j == i+2 { // aucun paramètre : flèche simple potentielle
				if c, ok := arrowCode(in[j]); ok {
					out = append(out, c)
				}
			}
			// sinon (avec paramètres, ou final non-flèche) : séquence ignorée
			i = j + 1
		case 'O': // SS3 : ESC O <final>
			if i+2 >= len(in) {
				return out, in[i:] // incomplète -> reporter
			}
			if c, ok := arrowCode(in[i+2]); ok {
				out = append(out, c)
			}
			i += 3
		default:
			// ESC isolé (ni [ ni O) : transmis tel quel (ESC = quitter côté serveur).
			out = append(out, b)
			i++
		}
	}
	return out, nil
}
