package updater

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "0.1.0", true},
		{"v0.1.0", "0.1.0", false},
		{"v0.1.0", "0.2.0", false},
		{"v1.0.0", "0.9.9", true},
		{"v0.10.0", "0.9.0", true},
		{"v0.1.1", "dev", true},
		{"v0.1.0", "5d76a4f", true},
		{"pas-une-version", "0.1.0", false},
		{"v0.2.0-rc1", "0.1.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, attendu %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	c := &Client{GOOS: "darwin", GOARCH: "arm64"}
	if got := c.AssetName("v1.2.3"); got != "vk_1.2.3_macos_arm64.zip" {
		t.Fatal(got)
	}
	c = &Client{GOOS: "windows", GOARCH: "amd64"}
	if got := c.AssetName("v1.2.3"); got != "vk_1.2.3_windows_amd64.zip" {
		t.Fatal(got)
	}
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("vk_9.9.9_windows_amd64/" + name)
	w.Write(content)
	zw.Close()
	return buf.Bytes()
}

// fakeGitHub simule l'API GitHub avec une release et ses fichiers.
func fakeGitHub(t *testing.T, zipData []byte, sums string, token string) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.NotFound(w, r) // comportement GitHub pour un dépôt privé
			return
		}
		switch r.URL.Path {
		case "/repos/o/r/releases/latest":
			json.NewEncoder(w).Encode(Release{Tag: "v9.9.9", Assets: []Asset{
				{Name: "vk_9.9.9_windows_amd64.zip", URL: srv.URL + "/assets/1"},
				{Name: "SHA256SUMS", URL: srv.URL + "/assets/2"},
			}})
		case "/assets/1":
			if r.Header.Get("Accept") != "application/octet-stream" {
				t.Errorf("Accept incorrect : %s", r.Header.Get("Accept"))
			}
			w.Write(zipData)
		case "/assets/2":
			w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func client(srv *httptest.Server, token string) *Client {
	return &Client{APIBase: srv.URL, Repo: "o/r", Token: token, HTTP: srv.Client(), GOOS: "windows", GOARCH: "amd64"}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	bin := []byte("NOUVEAU BINAIRE")
	z := makeZip(t, "vk.exe", bin)
	sum := sha256.Sum256(z)
	good := fmt.Sprintf("%s  vk_9.9.9_windows_amd64.zip\n", hex.EncodeToString(sum[:]))

	srv := fakeGitHub(t, z, good, "secret")
	c := client(srv, "secret")
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Download(context.Background(), rel)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bin) {
		t.Fatalf("binaire extrait incorrect : %q", got)
	}

	// somme altérée -> refus
	bad := strings.Replace(good, good[:4], "0000", 1)
	srv2 := fakeGitHub(t, z, bad, "")
	if _, err := client(srv2, "").Download(context.Background(), rel2(t, srv2)); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("une somme invalide doit être refusée, obtenu %v", err)
	}
}

func rel2(t *testing.T, srv *httptest.Server) *Release {
	r, err := client(srv, "").Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPrivateRepoWithoutToken(t *testing.T) {
	srv := fakeGitHub(t, nil, "", "secret")
	_, err := client(srv, "").Latest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "GH_TOKEN") {
		t.Fatalf("message d'aide attendu, obtenu %v", err)
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "vk")
	os.WriteFile(exe, []byte("ancien"), 0o755)
	if err := Replace(exe, []byte("nouveau")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "nouveau" {
		t.Fatalf("contenu : %q", got)
	}
	st, _ := os.Stat(exe)
	if st.Mode().Perm()&0o100 == 0 && os.PathSeparator == '/' {
		t.Fatal("le binaire doit rester exécutable")
	}
}
