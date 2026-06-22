package content

import (
	"os"
	"testing"
)

// TestRepoSiteJSON garantit que le fichier de contenu versionné du dépôt
// (content/site.json) est valide.
func TestRepoSiteJSON(t *testing.T) {
	const path = "../../content/site.json"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s absent: %v", path, err)
	}
	if _, err := Parse(b); err != nil {
		t.Fatalf("content/site.json invalide: %v", err)
	}
}
