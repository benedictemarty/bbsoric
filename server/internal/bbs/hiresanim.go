package bbs

import (
	"context"
	"time"

	"github.com/benedictemarty/bbsoric/internal/oascii"
	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// init enregistre l'applet de démonstration d'animation HIRES.
func init() { Register("hiresanim", hiresAnimApplet) }

// hiresAnimApplet montre l'ANIMATION HIRES via le buffer différentiel
// (oascii.HiresScreen) : une balle rebondit dans un cadre fixe. Le cadre ne change
// pas d'une image à l'autre → seule la zone de la balle (l'ancienne effacée, la
// nouvelle allumée) est réémise, ce qui rend l'animation fluide sur la liaison
// série 9600 bauds. Preuve vivante que « ne renvoyer que ce qui change » marche.
//
// Durée bornée (le lien est en clair, pas d'entrée non bloquante ici) : ~8 s puis
// retour au menu. La 1ʳᵉ image émet HiOn (bascule + efface) ; les suivantes ne
// portent que des HiBlit (diff).
func hiresAnimApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	// Contrainte SÉRIE (pas de contrôle de flux) : chaque image doit rester PETITE
	// pour ne pas saturer le FIFO du terminal. On évite donc un cadre en bordure
	// (200 mini-blits dispersés) : le décor fixe est fait de 2 rails HORIZONTAUX
	// (octets contigus → 1 blit chacun) et le sprite est un rectangle PLEIN (lignes
	// contiguës → peu de runs). La cadence laisse au terminal le temps de rendre.
	const (
		frames  = 24
		sprW    = 14
		sprH    = 10
		railTop = 20
		railBot = oascii.HiresPixH - 20
		minX    = 2
		maxX    = oascii.HiresPixW - 2 - sprW
		// Plancher de période pour la lisibilité (le pacing gère le débit ; ce
		// délai évite juste que de tout petits diffs défilent trop vite).
		minPeriode = 60 * time.Millisecond
	)
	hs := oascii.NewHiresScreen()
	x, y := 40, 90
	dx, dy := 8, 5

	for f := 0; f < frames; f++ {
		select {
		case <-ctx.Done():
			return Outcome{Quit: true}
		default:
		}

		hs.Clear()
		hs.Line(0, railTop, oascii.HiresPixW-1, railTop) // rails fixes (blits contigus,
		hs.Line(0, railBot, oascii.HiresPixW-1, railBot) //  non réémis après l'image 0)
		hs.FillBox(x, y, x+sprW, y+sprH)                 // le sprite (seul à bouger)

		if out := hs.Render(); out != nil {
			// Écriture CADENCÉE : ne jamais dépasser le débit du lien (sinon le
			// FIFO du terminal déborde et l'animation se corrompt). Le pacing
			// fournit aussi l'essentiel de l'intervalle entre images.
			if err := writeHiresPaced(s, out); err != nil {
				return Outcome{Quit: true}
			}
		}

		// Rebonds entre les rails.
		x += dx
		y += dy
		if x <= minX || x >= maxX {
			dx = -dx
			x += dx
		}
		if y <= railTop+2 || y+sprH >= railBot-2 {
			dy = -dy
			y += dy
		}
		time.Sleep(minPeriode)
	}

	// Retour au mode TEXT puis invite (sinon l'écran resterait figé en HIRES).
	_ = s.Write(oascii.HiresOff())
	header(s, "ANIMATION HIRES")
	info := oascii.New()
	info.Newline().Ink(oascii.Green).Text("Animation terminee (buffer differentiel).").Newline()
	info.Ink(oascii.White).Text("Seul le sprite etait reemis a chaque image.").Newline().Newline()
	info.Text("Appuyez sur une touche...").Newline()
	_ = s.Write(info.String())
	anyKey(s)
	return Outcome{Done: true}
}
