// Commande validate-content : valide un fichier de contenu (site.json) avec le
// MÊME paquet que le serveur (internal/content) — pratique avant un déploiement.
//
//	go run ./tools/validate-content <site.json>
//
// Sortie : « VALIDE : N pages, M sources » (code 0) ou « INVALIDE : … » (code 1).
package main

import (
	"fmt"
	"os"

	"github.com/benedictemarty/bbsoric/internal/content"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage : validate-content <site.json>")
		os.Exit(2)
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "lecture :", err)
		os.Exit(1)
	}
	s, err := content.Parse(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "INVALIDE :", err)
		os.Exit(1)
	}
	fmt.Printf("VALIDE : %d pages, %d sources\n", len(s.Pages), len(s.SourcesDonnees))
}
