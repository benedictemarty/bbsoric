// Package deploy pousse un contenu de site (site.json) vers un environnement
// décrit par un profil (dev / int / prod). Le studio est la source de vérité :
// le déploiement ÉCRASE la cible, après avoir validé le contenu (même paquet que
// le serveur) et SAUVEGARDÉ la version précédente (copie horodatée).
//
// Profil local : copie de fichier (le bbsd local recharge à chaud).
// Profil distant : ssh/scp (réutilise le mécanisme de deploy/vps-deploy.sh),
// sans dépendance externe.
package deploy

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/benedictemarty/bbsoric/internal/content"
)

// profileFieldRe restreint les valeurs de profil à un jeu de caractères sûr.
// HOST/USER/PORT/CONTENT_PATH/SERVICE sont interpolés dans des commandes exécutées
// par le shell distant (ssh « test -f … && cp … », « systemctl reload … ») : un
// métacaractère shell (`;`, `$(…)`, backtick, espace, guillemet…) permettrait une
// injection de commande. On refuse tout ce qui sort de ce jeu.
var profileFieldRe = regexp.MustCompile(`^[A-Za-z0-9._@/-]*$`)

// validateProfileFields refuse un profil dont un champ interpolé dans une commande
// shell distante contient un caractère non sûr.
func validateProfileFields(p *Profile) error {
	for _, f := range []struct{ label, v string }{
		{"HOST", p.Host}, {"USER", p.User}, {"PORT", p.Port},
		{"CONTENT_PATH", p.ContentPath}, {"SERVICE", p.Service},
	} {
		if !profileFieldRe.MatchString(f.v) {
			return fmt.Errorf("valeur de profil %s invalide (caractères non autorisés) : %q", f.label, f.v)
		}
	}
	return nil
}

// Profile décrit un environnement de déploiement.
type Profile struct {
	Name        string `json:"name"`        // identifiant (dev, int, prod…)
	Local       bool   `json:"local"`       // true = copie de fichier locale (pas de SSH)
	Host        string `json:"host"`        // hôte SSH (distant)
	User        string `json:"user"`        // utilisateur SSH
	Port        string `json:"port"`        // port SSH (défaut 22)
	ContentPath string `json:"contentPath"` // chemin cible du site.json
	Service     string `json:"service"`     // service systemd (pour reload/restart)
	Reload      string `json:"reload"`      // none | reload | restart
}

// Marshal sérialise le profil au format .conf (KEY=VALUE).
func (p *Profile) Marshal() []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Profil %s — généré par le studio\n", p.Name)
	if p.Local {
		b.WriteString("LOCAL=1\n")
	}
	for _, kv := range [][2]string{
		{"HOST", p.Host}, {"USER", p.User}, {"PORT", p.Port},
		{"CONTENT_PATH", p.ContentPath}, {"SERVICE", p.Service},
	} {
		if kv[1] != "" {
			fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
		}
	}
	reload := p.Reload
	if reload == "" {
		reload = "none"
	}
	fmt.Fprintf(&b, "RELOAD=%s\n", reload)
	return []byte(b.String())
}

// SaveProfile écrit le profil dans <baseDir>/<site>/<env>.conf (écriture atomique).
func SaveProfile(baseDir, siteFile, env string, p *Profile) error {
	if siteFile == "" || strings.ContainsAny(siteFile, `/\`) || strings.Contains(siteFile, "..") {
		return fmt.Errorf("site invalide : %q", siteFile)
	}
	if env == "" || strings.ContainsAny(env, `/\`) || strings.Contains(env, "..") {
		return fmt.Errorf("environnement invalide : %q", env)
	}
	if err := validateProfileFields(p); err != nil {
		return err
	}
	dir := filepath.Join(baseDir, SiteKey(siteFile))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	p.Name = env
	path := filepath.Join(dir, env+".conf")
	tmp, err := os.CreateTemp(dir, ".profile-*.conf.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(p.Marshal()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// LoadProfiles lit les profils d'un répertoire. Chaque fichier `<nom>.conf` ou
// `<nom>.conf.example` définit le profil `<nom>` ; un `.conf` réel a priorité sur
// l'exemple de même nom (pratique : les exemples servent de défauts, les `.conf`
// gitignorés contiennent l'infra réelle).
func LoadProfiles(dir string) (map[string]*Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	profs := map[string]*Profile{}
	real := map[string]bool{} // ce profil a un .conf réel (priorité sur .example)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		var name string
		isExample := false
		switch {
		case strings.HasSuffix(n, ".conf.example"):
			name, isExample = strings.TrimSuffix(n, ".conf.example"), true
		case strings.HasSuffix(n, ".conf"):
			name = strings.TrimSuffix(n, ".conf")
		default:
			continue
		}
		if isExample && real[name] {
			continue // un .conf réel déjà chargé l'emporte
		}
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		if !isExample {
			real[name] = true // écrase un éventuel .example déjà lu
		}
		profs[name] = parseProfile(name, data)
	}
	return profs, nil
}

// SiteKey dérive la clé de profil d'un nom de fichier site (base sans .json) :
// "site.json" -> "site". Les profils d'un site vivent dans <baseDir>/<clé>/.
func SiteKey(siteFile string) string {
	return strings.TrimSuffix(filepath.Base(siteFile), ".json")
}

// LoadSiteProfiles charge les profils PROPRES À UN SITE :
// <baseDir>/<clé>/<env>.conf (ou .example). Chaque site a son trio dev/int/prod.
// Map vide si le répertoire du site n'existe pas encore.
func LoadSiteProfiles(baseDir, siteFile string) (map[string]*Profile, error) {
	if siteFile == "" || strings.ContainsAny(siteFile, `/\`) || strings.Contains(siteFile, "..") {
		return nil, fmt.Errorf("site invalide : %q", siteFile)
	}
	profs, err := LoadProfiles(filepath.Join(baseDir, SiteKey(siteFile)))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*Profile{}, nil
		}
		return nil, err
	}
	return profs, nil
}

// parseProfile lit un format KEY=VALUE simple (commentaires '#', valeurs sans
// guillemets requis).
func parseProfile(name string, data []byte) *Profile {
	p := &Profile{Name: name, Port: "22", Reload: "none"}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.ToUpper(k) {
		case "LOCAL":
			p.Local = v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
		case "HOST":
			p.Host = v
		case "USER":
			p.User = v
		case "PORT":
			p.Port = v
		case "CONTENT_PATH":
			p.ContentPath = v
		case "SERVICE":
			p.Service = v
		case "RELOAD":
			p.Reload = strings.ToLower(v)
		}
	}
	return p
}

// Names renvoie les noms de profils triés.
func Names(profs map[string]*Profile) []string {
	out := make([]string, 0, len(profs))
	for n := range profs {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Result est le compte rendu d'un déploiement.
type Result struct {
	OK  bool     `json:"ok"`
	Log []string `json:"log"`
}

// Deploy valide le site puis le pousse vers le profil. En dry-run, aucune action
// n'est exécutée : le journal décrit ce qui SERAIT fait. `stamp` horodate la
// sauvegarde (injecté pour des tests déterministes).
func Deploy(p *Profile, site []byte, dryRun bool, stamp string) (*Result, error) {
	if p == nil {
		return nil, fmt.Errorf("profil inconnu")
	}
	if p.ContentPath == "" {
		return nil, fmt.Errorf("profil %q : CONTENT_PATH manquant", p.Name)
	}
	if err := validateProfileFields(p); err != nil {
		return nil, err
	}
	if _, err := content.Parse(site); err != nil {
		return nil, fmt.Errorf("contenu invalide, déploiement refusé : %w", err)
	}
	r := &Result{Log: []string{}}
	log := func(f string, a ...any) { r.Log = append(r.Log, fmt.Sprintf(f, a...)) }
	backup := p.ContentPath + ".bak." + stamp

	log("profil %q (%s) -> %s", p.Name, kind(p), p.ContentPath)
	log("validation du contenu : OK (%d octets)", len(site))

	if p.Local {
		return r, deployLocal(p, site, backup, dryRun, log)
	}
	return r, deployRemote(p, site, backup, dryRun, log)
}

func kind(p *Profile) string {
	if p.Local {
		return "local"
	}
	return "ssh " + p.User + "@" + p.Host + ":" + p.Port
}

func deployLocal(p *Profile, site []byte, backup string, dryRun bool, log func(string, ...any)) error {
	if _, err := os.Stat(p.ContentPath); err == nil {
		log("sauvegarde : cp %s %s", p.ContentPath, backup)
		if !dryRun {
			old, err := os.ReadFile(p.ContentPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(backup, old, 0o644); err != nil {
				return err
			}
		}
	} else {
		log("pas de fichier existant (première écriture)")
	}
	log("écriture (écrase) : %s", p.ContentPath)
	if !dryRun {
		if err := atomicWrite(p.ContentPath, site); err != nil {
			return err
		}
	}
	log("reload : aucun (bbsd local recharge à chaud)")
	if dryRun {
		log("[dry-run] aucune action exécutée")
	}
	return nil
}

func deployRemote(p *Profile, site []byte, backup string, dryRun bool, log func(string, ...any)) error {
	sshArgs := []string{"-p", p.Port, "-o", "ConnectTimeout=8", p.User + "@" + p.Host}
	dest := p.User + "@" + p.Host + ":" + p.ContentPath
	bkCmd := fmt.Sprintf("test -f %s && cp %s %s || true", p.ContentPath, p.ContentPath, backup)

	log("ssh %s '%s'", strings.Join(sshArgs, " "), bkCmd)
	log("scp -P %s <contenu> %s", p.Port, dest)
	switch p.Reload {
	case "reload":
		log("ssh … systemctl reload %s", p.Service)
	case "restart":
		log("ssh … systemctl restart %s", p.Service)
	default:
		log("reload : aucun (rechargement à chaud)")
	}
	if dryRun {
		log("[dry-run] aucune action exécutée")
		return nil
	}

	// Exécution réelle (ssh/scp doivent être disponibles et configurés).
	if err := run("ssh", append(sshArgs, bkCmd)...); err != nil {
		return fmt.Errorf("sauvegarde distante : %w", err)
	}
	tmp, err := os.CreateTemp("", "site-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(site); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if err := run("scp", "-P", p.Port, "-o", "ConnectTimeout=8", tmp.Name(), dest); err != nil {
		return fmt.Errorf("copie distante : %w", err)
	}
	if p.Reload == "reload" || p.Reload == "restart" {
		if err := run("ssh", append(sshArgs, "systemctl "+p.Reload+" "+p.Service)...); err != nil {
			return fmt.Errorf("%s service : %w", p.Reload, err)
		}
	}
	log("déploiement distant terminé")
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s : %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".site-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
