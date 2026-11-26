package scanner

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Dossiers ignorés pendant la recherche des .env (lourds ou sans intérêt).
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, ".venv": true, "venv": true,
	"__pycache__": true, ".cache": true, ".npm": true, ".yarn": true, ".pnpm-store": true,
	"dist": true, "build": true, ".next": true, ".nuxt": true, "target": true,
	"Library": true, "AppData": true, "Application Data": true, ".Trash": true,
	"$Recycle.Bin": true, "go": true, ".rustup": true, ".cargo": true, ".gradle": true,
	".m2": true, ".nvm": true, ".local": true, "snap": true, ".vscode-server": true,
	"OneDrive": false,
}

var exampleSuffixes = []string{".example", ".sample", ".template", ".dist", ".tpl", ".defaults"}

var secretNameRe = regexp.MustCompile(`(?i)(secret|token|passw|pwd|api[_-]?key|private|credential|auth|dsn|_key$|access[_-]?key|database_url|db_url|conn(ection)?_?str|webhook|smtp)`)

// isEnvFile reconnaît .env, .env.local, .env.production, prod.env, etc.
func isEnvFile(name string) (env bool, example bool) {
	lower := strings.ToLower(name)
	if lower == ".env" || strings.HasPrefix(lower, ".env.") || strings.HasSuffix(lower, ".env") {
		for _, s := range exampleSuffixes {
			if strings.HasSuffix(lower, s) {
				return true, true
			}
		}
		return true, false
	}
	return false, false
}

// parseEnvKeys renvoie les noms de variables (jamais les valeurs).
func parseEnvKeys(data []byte) (keys []string, secretKeys []string, empty int) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k == "" || strings.ContainsAny(k, " \t") {
			continue
		}
		keys = append(keys, k)
		if v == "" {
			empty++
			continue
		}
		if secretNameRe.MatchString(k) {
			secretKeys = append(secretKeys, k)
		}
	}
	return
}

func detectEnvFiles(ctx *context) []Finding {
	var out []Finding
	for _, root := range ctx.opts.ProjectDirs {
		rootDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // dossier illisible : on continue
			}
			if d.IsDir() {
				if p != root && (skipDirs[d.Name()] || (strings.HasPrefix(d.Name(), ".") && d.Name() != ".config")) {
					return filepath.SkipDir
				}
				if strings.Count(p, string(os.PathSeparator))-rootDepth >= ctx.opts.MaxDepth {
					return filepath.SkipDir
				}
				return nil
			}
			env, example := isEnvFile(d.Name())
			if !env || example {
				return nil
			}
			data, err := readSmall(p, 1<<20)
			if err != nil {
				return nil
			}
			keys, secretKeys, _ := parseEnvKeys(data)
			if len(keys) == 0 {
				return nil
			}
			project := filepath.Base(filepath.Dir(p))
			f := Finding{
				ID:          makeID("env", p),
				Category:    CatEnv,
				Type:        "Fichier " + d.Name(),
				Path:        p,
				Description: fmt.Sprintf("Projet « %s » : %d variables dont %d probablement sensibles", project, len(keys), len(secretKeys)),
				Keys:        keys,
				Targets:     []string{"projet:" + project, "env:" + guessEnvironment(d.Name())},
				Severity:    SevLow,
			}
			if info, err := d.Info(); err == nil {
				f.ModTime = info.ModTime()
			}
			if len(secretKeys) > 0 {
				f.warn(SevMedium, "Secrets en clair sur disque : "+strings.Join(secretKeys, ", "))
			}
			if tracked, ok := gitTracked(p); ok && tracked {
				f.warn(SevCritical, "Ce fichier semble versionné dans Git (absent du .gitignore) : secrets potentiellement publiés")
			}
			out = append(out, f)
			return nil
		})
		ctx.errorf(err)
	}
	return out
}

func guessEnvironment(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "prod"):
		return "prod"
	case strings.Contains(n, "stag") || strings.Contains(n, "preprod"):
		return "staging"
	case strings.Contains(n, "test"):
		return "test"
	}
	return "dev"
}

// gitTracked fait une vérification légère : le fichier est dans un dépôt Git
// et aucun .gitignore du dépôt ne mentionne ".env".
func gitTracked(p string) (bool, bool) {
	dir := filepath.Dir(p)
	for i := 0; i < 8; i++ {
		if _, ok := exists(filepath.Join(dir, ".git")); ok {
			ignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
			if err != nil {
				return true, true
			}
			for _, l := range strings.Split(string(ignore), "\n") {
				l = strings.TrimSpace(l)
				if strings.Contains(l, ".env") && !strings.HasPrefix(l, "!") && !strings.HasPrefix(l, "#") {
					return false, true
				}
			}
			return true, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false, false
}
