package bbs

import (
	"time"

	"github.com/benedictemarty/bbsoric/server/internal/server"
)

// Contrôle de flux HIRES par CADENCE (pas d'acquittement, donc aucun changement
// firmware — profite à tous les terminaux déjà déployés). Le terminal n'exerce
// aucune contre-pression : si le serveur déverse un flux HIRES d'un bloc (un gros
// bitmap de fond, ou des images d'animation rapprochées), le FIFO série du
// terminal déborde et le flux se désynchronise. On écrit donc le flux en
// TRONÇONS bornés, en dormant entre chaque le temps que le lien (et le 6502) le
// draine — jamais plus vite que le débit réel.
const (
	// Débit prudent en octets/seconde : 9600 baud ≈ 960 o/s en série brute, moins
	// une marge pour le décodage RLE + écriture VRAM côté 6502 (surtout quand le
	// flux comporte beaucoup de petits blits d'en-tête).
	hiresBytesPerSec = 700
	// Taille d'un tronçon : nettement sous le FIFO du terminal (512 o) pour qu'il
	// soit drainé avant l'envoi du suivant.
	hiresChunk = 240
)

// writeHiresPaced écrit un flux HIRES en tronçons cadencés au débit du lien, pour
// ne pas saturer le FIFO série du terminal. Renvoie l'erreur d'écriture éventuelle.
// data vide : no-op.
func writeHiresPaced(s *server.Session, data []byte) error {
	return paceHires(s.Write, data, hiresBytesPerSec, hiresChunk, time.Sleep)
}

// paceHires est le cœur testable : écrit data en tronçons via write, en dormant
// entre chaque au débit choisi. write/sleep sont injectables (tests). Un tronçon
// ≤ 0 envoie tout d'un bloc ; un débit ≤ 0 n'attend pas.
func paceHires(write func(string) error, data []byte, bytesPerSec, chunk int, sleep func(time.Duration)) error {
	if chunk <= 0 {
		chunk = len(data)
	}
	for off := 0; off < len(data); off += chunk {
		end := off + chunk
		if end > len(data) {
			end = len(data)
		}
		if err := write(string(data[off:end])); err != nil {
			return err
		}
		if bytesPerSec > 0 {
			// Temps de drainage du tronçon au débit choisi.
			sleep(time.Duration(end-off) * time.Second / time.Duration(bytesPerSec))
		}
	}
	return nil
}
