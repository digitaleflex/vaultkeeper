// Package report met en forme un rapport d'analyse (console, JSON, HTML).
package report

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/digitaleflex/vaultkeeper/internal/scanner"
)

// WriteJSON écrit le rapport en JSON (permissions 0600 : il révèle où sont les secrets).
func WriteJSON(r *scanner.Report, path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

//go:embed report.html.tmpl
var htmlTmpl string

func WriteHTML(r *scanner.Report, path string) error {
	t, err := template.New("r").Funcs(template.FuncMap{
		"sevClass": func(s scanner.Severity) string { return fmt.Sprintf("sev%d", s.Rank()) },
		"count":    func(sev string) int { return countSev(r, sev) },
	}).Parse(htmlTmpl)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, r)
}

func countSev(r *scanner.Report, sev string) int {
	n := 0
	for _, f := range r.Findings {
		if string(f.Severity) == sev {
			n++
		}
	}
	return n
}

// Console --------------------------------------------------------------------

type palette struct{ reset, bold, dim, red, yellow, blue, magenta, green string }

func colors(enabled bool) palette {
	if !enabled {
		return palette{}
	}
	return palette{"\033[0m", "\033[1m", "\033[2m", "\033[31m", "\033[33m", "\033[34m", "\033[35m", "\033[32m"}
}

func (p palette) sev(s scanner.Severity) string {
	switch s {
	case scanner.SevCritical:
		return p.bold + p.magenta
	case scanner.SevHigh:
		return p.red
	case scanner.SevMedium:
		return p.yellow
	case scanner.SevLow:
		return p.blue
	}
	return p.dim
}

// WriteConsole affiche un résumé lisible dans le terminal.
func WriteConsole(w io.Writer, r *scanner.Report, color bool, verbose bool) {
	c := colors(color)
	fmt.Fprintf(w, "\n%s🔐 VaultKeeper — inventaire des secrets%s\n", c.bold, c.reset)
	fmt.Fprintf(w, "%sMachine %s (%s) · utilisateur %s · %s%s\n\n", c.dim, r.Host, r.OS, r.User, r.ScannedAt.Format("2006-01-02 15:04 UTC"), c.reset)

	if len(r.Findings) == 0 {
		fmt.Fprintf(w, "%sAucun secret détecté.%s\n", c.green, c.reset)
		return
	}
	var lastCat scanner.Category
	for _, f := range r.Findings {
		if f.Category != lastCat {
			lastCat = f.Category
		}
		fmt.Fprintf(w, "%s[%-8s]%s %s%s%s  %s[%s]%s\n", c.sev(f.Severity), f.Severity, c.reset, c.bold, f.Type, c.reset, c.dim, f.Category, c.reset)
		fmt.Fprintf(w, "           %s\n", f.Path)
		if f.Description != "" {
			fmt.Fprintf(w, "           %s\n", f.Description)
		}
		if f.Fingerprint != "" {
			fmt.Fprintf(w, "           %sempreinte %s%s\n", c.dim, f.Fingerprint, c.reset)
		}
		writeList(w, c, "→", f.Targets, verbose)
		if verbose {
			writeList(w, c, "•", f.Keys, verbose)
		}
		for _, warn := range f.Warnings {
			fmt.Fprintf(w, "           %s⚠ %s%s\n", c.yellow, warn, c.reset)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "%sRésumé :%s %d élément(s) — %s%d critique(s)%s, %s%d élevé(s)%s, %s%d moyen(s)%s\n",
		c.bold, c.reset, len(r.Findings),
		c.sev(scanner.SevCritical), countSev(r, string(scanner.SevCritical)), c.reset,
		c.sev(scanner.SevHigh), countSev(r, string(scanner.SevHigh)), c.reset,
		c.sev(scanner.SevMedium), countSev(r, string(scanner.SevMedium)), c.reset)
	if len(r.Errors) > 0 && verbose {
		fmt.Fprintf(w, "%s%d erreur(s) de lecture ignorée(s) : %s%s\n", c.dim, len(r.Errors), strings.Join(r.Errors, "; "), c.reset)
	}
}

func writeList(w io.Writer, c palette, bullet string, items []string, all bool) {
	max := 5
	if all {
		max = len(items)
	}
	for i, it := range items {
		if i >= max {
			fmt.Fprintf(w, "           %s  … +%d (utilisez -v)%s\n", c.dim, len(items)-max, c.reset)
			break
		}
		fmt.Fprintf(w, "           %s %s\n", bullet, it)
	}
}
