package scanner

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Motifs de secrets tapés en ligne de commande. On ne rapporte QUE le nom du
// motif et les numéros de ligne, jamais le contenu de la ligne.
var historyPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"sshpass -p", regexp.MustCompile(`sshpass\s+-p\s*\S+`)},
	{"mysql -p<motdepasse>", regexp.MustCompile(`\bmysql(dump)?\b.*\s-p[^\s-]\S*`)},
	{"PGPASSWORD=", regexp.MustCompile(`PGPASSWORD=\S+`)},
	{"URL avec identifiants (user:pass@)", regexp.MustCompile(`[a-z][a-z0-9+.-]*://[^\s/:@]+:[^\s/@]+@`)},
	{"curl -u user:pass", regexp.MustCompile(`curl\b.*\s(-u|--user)\s+\S+:\S+`)},
	{"export *SECRET/TOKEN/PASSWORD=", regexp.MustCompile(`(?i)(export\s+|\$env:|set\s+)\w*(secret|token|passw|api_?key)\w*\s*=\s*\S+`)},
	{"docker login -p", regexp.MustCompile(`docker\s+login\b.*\s(-p|--password)\s+\S+`)},
	{"Clé AWS (AKIA…)", regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"Token GitHub (ghp_/github_pat_)", regexp.MustCompile(`\b(ghp|gho|ghu|ghs)_[A-Za-z0-9]{36}\b|github_pat_[A-Za-z0-9_]{50,}`)},
	{"Token GitLab (glpat-)", regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20}\b`)},
	{"Clé Stripe (sk_live_)", regexp.MustCompile(`\bsk_live_[A-Za-z0-9]{20,}`)},
	{"Token Slack (xox…)", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}`)},
	{"Clé OpenAI/Anthropic (sk-…)", regexp.MustCompile(`\bsk-(ant-|proj-)?[A-Za-z0-9_-]{20,}`)},
	{"ConvertTo-SecureString -AsPlainText", regexp.MustCompile(`(?i)ConvertTo-SecureString\s.*-AsPlainText`)},
	{"--password=", regexp.MustCompile(`--password[= ]\S+`)},
}

type historyHit struct {
	pattern string
	lines   []int
}

func scanHistory(data []byte) []historyHit {
	hits := map[string][]int{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		for _, p := range historyPatterns {
			if p.re.MatchString(line) {
				hits[p.name] = append(hits[p.name], n)
			}
		}
	}
	var out []historyHit
	for k, v := range hits {
		out = append(out, historyHit{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pattern < out[j].pattern })
	return out
}

func historyFiles(home string) []string {
	files := []string{
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zsh_history"),
		filepath.Join(home, ".zhistory"),
		filepath.Join(home, ".local", "share", "fish", "fish_history"),
		filepath.Join(home, ".mysql_history"),
		filepath.Join(home, ".psql_history"),
		filepath.Join(home, ".python_history"),
		filepath.Join(home, ".node_repl_history"),
	}
	ps := filepath.Join("Microsoft", "Windows", "PowerShell", "PSReadLine", "ConsoleHost_history.txt")
	if runtime.GOOS == "windows" {
		files = append(files, filepath.Join(appData(home), ps))
	} else {
		files = append(files, filepath.Join(home, ".local", "share", "powershell", "PSReadLine", "ConsoleHost_history.txt"))
	}
	return files
}

func detectHistory(ctx *context) []Finding {
	var out []Finding
	for _, p := range historyFiles(ctx.opts.Home) {
		data, err := readSmall(p, 64<<20)
		if err != nil {
			continue
		}
		hits := scanHistory(data)
		if len(hits) == 0 {
			continue
		}
		f := newFileFinding(CatLeak, "Secrets dans l'historique shell", p, "")
		total := 0
		for _, h := range hits {
			total += len(h.lines)
			lines := h.lines
			suffix := ""
			if len(lines) > 10 {
				lines, suffix = lines[:10], fmt.Sprintf(" … (+%d)", len(h.lines)-10)
			}
			f.Keys = append(f.Keys, fmt.Sprintf("%s → lignes %s%s", h.pattern, joinInts(lines), suffix))
		}
		f.Description = fmt.Sprintf("%d commande(s) contenant probablement un secret en clair", total)
		f.warn(SevHigh, "Nettoyez ces lignes de l'historique et considérez ces secrets comme exposés (rotation)")
		out = append(out, f)
	}
	return out
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ",")
}
