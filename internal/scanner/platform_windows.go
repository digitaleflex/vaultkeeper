//go:build windows

package scanner

import (
	"fmt"
	"net/url"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// detectPlatform lit les sessions PuTTY et WinSCP enregistrées dans le registre.
func detectPlatform(ctx *context) []Finding {
	var out []Finding
	out = append(out, puttySessions()...)
	out = append(out, winscpSessions(ctx)...)
	return out
}

func subKeys(root registry.Key, path string) []string {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, _ := k.ReadSubKeyNames(-1)
	return names
}

func regString(k registry.Key, name string) string {
	v, _, err := k.GetStringValue(name)
	if err != nil {
		return ""
	}
	return v
}

func puttySessions() []Finding {
	const base = `Software\SimonTatham\PuTTY\Sessions`
	names := subKeys(registry.CURRENT_USER, base)
	if len(names) == 0 {
		return nil
	}
	f := Finding{
		ID:       makeID("putty-sessions"),
		Category: CatServer,
		Type:     "Sessions PuTTY",
		Path:     `HKCU\` + base,
		Severity: SevInfo,
	}
	keys := map[string]bool{}
	for _, n := range names {
		k, err := registry.OpenKey(registry.CURRENT_USER, base+`\`+n, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		host := regString(k, "HostName")
		user := regString(k, "UserName")
		ppk := regString(k, "PublicKeyFile")
		k.Close()
		if host == "" {
			continue
		}
		name, _ := url.QueryUnescape(n)
		label := host
		if user != "" {
			label = user + "@" + host
		}
		label += " (" + name + ")"
		if ppk != "" {
			label += " clé=" + filepath.Base(ppk)
			keys[ppk] = true
		}
		f.Targets = append(f.Targets, label)
	}
	f.Description = fmt.Sprintf("%d session(s) PuTTY", len(f.Targets))
	out := []Finding{f}
	// les .ppk référencés hors de ~/.ssh
	for p := range keys {
		data, err := readSmall(p, 64*1024)
		if err != nil {
			continue
		}
		if ki, ok := inspectPrivateKey(p, data); ok {
			kf := Finding{
				ID: makeID("ssh", p), Category: CatSSH, Type: "Clé PuTTY (.ppk) " + ki.algo,
				Path: p, Description: "Référencée par une session PuTTY", Severity: SevMedium,
			}
			if !ki.encrypted {
				kf.warn(SevHigh, "Aucune passphrase sur la clé PuTTY")
			}
			out = append(out, kf)
		}
	}
	return out
}

func winscpSessions(ctx *context) []Finding {
	const base = `Software\Martin Prikryl\WinSCP 2\Sessions`
	names := subKeys(registry.CURRENT_USER, base)
	if len(names) == 0 {
		return nil
	}
	f := Finding{
		ID: makeID("winscp-sessions"), Category: CatServer, Type: "Sessions WinSCP",
		Path: `HKCU\` + base, Severity: SevLow,
	}
	withPass := 0
	for _, n := range names {
		k, err := registry.OpenKey(registry.CURRENT_USER, base+`\`+n, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		host := regString(k, "HostName")
		user := regString(k, "UserName")
		if regString(k, "Password") != "" {
			withPass++
		}
		k.Close()
		if host == "" {
			continue
		}
		name, _ := url.QueryUnescape(n)
		f.Targets = append(f.Targets, fmt.Sprintf("%s@%s (%s)", user, host, name))
	}
	f.Description = fmt.Sprintf("%d session(s) WinSCP", len(f.Targets))
	if withPass > 0 {
		f.warn(SevCritical, fmt.Sprintf("%d mot(s) de passe WinSCP stocké(s) avec un simple brouillage réversible", withPass))
	}
	return []Finding{f}
}
