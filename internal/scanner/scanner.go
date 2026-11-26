package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

// Options configure une analyse.
type Options struct {
	Home        string   // dossier utilisateur (défaut : os.UserHomeDir)
	ProjectDirs []string // dossiers où chercher les .env (défaut : Home)
	MaxDepth    int      // profondeur max de recherche des .env
	SkipHistory bool
}

// Report est le résultat complet d'une analyse.
type Report struct {
	Host      string    `json:"host"`
	OS        string    `json:"os"`
	User      string    `json:"user"`
	ScannedAt time.Time `json:"scanned_at"`
	Findings  []Finding `json:"findings"`
	Errors    []string  `json:"errors,omitempty"`
}

type detector func(ctx *context) []Finding

type context struct {
	opts Options
	errs []string
}

func (c *context) errorf(err error) {
	if err != nil {
		c.errs = append(c.errs, err.Error())
	}
}

// Scan lance tous les détecteurs et retourne le rapport.
func Scan(opts Options) (*Report, error) {
	if opts.Home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		opts.Home = h
	}
	if len(opts.ProjectDirs) == 0 {
		opts.ProjectDirs = []string{opts.Home}
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 6
	}

	ctx := &context{opts: opts}
	detectors := []detector{detectSSH, detectEnvFiles, detectCloud, detectDevTools, detectFileZilla, detectPlatform}
	if !opts.SkipHistory {
		detectors = append(detectors, detectHistory)
	}

	var all []Finding
	seen := map[string]bool{}
	for _, d := range detectors {
		for _, f := range d(ctx) {
			if seen[f.ID] {
				continue
			}
			seen[f.ID] = true
			if f.Severity == "" {
				f.Severity = SevInfo
			}
			all = append(all, f)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Severity.Rank() != all[j].Severity.Rank() {
			return all[i].Severity.Rank() > all[j].Severity.Rank()
		}
		if all[i].Category != all[j].Category {
			return all[i].Category < all[j].Category
		}
		return all[i].Path < all[j].Path
	})

	host, _ := os.Hostname()
	return &Report{
		Host:      host,
		OS:        runtime.GOOS + "/" + runtime.GOARCH,
		User:      filepath.Base(opts.Home),
		ScannedAt: time.Now().UTC(),
		Findings:  all,
		Errors:    ctx.errs,
	}, nil
}

// helpers -------------------------------------------------------------------

func exists(p string) (os.FileInfo, bool) {
	st, err := os.Stat(p)
	return st, err == nil
}

func readSmall(p string, max int64) ([]byte, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if st.Size() > max {
		return nil, os.ErrInvalid
	}
	return os.ReadFile(p)
}

// appData renvoie %APPDATA% sous Windows, ~/.config ailleurs.
func appData(home string) string {
	if runtime.GOOS == "windows" {
		if v := os.Getenv("APPDATA"); v != "" {
			return v
		}
		return filepath.Join(home, "AppData", "Roaming")
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(home, ".config")
}
