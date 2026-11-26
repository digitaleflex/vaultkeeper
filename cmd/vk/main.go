// vk — CLI VaultKeeper : inventaire et import des secrets d'un poste de travail.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/digitaleflex/vaultkeeper/internal/report"
	"github.com/digitaleflex/vaultkeeper/internal/scanner"
)

var version = "dev"

const usage = `VaultKeeper — gestion sécurisée des secrets DevOps

Usage :
  vk scan [options]     Analyse ce poste et liste les secrets trouvés
  vk version            Affiche la version

Options de scan :
  -p, --path DIR        Dossier de projets à analyser pour les .env (répétable, défaut : dossier utilisateur)
  --depth N             Profondeur max de recherche des .env (défaut 6)
  --json FICHIER        Exporte le rapport en JSON
  --html FICHIER        Exporte le rapport en HTML
  --no-history          N'analyse pas les historiques de shell
  --no-color            Désactive les couleurs
  -v                    Affiche tous les détails

Aucune valeur secrète n'est jamais affichée ni écrite dans les rapports.
`

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "scan":
		os.Exit(runScan(os.Args[2:]))
	case "version", "--version":
		fmt.Println("vk", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "commande inconnue : %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.Usage = func() { fmt.Print(usage) }
	var paths multiFlag
	fs.Var(&paths, "path", "")
	fs.Var(&paths, "p", "")
	depth := fs.Int("depth", 6, "")
	jsonOut := fs.String("json", "", "")
	htmlOut := fs.String("html", "", "")
	noHist := fs.Bool("no-history", false, "")
	noColor := fs.Bool("no-color", false, "")
	verbose := fs.Bool("v", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	fmt.Fprintln(os.Stderr, "Analyse en cours…")
	r, err := scanner.Scan(scanner.Options{ProjectDirs: paths, MaxDepth: *depth, SkipHistory: *noHist})
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	report.WriteConsole(os.Stdout, r, !*noColor && os.Getenv("NO_COLOR") == "" && isTerminal(), *verbose)

	if *jsonOut != "" {
		if err := report.WriteJSON(r, *jsonOut); err != nil {
			fmt.Fprintln(os.Stderr, "erreur JSON :", err)
			return 1
		}
		fmt.Println("Rapport JSON :", *jsonOut)
	}
	if *htmlOut != "" {
		if err := report.WriteHTML(r, *htmlOut); err != nil {
			fmt.Fprintln(os.Stderr, "erreur HTML :", err)
			return 1
		}
		fmt.Println("Rapport HTML :", *htmlOut)
	}
	return 0
}

func isTerminal() bool {
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
