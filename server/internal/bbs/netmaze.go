package bbs

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/server"
	"github.com/benedictemarty/netmaze/arena"
	"github.com/benedictemarty/netmaze/engine"
	"github.com/benedictemarty/netmaze/proto"
)

// init enregistre l'applet NetMaze (deathmatch de labyrinthe contre des bots).
func init() { Register("netmaze", netmazeApplet) }

// netmazeTick cadence la partie (~3 Hz), comme le serveur standalone : assez
// vif pour un duel, assez lent pour le lien série et le rendu texte.
const netmazeTick = 333 * time.Millisecond

// netmazeQuit est la touche qui abandonne la partie et revient au menu. Elle est
// choisie hors des commandes de jeu (w/x/q/d/a/e/f/s/espace, cf. proto) : ici le
// chiffre 0 (ou ESC).
const netmazeQuit = '0'

// netmazeApplet fait jouer une partie solo de NetMaze dans le terminal BBS : le
// joueur (siège 1) affronte des bots, rendu en OASCII texte 40×28. La logique de
// jeu vient du module netmaze (engine/arena) ; l'applet n'est qu'un adaptateur de
// transport + un rendu (cf. netmaze/docs/integration-bbsoric.md).
//
// NetMaze est temps réel : contrairement à connect4 (tour par tour), on ne peut
// pas bloquer sur ReadKey. La boucle est donc pilotée par un ticker et, à chaque
// tick, draine les touches disponibles sans bloquer (Session.Raw + deadline
// courte) avant d'avancer l'arène et de rendre l'état.
func netmazeApplet(ctx context.Context, s *server.Session, ac *AppContext) Outcome {
	seat := &netmazeSeat{s: s}
	a := arena.NewSolo(arena.DefaultSoloConfig(), seat)
	seat.game = a.Game() // le rendu lit le labyrinthe (mini-carte) + la config

	if err := s.Write(netmazeIntro()); err != nil {
		return Outcome{Quit: true}
	}

	ticker := time.NewTicker(netmazeTick)
	defer ticker.Stop()

	for !a.Over() {
		select {
		case <-ctx.Done():
			s.ClearDeadline()
			return Outcome{Quit: true}
		case <-ticker.C:
			seat.poll() // lecture clavier non bloquante → action de ce tick
			if seat.quit {
				s.ClearDeadline()
				return Outcome{}
			}
			a.RunOnce() // NextAction → bots → Step → Deliver (rendu)
			if seat.writeErr != nil {
				s.ClearDeadline()
				return Outcome{Quit: true}
			}
		}
	}
	s.ClearDeadline()

	if err := s.Write(netmazeEnd(seat.game, a.Winner())); err != nil {
		return Outcome{Quit: true}
	}
	anyKey(s)
	return Outcome{}
}

// netmazeSeat adapte la session BBS à arena.Session. Il ne lance pas de
// goroutine : la boucle de l'applet appelle poll() puis RunOnce(), si bien que
// les octets non consommés restent dans le tampon de la session et ne sont pas
// « volés » au menu quand on rend la main.
type netmazeSeat struct {
	s    *server.Session
	game *engine.GameState

	last     engine.Action // dernière action drainée depuis le tick précédent
	has      bool
	quit     bool  // le joueur a demandé à quitter
	writeErr error // première erreur d'écriture (déconnexion)
}

// NextAction renvoie l'action la plus récente drainée par poll, puis la consomme.
func (o *netmazeSeat) NextAction() (engine.Action, bool) {
	if !o.has {
		return engine.ActNone, false
	}
	o.has = false
	return o.last, true
}

// Deliver rend l'état du tick au joueur (HUD + mini-carte + radar en OASCII).
func (o *netmazeSeat) Deliver(v *arena.StateView) error {
	err := o.s.Write(renderNetmaze(o.game, v))
	if err != nil && o.writeErr == nil {
		o.writeErr = err
	}
	return err
}

// poll consomme, sans bloquer, les octets clavier déjà arrivés et met à jour la
// dernière action (modèle « une action par tick » : les frappes intermédiaires
// sont écrasées). Une courte deadline de lecture garantit qu'on ne bloque pas la
// cadence quand le joueur ne tape rien.
func (o *netmazeSeat) poll() {
	raw := o.s.Raw()
	_ = raw.SetReadDeadline(time.Now().Add(3 * time.Millisecond))
	var buf [32]byte
	for {
		n, err := raw.Read(buf[:])
		for i := 0; i < n; i++ {
			c := buf[i]
			if c == netmazeQuit || c == 27 { // 0 ou ESC : abandon
				o.quit = true
				continue
			}
			if act, ok := proto.ParseAction(c); ok { // même mapping que le client Oric
				o.last, o.has = act, true
			}
		}
		if err != nil {
			// Un timeout de deadline signifie « rien tapé ce tick » : normal. Toute
			// autre erreur est une déconnexion → on la propage comme erreur d'I/O.
			var ne net.Error
			if !(errors.As(err, &ne) && ne.Timeout()) && o.writeErr == nil {
				o.writeErr = err
			}
			break
		}
		if n == 0 {
			break
		}
	}
	o.s.ClearDeadline()
}
