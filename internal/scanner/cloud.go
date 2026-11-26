package scanner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

func newFileFinding(cat Category, typ, path, desc string) Finding {
	f := Finding{ID: makeID(string(cat), typ, path), Category: cat, Type: typ, Path: path, Description: desc, Severity: SevMedium}
	if st, ok := exists(path); ok {
		f.ModTime = st.ModTime()
	}
	return f
}

// iniSections renvoie les noms de sections [xxx] d'un fichier INI.
func iniSections(data []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			out = append(out, strings.TrimSpace(l[1:len(l)-1]))
		}
	}
	return out
}

func detectCloud(ctx *context) []Finding {
	home := ctx.opts.Home
	cfg := appData(home)
	var out []Finding

	// AWS
	if p := filepath.Join(home, ".aws", "credentials"); true {
		if data, err := readSmall(p, 1<<20); err == nil {
			profiles := iniSections(data)
			f := newFileFinding(CatCloud, "AWS credentials", p, fmt.Sprintf("%d profil(s) AWS", len(profiles)))
			f.Keys = profiles
			if bytes.Contains(data, []byte("aws_access_key_id")) && !bytes.Contains(data, []byte("aws_session_token")) {
				f.warn(SevHigh, "Clés d'accès longue durée : préférez AWS SSO / IAM Identity Center")
			}
			out = append(out, f)
		}
	}

	// Kubernetes
	if p := filepath.Join(home, ".kube", "config"); true {
		if data, err := readSmall(p, 4<<20); err == nil {
			f := newFileFinding(CatCloud, "Kubeconfig", p, "Accès clusters Kubernetes")
			f.Targets = yamlValues(data, "server")
			f.Keys = yamlValues(data, "name")
			if bytes.Contains(data, []byte("client-key-data")) || bytes.Contains(data, []byte("token:")) {
				f.warn(SevHigh, "Contient des identifiants clients embarqués (certificat ou token)")
			}
			out = append(out, f)
		}
	}

	// Google Cloud
	for _, p := range []string{
		filepath.Join(cfg, "gcloud", "application_default_credentials.json"),
		filepath.Join(cfg, "gcloud", "credentials.db"),
		filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"),
	} {
		if _, ok := exists(p); ok {
			out = append(out, newFileFinding(CatCloud, "Google Cloud credentials", p, "Identifiants gcloud"))
		}
	}

	// Azure
	for _, p := range []string{filepath.Join(home, ".azure", "msal_token_cache.json"), filepath.Join(home, ".azure", "accessTokens.json"), filepath.Join(home, ".azure", "msal_token_cache.bin")} {
		if _, ok := exists(p); ok {
			out = append(out, newFileFinding(CatCloud, "Azure CLI tokens", p, "Jetons Azure CLI en cache"))
		}
	}

	// Fichiers JSON de compte de service GCP éparpillés dans le dossier utilisateur (niveau 1-2)
	out = append(out, findServiceAccountJSON(home)...)
	return out
}

var yamlKVRe = regexp.MustCompile(`^\s*-?\s*([A-Za-z_-]+):\s*(.+?)\s*$`)

// yamlValues extrait (sans dépendance YAML) les valeurs d'une clé donnée.
func yamlValues(data []byte, key string) []string {
	seen := map[string]bool{}
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	for sc.Scan() {
		m := yamlKVRe.FindStringSubmatch(sc.Text())
		if m != nil && m[1] == key {
			v := strings.Trim(m[2], `"'`)
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return out
}

func findServiceAccountJSON(home string) []Finding {
	var out []Finding
	dirs := []string{home, filepath.Join(home, "Downloads"), filepath.Join(home, "Téléchargements"), filepath.Join(home, "Desktop"), filepath.Join(home, "Documents")}
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			p := filepath.Join(d, e.Name())
			data, err := readSmall(p, 64*1024)
			if err != nil {
				continue
			}
			if bytes.Contains(data, []byte(`"type": "service_account"`)) || bytes.Contains(data, []byte(`"type":"service_account"`)) {
				var sa struct {
					ProjectID   string `json:"project_id"`
					ClientEmail string `json:"client_email"`
				}
				_ = json.Unmarshal(data, &sa)
				f := newFileFinding(CatCloud, "Clé de compte de service GCP", p, "Compte "+sa.ClientEmail)
				f.Targets = []string{"projet:" + sa.ProjectID}
				f.warn(SevHigh, "Clé JSON de compte de service traînant dans un dossier personnel")
				out = append(out, f)
			}
		}
	}
	return out
}

func detectDevTools(ctx *context) []Finding {
	home := ctx.opts.Home
	cfg := appData(home)
	var out []Finding

	// Docker
	if p := filepath.Join(home, ".docker", "config.json"); true {
		if data, err := readSmall(p, 1<<20); err == nil {
			var dc struct {
				Auths      map[string]struct{ Auth string } `json:"auths"`
				CredsStore string                           `json:"credsStore"`
			}
			if json.Unmarshal(data, &dc) == nil && len(dc.Auths) > 0 {
				var regs []string
				inline := false
				for r, a := range dc.Auths {
					regs = append(regs, r)
					if a.Auth != "" {
						inline = true
					}
				}
				sort.Strings(regs)
				f := newFileFinding(CatDevTool, "Docker registries", p, fmt.Sprintf("%d registre(s) authentifié(s)", len(regs)))
				f.Targets = regs
				f.Severity = SevLow
				if inline {
					f.warn(SevHigh, "Identifiants stockés en base64 dans le fichier (pas de credsStore)")
				}
				out = append(out, f)
			}
		}
	}

	// npm
	for _, p := range []string{filepath.Join(home, ".npmrc")} {
		if data, err := readSmall(p, 1<<20); err == nil && (bytes.Contains(data, []byte("_authToken")) || bytes.Contains(data, []byte("_auth="))) {
			f := newFileFinding(CatDevTool, "npm token", p, "Jeton de registre npm")
			f.Targets = registryHosts(data)
			out = append(out, f)
		}
	}

	// PyPI
	if p := filepath.Join(home, ".pypirc"); true {
		if data, err := readSmall(p, 1<<20); err == nil && bytes.Contains(data, []byte("password")) {
			f := newFileFinding(CatDevTool, "PyPI credentials", p, "Identifiants de publication PyPI")
			f.Keys = iniSections(data)
			out = append(out, f)
		}
	}

	// Git credentials (stockage en clair)
	for _, p := range []string{filepath.Join(home, ".git-credentials"), filepath.Join(cfg, "git", "credentials")} {
		if data, err := readSmall(p, 1<<20); err == nil {
			f := newFileFinding(CatDevTool, "Git credentials", p, "Identifiants Git stockés EN CLAIR (credential.helper=store)")
			for _, l := range strings.Split(string(data), "\n") {
				if u, err := url.Parse(strings.TrimSpace(l)); err == nil && u.Host != "" {
					f.Targets = append(f.Targets, u.Scheme+"://"+u.User.Username()+"@"+u.Host)
				}
			}
			f.warn(SevHigh, "Mots de passe / tokens Git en clair : utilisez le Git Credential Manager")
			out = append(out, f)
		}
	}

	// GitHub CLI
	if p := filepath.Join(cfg, "gh", "hosts.yml"); true {
		if data, err := readSmall(p, 1<<20); err == nil && bytes.Contains(data, []byte("oauth_token")) {
			f := newFileFinding(CatDevTool, "GitHub CLI token", p, "Jeton gh en clair (pas de trousseau système)")
			out = append(out, f)
		}
	}

	// .netrc
	for _, name := range []string{".netrc", "_netrc"} {
		p := filepath.Join(home, name)
		if data, err := readSmall(p, 1<<20); err == nil {
			f := newFileFinding(CatDevTool, "netrc", p, "Identifiants HTTP/FTP en clair")
			fields := strings.Fields(string(data))
			for i := 0; i+1 < len(fields); i++ {
				if fields[i] == "machine" {
					f.Targets = append(f.Targets, fields[i+1])
				}
			}
			f.warn(SevHigh, "Mots de passe en clair")
			out = append(out, f)
		}
	}

	// Terraform
	tf := filepath.Join(home, ".terraform.d", "credentials.tfrc.json")
	if runtime.GOOS == "windows" {
		tf = filepath.Join(cfg, "terraform.d", "credentials.tfrc.json")
	}
	if data, err := readSmall(tf, 1<<20); err == nil {
		var t struct {
			Credentials map[string]any `json:"credentials"`
		}
		f := newFileFinding(CatDevTool, "Terraform Cloud token", tf, "Jetons Terraform")
		if json.Unmarshal(data, &t) == nil {
			for h := range t.Credentials {
				f.Targets = append(f.Targets, h)
			}
		}
		out = append(out, f)
	}
	return out
}

func registryHosts(npmrc []byte) []string {
	var out []string
	for _, l := range strings.Split(string(npmrc), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "//") {
			if host, _, ok := strings.Cut(l[2:], ":_auth"); ok {
				out = append(out, strings.TrimSuffix(host, "/"))
			}
		}
	}
	return out
}
