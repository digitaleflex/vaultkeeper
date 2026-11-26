package scanner

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

type fzServer struct {
	Host     string `xml:"Host"`
	Port     string `xml:"Port"`
	Protocol int    `xml:"Protocol"`
	User     string `xml:"User"`
	Pass     struct {
		Encoding string `xml:"encoding,attr"`
		Value    string `xml:",chardata"`
	} `xml:"Pass"`
	Name string `xml:"Name"`
}

// collecte récursive : les sites peuvent être rangés dans des <Folder>.
type fzNode struct {
	Servers []fzServer `xml:"Server"`
	Folders []fzNode   `xml:"Folder"`
}

type fzFile struct {
	Servers       fzNode `xml:"Servers"`
	RecentServers fzNode `xml:"RecentServers"`
}

func (n fzNode) all() []fzServer {
	out := append([]fzServer{}, n.Servers...)
	for _, f := range n.Folders {
		out = append(out, f.all()...)
	}
	return out
}

func sameAsSeen(st os.FileInfo, dirs []string, seen map[string]bool) bool {
	for _, d := range dirs {
		if !seen[normPath(d)] {
			continue
		}
		if other, ok := exists(d); ok && os.SameFile(st, other) {
			return true
		}
	}
	return false
}

func detectFileZilla(ctx *context) []Finding {
	dirs := []string{
		filepath.Join(appData(ctx.opts.Home), "FileZilla"), // Windows
		filepath.Join(appData(ctx.opts.Home), "filezilla"), // Linux (~/.config/filezilla)
		filepath.Join(ctx.opts.Home, ".filezilla"),         // macOS / anciennes versions
	}
	seenDir := map[string]bool{}
	var out []Finding
	for _, d := range dirs {
		// système de fichiers insensible à la casse (Windows/macOS) : éviter les doublons
		if st, ok := exists(d); !ok || seenDir[normPath(d)] || (len(seenDir) > 0 && sameAsSeen(st, dirs, seenDir)) {
			continue
		}
		seenDir[normPath(d)] = true
		for _, name := range []string{"sitemanager.xml", "recentservers.xml"} {
			p := filepath.Join(d, name)
			data, err := readSmall(p, 8<<20)
			if err != nil {
				continue
			}
			var fz fzFile
			if err := xml.Unmarshal(data, &fz); err != nil {
				ctx.errorf(fmt.Errorf("filezilla %s: %w", p, err))
				continue
			}
			servers := append(fz.Servers.all(), fz.RecentServers.all()...)
			if len(servers) == 0 {
				continue
			}
			f := newFileFinding(CatServer, "FileZilla "+name, p, fmt.Sprintf("%d serveur(s) FTP/SFTP enregistré(s)", len(servers)))
			f.Severity = SevLow
			withPass := 0
			for _, s := range servers {
				proto := "ftp"
				if s.Protocol == 1 {
					proto = "sftp"
				}
				label := fmt.Sprintf("%s://%s@%s:%s", proto, s.User, s.Host, s.Port)
				if s.Name != "" {
					label += " (" + s.Name + ")"
				}
				f.Targets = append(f.Targets, label)
				if s.Pass.Value != "" && s.Pass.Encoding != "crypt" {
					withPass++
				}
			}
			if withPass > 0 {
				f.warn(SevCritical, fmt.Sprintf("%d mot(s) de passe stocké(s) en base64 (lisibles par n'importe qui) : importez-les puis activez le mot de passe maître FileZilla", withPass))
			}
			out = append(out, f)
		}
	}
	return out
}
