// vk — CLI VaultKeeper : inventaire et import des secrets d'un poste de travail.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/digitaleflex/vaultkeeper/internal/report"
	"github.com/digitaleflex/vaultkeeper/internal/scanner"
	"github.com/digitaleflex/vaultkeeper/internal/server"
	"github.com/digitaleflex/vaultkeeper/internal/updater"
	"github.com/digitaleflex/vaultkeeper/web"
)

var version = "dev"

const usage = `VaultKeeper — gestion sécurisée des secrets DevOps

Usage :
  vk ui [options]       Ouvre l'interface graphique dans le navigateur
  vk scan [options]     Analyse ce poste et liste les secrets trouvés
  vk update             Met à jour vk vers la dernière version
  vk update --check     Vérifie seulement si une mise à jour existe
  vk version            Affiche la version

Options de ui :
  --port N              Port local (défaut : aléatoire)
  --no-open             N'ouvre pas le navigateur automatiquement

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
	updater.CleanupOld()
	// sans argument (ex. double-clic sur vk.exe) : on ouvre l'interface graphique
	if len(os.Args) < 2 {
		os.Exit(runUI(nil))
	}
	switch os.Args[1] {
	case "ui":
		os.Exit(runUI(os.Args[2:]))
	case "scan":
		os.Exit(runScan(os.Args[2:]))
	case "update", "upgrade":
		os.Exit(runUpdate(os.Args[2:]))
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

func runUI(args []string) int {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.Usage = func() { fmt.Print(usage) }
	port := fs.Int("port", 0, "")
	noOpen := fs.Bool("no-open", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	srv, err := server.New(version, web.Dist())
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	ln, url, err := srv.Listen(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	fmt.Println("🔐 VaultKeeper — interface graphique")
	fmt.Println("   Adresse (privée, ne la partagez pas) :")
	fmt.Println("   " + url)
	fmt.Println("   Ctrl+C pour arrêter.")
	if !*noOpen {
		if err := server.OpenBrowser(url); err != nil {
			fmt.Println("   (ouvrez l'adresse ci-dessus dans votre navigateur)")
		}
	}
	if err := srv.Serve(ln); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	return 0
}

func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.Usage = func() { fmt.Print(usage) }
	check := fs.Bool("check", false, "")
	force := fs.Bool("force", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	c := updater.New()
	fmt.Println("Version installée :", version)
	rel, err := c.Latest(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	fmt.Println("Dernière version  :", strings.TrimPrefix(rel.Tag, "v"))
	if !updater.Newer(rel.Tag, version) && !*force {
		fmt.Println("✔ vk est à jour.")
		return 0
	}
	if *check {
		fmt.Println("→ Mise à jour disponible : lancez `vk update`.")
		fmt.Println("  Notes de version :", rel.HTMLURL)
		return 0
	}

	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	fmt.Println("Téléchargement de", c.AssetName(rel.Tag), "…")
	bin, err := c.Download(ctx, rel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		return 1
	}
	fmt.Println("✔ Somme SHA-256 vérifiée")
	if err := updater.Replace(exe, bin); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		if strings.HasPrefix(exe, "/usr/") {
			fmt.Fprintln(os.Stderr, "  Installé via .deb : `sudo vk update`, ou téléchargez le nouveau .deb puis `sudo apt install ./vk_X.Y.Z_amd64.deb`.")
		}
		return 1
	}
	fmt.Printf("✔ vk mis à jour en %s (%s)\n", strings.TrimPrefix(rel.Tag, "v"), exe)
	return 0
}
