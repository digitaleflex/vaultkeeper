package scanner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshHost représente un bloc "Host" de ~/.ssh/config.
type sshHost struct {
	Patterns      []string
	HostName      string
	User          string
	Port          string
	IdentityFiles []string
}

func (h sshHost) label() string {
	name := strings.Join(h.Patterns, ",")
	target := h.HostName
	if target == "" {
		target = name
	}
	if h.User != "" {
		target = h.User + "@" + target
	}
	if h.Port != "" && h.Port != "22" {
		target += ":" + h.Port
	}
	if h.HostName != "" && h.HostName != name {
		return fmt.Sprintf("%s (%s)", target, name)
	}
	return target
}

// parseSSHConfig lit un fichier ssh_config (gère Include de premier niveau).
func parseSSHConfig(path, sshDir string, depth int) []sshHost {
	data, err := readSmall(path, 1<<20)
	if err != nil || depth > 3 {
		return nil
	}
	var hosts []sshHost
	var cur *sshHost
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val := splitDirective(line)
		switch strings.ToLower(key) {
		case "host":
			if cur != nil {
				hosts = append(hosts, *cur)
			}
			cur = &sshHost{Patterns: strings.Fields(val)}
		case "match":
			if cur != nil {
				hosts = append(hosts, *cur)
			}
			cur = nil
		case "include":
			for _, inc := range strings.Fields(val) {
				inc = expandHome(inc, filepath.Dir(sshDir))
				if !filepath.IsAbs(inc) {
					inc = filepath.Join(sshDir, inc)
				}
				matches, _ := filepath.Glob(inc)
				for _, m := range matches {
					hosts = append(hosts, parseSSHConfig(m, sshDir, depth+1)...)
				}
			}
		case "hostname":
			if cur != nil {
				cur.HostName = val
			}
		case "user":
			if cur != nil {
				cur.User = val
			}
		case "port":
			if cur != nil {
				cur.Port = val
			}
		case "identityfile":
			if cur != nil {
				cur.IdentityFiles = append(cur.IdentityFiles, strings.Trim(val, `"`))
			}
		}
	}
	if cur != nil {
		hosts = append(hosts, *cur)
	}
	return hosts
}

func splitDirective(line string) (string, string) {
	line = strings.Replace(line, "=", " ", 1)
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func expandHome(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(home, p[2:])
	}
	p = strings.ReplaceAll(p, "%d", home)
	return p
}

func normPath(p string) string {
	p = filepath.Clean(p)
	return strings.ToLower(filepath.ToSlash(p))
}

var defaultKeyNames = map[string]bool{
	"id_rsa": true, "id_ecdsa": true, "id_ecdsa_sk": true,
	"id_ed25519": true, "id_ed25519_sk": true, "id_dsa": true,
}

// parseKnownHosts renvoie la liste des hôtes lisibles (non hachés).
func parseKnownHosts(path string) (hosts []string, hashed int) {
	data, err := readSmall(path, 8<<20)
	if err != nil {
		return nil, 0
	}
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		field := f[0]
		if strings.HasPrefix(field, "@") && len(f) > 2 {
			field = f[1]
		}
		if strings.HasPrefix(field, "|1|") {
			hashed++
			continue
		}
		for _, h := range strings.Split(field, ",") {
			if !seen[h] {
				seen[h] = true
				hosts = append(hosts, h)
			}
		}
	}
	sort.Strings(hosts)
	return hosts, hashed
}

type keyInfo struct {
	algo        string
	bits        int
	encrypted   bool
	fingerprint string
	comment     string
	putty       bool
}

func inspectPrivateKey(path string, data []byte) (*keyInfo, bool) {
	s := string(data)
	switch {
	case strings.HasPrefix(s, "PuTTY-User-Key-File-"):
		ki := &keyInfo{putty: true}
		sc := bufio.NewScanner(strings.NewReader(s))
		for sc.Scan() {
			l := sc.Text()
			if strings.HasPrefix(l, "PuTTY-User-Key-File-") {
				ki.algo = strings.TrimSpace(strings.SplitN(l, ":", 2)[1])
			}
			if strings.HasPrefix(l, "Encryption:") {
				ki.encrypted = strings.TrimSpace(strings.TrimPrefix(l, "Encryption:")) != "none"
			}
			if strings.HasPrefix(l, "Comment:") {
				ki.comment = strings.TrimSpace(strings.TrimPrefix(l, "Comment:"))
			}
		}
		return ki, true
	case strings.Contains(s, "-----BEGIN") && strings.Contains(s, "PRIVATE KEY-----"):
	default:
		return nil, false
	}

	ki := &keyInfo{}
	if strings.Contains(s, "ENCRYPTED") || strings.Contains(s, "Proc-Type: 4,ENCRYPTED") {
		ki.encrypted = true
	}
	key, err := ssh.ParseRawPrivateKey(data)
	var missing *ssh.PassphraseMissingError
	switch {
	case errors.As(err, &missing):
		ki.encrypted = true
		if missing.PublicKey != nil {
			ki.algo = missing.PublicKey.Type()
			ki.fingerprint = ssh.FingerprintSHA256(missing.PublicKey)
		}
	case err == nil:
		if signer, err2 := ssh.NewSignerFromKey(key); err2 == nil {
			pub := signer.PublicKey()
			ki.algo = pub.Type()
			ki.fingerprint = ssh.FingerprintSHA256(pub)
			ki.bits = keyBits(pub)
		}
	}
	// le .pub voisin donne souvent le commentaire (ex. user@machine)
	if pubData, err := readSmall(path+".pub", 64*1024); err == nil {
		if pub, comment, _, _, err := ssh.ParseAuthorizedKey(pubData); err == nil {
			ki.comment = comment
			if ki.fingerprint == "" {
				ki.fingerprint = ssh.FingerprintSHA256(pub)
				ki.algo = pub.Type()
				ki.bits = keyBits(pub)
			}
		}
	}
	if ki.algo == "" {
		ki.algo = "inconnu"
	}
	return ki, true
}

func keyBits(pub ssh.PublicKey) int {
	if ck, ok := pub.(ssh.CryptoPublicKey); ok {
		type sizer interface{ Size() int }
		if s, ok := ck.CryptoPublicKey().(sizer); ok {
			return s.Size() * 8
		}
	}
	return 0
}

func detectSSH(ctx *context) []Finding {
	home := ctx.opts.Home
	sshDir := filepath.Join(home, ".ssh")
	if _, ok := exists(sshDir); !ok {
		return nil
	}

	hosts := parseSSHConfig(filepath.Join(sshDir, "config"), sshDir, 0)
	// index : chemin de clé -> hôtes qui l'utilisent
	keyHosts := map[string][]string{}
	var hostsWithoutIdentity []string
	for _, h := range hosts {
		if len(h.IdentityFiles) == 0 {
			if !(len(h.Patterns) == 1 && h.Patterns[0] == "*") {
				hostsWithoutIdentity = append(hostsWithoutIdentity, h.label())
			}
			continue
		}
		for _, id := range h.IdentityFiles {
			p := expandHome(id, home)
			if !filepath.IsAbs(p) {
				p = filepath.Join(sshDir, p)
			}
			keyHosts[normPath(p)] = append(keyHosts[normPath(p)], h.label())
		}
	}

	var out []Finding
	entries, err := os.ReadDir(sshDir)
	ctx.errorf(err)
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		p := filepath.Join(sshDir, e.Name())
		data, err := readSmall(p, 64*1024)
		if err != nil {
			continue
		}
		ki, ok := inspectPrivateKey(p, data)
		if !ok {
			continue
		}
		st, _ := e.Info()
		f := Finding{
			ID:          makeID("ssh", p),
			Category:    CatSSH,
			Type:        "Clé SSH privée " + ki.algo,
			Path:        p,
			Fingerprint: ki.fingerprint,
			Severity:    SevMedium, // une clé privée non sauvegardée = risque de perte
		}
		if ki.putty {
			f.Type = "Clé PuTTY (.ppk) " + ki.algo
		}
		if st != nil {
			f.ModTime = st.ModTime()
		}
		f.Targets = keyHosts[normPath(p)]
		if defaultKeyNames[e.Name()] {
			f.Targets = append(f.Targets, "clé par défaut : tout hôte sans IdentityFile")
		}
		desc := "Clé " + ki.algo
		if ki.bits > 0 {
			desc += fmt.Sprintf(" %d bits", ki.bits)
		}
		if ki.comment != "" {
			desc += " — commentaire : " + ki.comment
		}
		f.Description = desc

		if !ki.encrypted {
			f.warn(SevHigh, "Aucune passphrase : quiconque copie ce fichier accède à vos serveurs")
		}
		if strings.Contains(ki.algo, "dss") {
			f.warn(SevHigh, "DSA est obsolète et refusé par les OpenSSH récents")
		}
		if strings.Contains(ki.algo, "rsa") && ki.bits > 0 && ki.bits < 3072 {
			f.warn(SevMedium, fmt.Sprintf("RSA %d bits : préférez ed25519", ki.bits))
		}
		if !f.ModTime.IsZero() && time.Since(f.ModTime) > 2*365*24*time.Hour {
			f.warn(SevLow, fmt.Sprintf("Clé âgée de %d ans : envisagez une rotation", int(time.Since(f.ModTime).Hours()/24/365)))
		}
		if len(f.Targets) == 0 {
			f.warn(SevLow, "Aucun serveur associé dans ~/.ssh/config : ajoutez une description à l'import")
		}
		out = append(out, f)
	}

	if known, hashed := parseKnownHosts(filepath.Join(sshDir, "known_hosts")); len(known) > 0 || hashed > 0 {
		f := Finding{
			ID:          makeID("ssh-known", sshDir),
			Category:    CatServer,
			Type:        "Inventaire serveurs (known_hosts)",
			Path:        filepath.Join(sshDir, "known_hosts"),
			Description: fmt.Sprintf("%d serveurs connus en clair, %d hachés", len(known), hashed),
			Targets:     known,
			Severity:    SevInfo,
		}
		out = append(out, f)
	}
	if len(hostsWithoutIdentity) > 0 {
		out = append(out, Finding{
			ID:          makeID("ssh-config-noid", sshDir),
			Category:    CatServer,
			Type:        "Hôtes ~/.ssh/config sans clé explicite",
			Path:        filepath.Join(sshDir, "config"),
			Description: "Ces hôtes utilisent la clé par défaut ou un mot de passe",
			Targets:     hostsWithoutIdentity,
			Severity:    SevInfo,
		})
	}
	return out
}
