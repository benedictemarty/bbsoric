// Package web embarque les ressources statiques du studio forge (servies par
// le binaire, sans dépendance de fichiers à l'exécution).
package web

import "embed"

//go:embed index.html app.js style.css charset.js altcharset.js
var FS embed.FS
