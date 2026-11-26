package scanner

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// fakeHome construit un dossier utilisateur factice rempli de secrets de test.
func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	w := func(rel, content string) {
		p := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// clé ed25519 sans passphrase
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "dev@laptop")
	if err != nil {
		t.Fatal(err)
	}
	w(".ssh/id_prod", string(pem.EncodeToMemory(block)))
	pub, _ := ssh.NewPublicKey(priv.Public())
	w(".ssh/id_prod.pub", strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))+" dev@laptop\n")

	// clé chiffrée
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	block2, err := ssh.MarshalPrivateKeyWithPassphrase(priv2, "ci", []byte("passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	w(".ssh/id_ed25519", string(pem.EncodeToMemory(block2)))

	w(".ssh/config", `Host prod-api
  HostName 51.68.1.10
  User deploy
  IdentityFile ~/.ssh/id_prod

Host db
  HostName 51.68.1.20
  Port 2222
  IdentityFile ~/.ssh/id_prod

Host legacy
  HostName 10.0.0.5
`)
	w(".ssh/known_hosts", "51.68.1.10 ssh-ed25519 AAAA\n|1|abc=|def= ssh-rsa AAAA\ngithub.com,140.82.121.4 ssh-ed25519 AAAA\n")

	w("code/shop/.env", "DATABASE_URL=postgres://u:p@db/shop\nSTRIPE_SECRET_KEY=sk_test_x\nAPP_NAME=shop\nEMPTY_TOKEN=\n")
	w("code/shop/.env.example", "DATABASE_URL=\n")
	w("code/shop/.env.production", "JWT_SECRET=xyz\n")
	w("code/shop/node_modules/pkg/.env", "IGNORED_SECRET=1\n")
	os.MkdirAll(filepath.Join(home, "code/shop/.git"), 0o700)
	w("code/shop/.gitignore", "node_modules\n")

	w(".aws/credentials", "[default]\naws_access_key_id=AKIAXXXXXXXXXXXXXXXX\naws_secret_access_key=y\n[client-x]\naws_access_key_id=z\n")
	w(".kube/config", "clusters:\n- cluster:\n    server: https://k8s.example.com:6443\n  name: prod\nusers:\n- name: admin\n  user:\n    token: abc\n")
	w(".docker/config.json", `{"auths":{"registry.gitlab.com":{"auth":"dXNlcjpwYXNz"}}}`)
	w(".git-credentials", "https://bob:ghp_xxx@github.com\n")
	w(".config/filezilla/sitemanager.xml", `<?xml version="1.0"?><FileZilla3><Servers><Folder><Server><Host>ftp.client.fr</Host><Port>21</Port><Protocol>0</Protocol><User>client</User><Pass encoding="base64">c2VjcmV0</Pass><Name>Client FTP</Name></Server></Folder></Servers></FileZilla3>`)
	w(".bash_history", "ls\nsshpass -p 'hunter2' ssh root@x\nexport API_TOKEN=abcd\nmysql -u root -pS3cret db\ngit status\n")
	return home
}

func findByType(r *Report, prefix string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if strings.HasPrefix(f.Type, prefix) {
			out = append(out, f)
		}
	}
	return out
}

func TestScanFullHome(t *testing.T) {
	home := fakeHome(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, ".config"))
	r, err := Scan(Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}

	keys := findByType(r, "Clé SSH")
	if len(keys) != 2 {
		t.Fatalf("attendu 2 clés SSH, obtenu %d", len(keys))
	}
	for _, k := range keys {
		switch filepath.Base(k.Path) {
		case "id_prod":
			if k.Severity != SevHigh {
				t.Errorf("id_prod sans passphrase devrait être 'élevé', obtenu %s", k.Severity)
			}
			if len(k.Targets) != 2 || !strings.Contains(strings.Join(k.Targets, " "), "deploy@51.68.1.10") {
				t.Errorf("serveurs de id_prod incorrects : %v", k.Targets)
			}
			if !strings.Contains(k.Description, "dev@laptop") {
				t.Errorf("commentaire manquant : %s", k.Description)
			}
			if !strings.HasPrefix(k.Fingerprint, "SHA256:") {
				t.Errorf("empreinte manquante")
			}
		case "id_ed25519":
			for _, w := range k.Warnings {
				if strings.Contains(w, "passphrase") {
					t.Errorf("clé chiffrée signalée sans passphrase")
				}
			}
		}
	}

	envs := findByType(r, "Fichier .env")
	if len(envs) != 2 {
		t.Fatalf("attendu 2 fichiers .env (hors example et node_modules), obtenu %d", len(envs))
	}
	for _, e := range envs {
		if e.Severity != SevCritical {
			t.Errorf("%s : .env non ignoré par git devrait être critique", e.Path)
		}
	}

	for _, typ := range []string{"AWS credentials", "Kubeconfig", "Docker registries", "Git credentials", "FileZilla sitemanager.xml", "Secrets dans l'historique shell", "Inventaire serveurs"} {
		if len(findByType(r, typ)) == 0 {
			t.Errorf("détection manquante : %s", typ)
		}
	}
}

func TestNoSecretValuesInReport(t *testing.T) {
	home := fakeHome(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, ".config"))
	r, _ := Scan(Options{Home: home})
	forbidden := []string{"hunter2", "S3cret", "sk_test_x", "xyz", "dXNlcjpwYXNz", "c2VjcmV0", "ghp_xxx", "abcd", "postgres://u:p"}
	for _, f := range r.Findings {
		blob := strings.Join(append(append([]string{f.Description, f.Type}, f.Targets...), f.Keys...), "|")
		for _, s := range forbidden {
			if strings.Contains(blob, s) {
				t.Errorf("valeur secrète %q divulguée dans %s : %s", s, f.Type, blob)
			}
		}
	}
}

func TestIsEnvFile(t *testing.T) {
	cases := map[string][2]bool{
		".env": {true, false}, ".env.local": {true, false}, "prod.env": {true, false},
		".env.example": {true, true}, ".env.sample": {true, true}, "env.go": {false, false}, ".envrc": {false, false},
	}
	for name, want := range cases {
		env, ex := isEnvFile(name)
		if env != want[0] || ex != want[1] {
			t.Errorf("%s : obtenu (%v,%v), attendu %v", name, env, ex, want)
		}
	}
}
